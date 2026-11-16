import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { PictureQuestion } from '../../../core/models/games';
import { errorStatus, GamesApiService } from '../../../core/services/games-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import {
  ClockState,
  clockLabel,
  pauseClock,
  remaining,
  resumeClock,
  startClock,
} from '../shared/game-clock.logic';
import { GameResult, MissedItem } from '../shared/game-result';
import { formatScore } from '../shared/game-score.logic';
import { GameShell } from '../shared/game-shell';
import {
  answerPicture,
  currentPicture,
  nextPicture,
  PictureState,
  pictureFinished,
  startPicture,
} from './picture-quiz.logic';

/** A round lasts 60 seconds (FR-010). */
const ROUND_MS = 60_000;
/** How long a right / wrong answer stays before the next picture. */
const RIGHT_PAUSE = 450;
const WRONG_PAUSE = 1000;

type Phase = 'loading' | 'empty' | 'error' | 'playing' | 'done';

/**
 * "Ảnh nhanh tay" (P1, design/de-xuat-tro-choi.gif part 1): 60 seconds of pictures, four words
 * each; right answers raise the combo, a wrong one shows the right word for a second.
 */
@Component({
  selector: 'lu-picture-quiz',
  imports: [GameShell, GameResult, RouterLink, Loading],
  templateUrl: './picture-quiz.html',
  styleUrl: './picture-quiz.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PictureQuiz {
  private readonly api = inject(GamesApiService);
  private readonly router = inject(Router);

  protected readonly phase = signal<Phase>('loading');
  protected readonly state = signal<PictureState>(startPicture([]));
  private readonly clock = signal<ClockState | null>(null);
  private readonly now = signal(0);
  protected readonly best = signal<number | null>(null);
  protected readonly newBest = signal(false);
  protected readonly offline = signal(false);
  /** Why the round ended: the time, or every picture was asked. */
  protected readonly endHeading = signal('Hết giờ!');
  /** What the screen reader is told after an answer or near the end. */
  protected readonly announce = signal('');

  protected readonly question = computed(() => currentPicture(this.state()));
  protected readonly left = computed(() => {
    const c = this.clock();
    return c ? remaining(c, this.now()) : ROUND_MS;
  });
  protected readonly timeLabel = computed(() => clockLabel(this.left()));
  protected readonly progress = computed(() => this.left() / ROUND_MS);
  protected readonly missed = computed<MissedItem[]>(() =>
    this.state().missed.map((m) => ({ text: m.lemma, detail: m.meaningVi })),
  );
  protected readonly format = formatScore;

  private timer: ReturnType<typeof setInterval> | null = null;
  private next: ReturnType<typeof setTimeout> | null = null;
  private readonly warned = new Set<number>();

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimers());
    this.load();
  }

  protected imageUrl(q: PictureQuestion): string {
    return serverUrl(q.imageUrl);
  }

  protected load(): void {
    this.phase.set('loading');
    this.stopTimers();
    this.api.pictureRound().subscribe({
      next: (r) => this.start(r.questions),
      error: (err: unknown) =>
        this.phase.set(errorStatus(err) === 409 ? 'empty' : 'error'),
    });
  }

  private start(questions: PictureQuestion[]): void {
    this.state.set(startPicture(questions));
    this.best.set(null);
    this.newBest.set(false);
    this.offline.set(false);
    this.endHeading.set('Hết giờ!');
    this.warned.clear();
    const t = performance.now();
    this.now.set(t);
    this.clock.set(startClock(ROUND_MS, t));
    this.phase.set('playing');
    this.timer = setInterval(() => this.tick(), 200);
  }

  private tick(): void {
    this.now.set(performance.now());
    const left = this.left();
    for (const mark of [30_000, 10_000]) {
      if (left <= mark && !this.warned.has(mark)) {
        this.warned.add(mark);
        this.announce.set(`Còn ${mark / 1000} giây`);
      }
    }
    if (left <= 0) {
      this.end(true);
    }
  }

  protected choose(i: number): void {
    const before = this.state();
    if (this.phase() !== 'playing' || before.chosen !== null || this.clock()?.since === null) {
      return;
    }
    const after = answerPicture(before, i);
    this.state.set(after);
    const q = currentPicture(before)!;
    const right = i === q.answer;
    this.announce.set(right ? `Đúng, cộng ${after.gained}` : `Sai. Đáp án: ${q.options[q.answer]}`);
    this.next = setTimeout(() => this.advance(), right ? RIGHT_PAUSE : WRONG_PAUSE);
  }

  private advance(): void {
    this.next = null;
    const s = nextPicture(this.state());
    this.state.set(s);
    if (pictureFinished(s)) {
      this.endHeading.set('Xong lượt!');
      this.end(true);
    }
  }

  protected pause(): void {
    const c = this.clock();
    if (c) {
      this.clock.set(pauseClock(c, performance.now()));
    }
  }

  protected resume(): void {
    const c = this.clock();
    if (c) {
      const t = performance.now();
      this.now.set(t);
      this.clock.set(resumeClock(c, t));
    }
  }

  /** Leaving: a round being played is sent as not completed (its misses still count). */
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
    this.stopTimers();
    const s = this.state();
    if (completed) {
      this.phase.set('done');
    }
    this.api
      .finish('picture', {
        score: s.score,
        completed,
        wrongWords: s.missed.map((m) => m.lemma),
        wrongVerbs: [],
      })
      .subscribe({
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

  private stopTimers(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    if (this.next) {
      clearTimeout(this.next);
      this.next = null;
    }
  }
}
