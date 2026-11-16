import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { GameWord } from '../../../core/models/games';
import { errorStatus, GamesApiService } from '../../../core/services/games-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { GameResult, MissedItem } from '../shared/game-result';
import { formatScore } from '../shared/game-score.logic';
import { GameShell } from '../shared/game-shell';
import {
  currentLevel,
  currentWord,
  dropFell,
  LetterState,
  lettersFinished,
  LIVES,
  neededLetter,
  nextWord,
  startLetters,
  tapLetter,
} from './letter-drop.logic';

/** How long a spelt word stays (its picture and "house ✓") before the next one. */
const SOLVED_PAUSE = 1100;

type Phase = 'loading' | 'empty' | 'error' | 'playing' | 'done';

/**
 * "Chữ rơi" (P3, design/de-xuat-tro-choi.gif part 2): the Vietnamese meaning on top, the word's
 * letters and some distractors falling, the boxes at the bottom filled by tapping in order.
 */
@Component({
  selector: 'lu-letter-drop',
  imports: [GameShell, GameResult, RouterLink, Loading],
  templateUrl: './letter-drop.html',
  styleUrl: './letter-drop.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LetterDrop {
  private readonly api = inject(GamesApiService);
  private readonly router = inject(Router);
  private readonly rand = Math.random;

  protected readonly phase = signal<Phase>('loading');
  protected readonly state = signal<LetterState>(startLetters([], Math.random));
  protected readonly paused = signal(false);
  protected readonly best = signal<number | null>(null);
  protected readonly newBest = signal(false);
  protected readonly offline = signal(false);
  protected readonly announce = signal('');

  protected readonly word = computed(() => currentWord(this.state()));
  protected readonly level = computed(() => currentLevel(this.state()));
  protected readonly hearts = computed(() => {
    const lives = Math.max(0, this.state().lives);
    return '❤'.repeat(lives) + '♡'.repeat(LIVES - lives);
  });
  protected readonly boxes = computed(() => {
    const w = this.word();
    const filled = this.state().filled;
    return w ? [...w.lemma].map((c, i) => (i < filled ? c : '')) : [];
  });
  protected readonly boxesLabel = computed(() => {
    const w = this.word();
    if (!w) {
      return '';
    }
    const done = w.lemma.slice(0, this.state().filled);
    return `Đã ghép ${this.state().filled} trên ${w.lemma.length} chữ${done ? ': ' + done.split('').join(' ') : ''}`;
  });
  /** The visible line under the meaning after a mistake. */
  protected readonly feedback = computed(() => {
    const s = this.state();
    if (s.last === 'wrong') {
      return `Sai chữ, mất 1 mạng. Cần chữ "${neededLetter(s)}"`;
    }
    if (s.last === 'fell') {
      return `Chữ "${neededLetter(s)}" rơi mất, mất 1 mạng`;
    }
    return '';
  });
  protected readonly heading = computed(() => (this.state().lives <= 0 ? 'Hết mạng!' : 'Hết từ!'));
  protected readonly extra = computed(() => `Đạt màn ${this.level()}`);
  protected readonly missed = computed<MissedItem[]>(() =>
    this.state().missed.map((m) => ({ text: m.lemma, detail: m.meaningVi })),
  );
  protected readonly format = formatScore;

  private next: ReturnType<typeof setTimeout> | null = null;
  /** The next word is waiting for the game to resume. */
  private advanceOnResume = false;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimer());
    this.load();
  }

  protected imageUrl(w: GameWord): string {
    return serverUrl(w.imageUrl);
  }

  /** "calc((100% - 3rem) * 0.42)": the drop stays inside the area whatever its x. */
  protected left(x: number): string {
    return `calc((100% - 3rem) * ${x / 100})`;
  }

  protected load(): void {
    this.phase.set('loading');
    this.stopTimer();
    this.api.lettersRound().subscribe({
      next: (r) => this.start(r.words),
      error: (err: unknown) =>
        this.phase.set(errorStatus(err) === 409 ? 'empty' : 'error'),
    });
  }

  private start(words: GameWord[]): void {
    if (words.length === 0) {
      this.phase.set('empty');
      return;
    }
    this.state.set(startLetters(words, this.rand));
    this.paused.set(false);
    this.advanceOnResume = false;
    this.best.set(null);
    this.newBest.set(false);
    this.offline.set(false);
    this.announce.set(`Màn 1. Ghép từ có nghĩa: ${words[0].meaningVi}`);
    this.phase.set('playing');
  }

  protected tap(id: number): void {
    if (this.phase() !== 'playing' || this.paused()) {
      return;
    }
    const before = this.state();
    const after = tapLetter(before, id);
    if (after === before) {
      return;
    }
    this.state.set(after);
    if (after.last === 'wrong') {
      this.announce.set(`Sai. Còn ${after.lives} mạng`);
    } else if (after.last === 'solved') {
      this.announce.set(`${currentWord(after)!.lemma}, đúng. Cộng ${after.gained}`);
      this.next = setTimeout(() => this.advance(), SOLVED_PAUSE);
    } else {
      this.announce.set(`Chữ ${neededLetter(before)}`);
    }
    this.checkEnd();
  }

  protected fell(id: number): void {
    if (this.phase() !== 'playing' || this.paused()) {
      return;
    }
    const before = this.state();
    const after = dropFell(before, id, this.rand);
    this.state.set(after);
    if (after.last === 'fell' && after.lives < before.lives) {
      this.announce.set(`Chữ ${neededLetter(before)} rơi mất. Còn ${after.lives} mạng`);
    }
    this.checkEnd();
  }

  private advance(): void {
    this.next = null;
    if (this.paused()) {
      this.advanceOnResume = true;
      return;
    }
    const before = currentLevel(this.state());
    const s = nextWord(this.state(), this.rand);
    this.state.set(s);
    if (lettersFinished(s)) {
      this.end(true);
      return;
    }
    const level = currentLevel(s);
    const w = currentWord(s)!;
    this.announce.set(`${level > before ? `Màn ${level}. ` : ''}Ghép từ có nghĩa: ${w.meaningVi}`);
  }

  private checkEnd(): void {
    if (lettersFinished(this.state())) {
      this.end(true);
    }
  }

  protected pause(): void {
    this.paused.set(true);
  }

  protected resume(): void {
    this.paused.set(false);
    if (this.advanceOnResume) {
      this.advanceOnResume = false;
      this.advance();
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
    this.stopTimer();
    const s = this.state();
    if (completed) {
      this.phase.set('done');
    }
    this.api
      .finish('letters', {
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

  private stopTimer(): void {
    if (this.next) {
      clearTimeout(this.next);
      this.next = null;
    }
  }
}
