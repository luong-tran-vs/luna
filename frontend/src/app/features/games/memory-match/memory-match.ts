import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { GameWord } from '../../../core/models/games';
import { errorStatus, GamesApiService } from '../../../core/services/games-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { GameResult } from '../shared/game-result';
import { GameShell } from '../shared/game-shell';
import {
  dealCards,
  elapsedLabel,
  flipCard,
  hideMismatch,
  isUp,
  MemoryCard,
  MemoryState,
  memoryFinished,
  memoryScore,
  pairsFound,
  startMemory,
} from './memory.logic';

/** How long a mismatched pair stays up before it turns back. */
const MISMATCH_MS = 1000;

type Phase = 'loading' | 'empty' | 'forbidden' | 'error' | 'playing' | 'done';

/**
 * "Lật thẻ ghi nhớ" (P4, design/de-xuat-tro-choi.gif part 4): 12 face-down cards, 6 pairs of an
 * English word and its picture (or meaning). The time counts up; fewer moves and less time score
 * higher. Opened from a finished lesson (?lessonId=…) it uses that lesson's words (FR-020).
 */
@Component({
  selector: 'lu-memory-match',
  imports: [GameShell, GameResult, RouterLink, Loading],
  templateUrl: './memory-match.html',
  styleUrl: './memory-match.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MemoryMatch {
  private readonly api = inject(GamesApiService);
  private readonly router = inject(Router);
  private readonly lessonId = inject(ActivatedRoute).snapshot.queryParamMap.get('lessonId');

  protected readonly phase = signal<Phase>('loading');
  protected readonly state = signal<MemoryState>(startMemory([]));
  /** Time played before the current run, ms; since is when the run started (null while paused). */
  private readonly used = signal(0);
  private readonly since = signal<number | null>(null);
  private readonly now = signal(0);
  protected readonly best = signal<number | null>(null);
  protected readonly newBest = signal(false);
  protected readonly offline = signal(false);
  protected readonly score = signal(0);
  protected readonly announce = signal('');

  protected readonly elapsed = computed(() => {
    const since = this.since();
    return this.used() + (since === null ? 0 : this.now() - since);
  });
  protected readonly timeLabel = computed(() => elapsedLabel(this.elapsed()));
  protected readonly pairs = computed(() => pairsFound(this.state()));
  protected readonly total = computed(() => this.state().cards.length / 2);
  protected readonly extra = computed(() => `${this.state().moves} lượt lật · ${this.timeLabel()}`);

  private timer: ReturnType<typeof setInterval> | null = null;
  private turnBack: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimers());
    this.load();
  }

  protected load(): void {
    this.phase.set('loading');
    this.stopTimers();
    this.api.memoryRound(this.lessonId).subscribe({
      next: (r) => this.start(r.pairs),
      error: (err: unknown) => {
        const status = errorStatus(err);
        this.phase.set(status === 409 ? 'empty' : status === 403 ? 'forbidden' : 'error');
      },
    });
  }

  private start(pairs: GameWord[]): void {
    this.state.set(startMemory(dealCards(pairs)));
    this.best.set(null);
    this.newBest.set(false);
    this.offline.set(false);
    this.score.set(0);
    this.announce.set('');
    const t = performance.now();
    this.used.set(0);
    this.now.set(t);
    this.since.set(t);
    this.phase.set('playing');
    this.timer = setInterval(() => this.now.set(performance.now()), 250);
  }

  protected up(i: number): boolean {
    return isUp(this.state(), i);
  }

  protected wrong(i: number): boolean {
    const s = this.state();
    return s.mismatch && s.flipped.includes(i);
  }

  protected imageUrl(c: MemoryCard): string {
    return serverUrl(c.imageUrl);
  }

  protected label(c: MemoryCard, i: number): string {
    const s = this.state();
    if (!isUp(s, i)) {
      return `Thẻ úp ${i + 1}`;
    }
    const face = c.face === 'image' ? 'ảnh của từ' : c.text;
    return `Thẻ: ${face}${s.matched[i] ? ', đã khớp' : ''}`;
  }

  protected flip(i: number): void {
    if (this.phase() !== 'playing' || this.since() === null) {
      return;
    }
    const before = this.state();
    const after = flipCard(before, i);
    if (after === before) {
      return;
    }
    this.clearTurnBack();
    this.state.set(after);
    if (after.moves > before.moves) {
      this.announce.set(after.mismatch ? 'Không khớp' : 'Khớp');
    }
    if (after.mismatch) {
      this.turnBack = setTimeout(() => {
        this.turnBack = null;
        this.state.update(hideMismatch);
      }, MISMATCH_MS);
    }
    if (memoryFinished(after)) {
      this.end(true);
    }
  }

  protected pause(): void {
    const since = this.since();
    if (since !== null) {
      const t = performance.now();
      this.used.update((u) => u + (t - since));
      this.since.set(null);
      this.now.set(t);
    }
  }

  protected resume(): void {
    if (this.phase() === 'playing' && this.since() === null) {
      const t = performance.now();
      this.now.set(t);
      this.since.set(t);
    }
  }

  /** Leaving: a round being played is sent as not completed. */
  protected quit(): void {
    if (this.phase() === 'playing') {
      this.end(false);
    }
    void this.router.navigateByUrl('/games');
  }

  private end(completed: boolean): void {
    if (this.phase() !== 'playing') {
      return;
    }
    this.pause();
    this.stopTimers();
    const s = this.state();
    const score = completed ? memoryScore(s.moves, this.elapsed(), this.total()) : 0;
    this.score.set(score);
    if (completed) {
      this.phase.set('done');
      this.announce.set(`Xong! ${this.extra()}`);
    }
    this.api.finish('memory', { score, completed, wrongWords: [], wrongVerbs: [] }).subscribe({
      next: (r) => {
        if (r) {
          this.best.set(r.best);
          this.newBest.set(r.newBest);
        } else {
          this.offline.set(true);
        }
      },
      error: () => undefined,
    });
  }

  private clearTurnBack(): void {
    if (this.turnBack) {
      clearTimeout(this.turnBack);
      this.turnBack = null;
    }
  }

  private stopTimers(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    this.clearTurnBack();
  }
}
