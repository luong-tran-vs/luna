import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  Injector,
  signal,
  viewChild,
} from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { forkJoin, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';

import { VerbProgress } from '../../../core/models/verbs';
import { GamesApiService } from '../../../core/services/games-api.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';
import { VerbPictures } from '../../irregular-verbs/verb-pictures.service';
import { GameResult, MissedItem } from '../shared/game-result';
import { GameShell } from '../shared/game-shell';
import {
  answerDuel,
  bothRight,
  currentVerb,
  DUEL_LIVES,
  DUEL_MAX_SCORE,
  duelAnswer,
  duelOrder,
  duelOver,
  DuelState,
  nextDuel,
  startDuel,
} from './verb-duel.logic';

/** How long a right / wrong answer stays before the next verb. */
const RIGHT_PAUSE = 700;
const WRONG_PAUSE = 1800;
/** Wait for the phone keyboard to open before scrolling the box into view. */
const KEYBOARD_DELAY = 300;

type Phase = 'loading' | 'error' | 'playing' | 'done';

/**
 * "Đấu động từ" (P2, design/de-xuat-tro-choi.gif part 3): type V2 and V3 of each verb; a wrong
 * form costs one of three lives and shows the answer. The score is the verbs with both forms right.
 */
@Component({
  selector: 'lu-verb-duel',
  imports: [GameShell, GameResult, RouterLink, Loading],
  templateUrl: './verb-duel.html',
  styleUrl: './verb-duel.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbDuel {
  private readonly api = inject(GamesApiService);
  private readonly verbsApi = inject(VerbsApiService);
  private readonly pictures = inject(VerbPictures);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);

  private readonly pastBox = viewChild<ElementRef<HTMLInputElement>>('pastBox');
  private readonly participleBox = viewChild<ElementRef<HTMLInputElement>>('participleBox');

  protected readonly phase = signal<Phase>('loading');
  protected readonly state = signal<DuelState>(startDuel([]));
  protected readonly typedPast = signal('');
  protected readonly typedParticiple = signal('');
  /** The best before the round (for the 🏆 pill), then the best after it. */
  protected readonly best = signal<number | null>(null);
  protected readonly newBest = signal(false);
  protected readonly offline = signal(false);
  protected readonly announce = signal('');

  protected readonly verb = computed(() => currentVerb(this.state()));
  protected readonly picture = computed(() => {
    const v = this.verb();
    return v ? this.pictures.url(v.base) : '';
  });
  protected readonly checked = computed(() => this.state().checked);
  protected readonly allRight = computed(() => {
    const c = this.checked();
    return c !== null && bothRight(c);
  });
  protected readonly hearts = computed(() =>
    Array.from({ length: DUEL_LIVES }, (_, i) => i < this.state().lives),
  );
  protected readonly missed = computed<MissedItem[]>(() =>
    this.state().missed.map((v) => ({ text: v.base, detail: duelAnswer(v) })),
  );
  protected readonly answerText = duelAnswer;

  private progress: ReadonlyMap<string, VerbProgress> = new Map();
  private next: ReturnType<typeof setTimeout> | null = null;
  private scroll: ReturnType<typeof setTimeout> | null = null;
  private paused = false;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimers());
    this.pictures.load();
    this.load();
  }

  protected load(): void {
    this.phase.set('loading');
    this.stopTimers();
    // A learner without progress (or when it fails) still plays the common verbs.
    const progress = this.verbsApi.progress().pipe(
      map((r) => r.verbs),
      catchError(() => of([] as VerbProgress[])),
    );
    const best = this.api.list().pipe(
      map((l) => l.games.find((g) => g.id === 'verbs')?.best ?? null),
      catchError(() => of(null)),
    );
    forkJoin([progress, best]).subscribe({
      next: ([verbs, b]) => {
        this.progress = new Map(verbs.map((v) => [v.base, v]));
        this.start(b);
      },
      error: () => this.phase.set('error'),
    });
  }

  private order(): IrregularVerb[] {
    return duelOrder(this.progress, new Date(), Math.floor(Math.random() * 1e9));
  }

  private start(best: number | null): void {
    const queue = this.order();
    if (!queue.length) {
      this.phase.set('error');
      return;
    }
    this.state.set(startDuel(queue));
    this.best.set(best);
    this.newBest.set(false);
    this.offline.set(false);
    this.paused = false;
    this.clearBoxes();
    this.phase.set('playing');
    this.focusPast();
  }

  protected onPastEnter(event: Event): void {
    event.preventDefault();
    if (this.checked()) {
      this.advance();
      return;
    }
    this.participleBox()?.nativeElement.focus();
  }

  protected onParticipleEnter(event: Event): void {
    event.preventDefault();
    if (this.checked()) {
      this.advance();
      return;
    }
    if (!this.typedPast().trim()) {
      this.pastBox()?.nativeElement.focus();
      return;
    }
    this.check();
  }

  protected check(): void {
    const before = this.state();
    if (this.phase() !== 'playing' || before.checked !== null) {
      return;
    }
    const after = answerDuel(before, this.typedPast(), this.typedParticiple());
    this.state.set(after);
    const v = currentVerb(before)!;
    if (bothRight(after.checked!)) {
      this.announce.set(`Đúng. Chuỗi ${after.streak}`);
    } else {
      const lives = after.lives > 0 ? `Còn ${after.lives} mạng` : 'Hết mạng';
      this.announce.set(`Sai. Đáp án: ${duelAnswer(v)}. ${lives}`);
    }
    this.arm();
  }

  /** Waits before the next verb (not while paused). */
  private arm(): void {
    if (this.next) {
      clearTimeout(this.next);
    }
    const c = this.checked();
    if (!c || this.paused) {
      return;
    }
    this.next = setTimeout(() => this.advance(), bothRight(c) ? RIGHT_PAUSE : WRONG_PAUSE);
  }

  protected advance(): void {
    if (this.next) {
      clearTimeout(this.next);
      this.next = null;
    }
    const s = this.state();
    if (this.phase() !== 'playing' || s.checked === null) {
      return;
    }
    if (duelOver(s)) {
      this.end(true);
      return;
    }
    const more = s.index + 1 >= s.queue.length ? this.order() : [];
    this.state.set(nextDuel(s, more));
    this.clearBoxes();
    this.focusPast();
  }

  protected setPast(event: Event): void {
    this.typedPast.set((event.target as HTMLInputElement).value);
  }

  protected setParticiple(event: Event): void {
    this.typedParticiple.set((event.target as HTMLInputElement).value);
  }

  /** Keeps the focused box above the phone keyboard. */
  protected reveal(event: FocusEvent): void {
    const el = event.target as HTMLElement;
    if (this.scroll) {
      clearTimeout(this.scroll);
    }
    this.scroll = setTimeout(() => {
      this.scroll = null;
      const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
      el.scrollIntoView?.({ block: 'center', behavior: reduce ? 'auto' : 'smooth' });
    }, KEYBOARD_DELAY);
  }

  protected pause(): void {
    this.paused = true;
    if (this.next) {
      clearTimeout(this.next);
      this.next = null;
    }
  }

  protected resume(): void {
    this.paused = false;
    this.arm();
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
      this.best.set(null);
      this.phase.set('done');
    }
    this.api
      .finish('verbs', {
        score: Math.min(s.score, DUEL_MAX_SCORE),
        completed,
        wrongWords: [],
        wrongVerbs: s.misses,
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

  private clearBoxes(): void {
    this.typedPast.set('');
    this.typedParticiple.set('');
  }

  private focusPast(): void {
    afterNextRender(() => this.pastBox()?.nativeElement.focus(), { injector: this.injector });
  }

  private stopTimers(): void {
    if (this.next) {
      clearTimeout(this.next);
      this.next = null;
    }
    if (this.scroll) {
      clearTimeout(this.scroll);
      this.scroll = null;
    }
  }
}
