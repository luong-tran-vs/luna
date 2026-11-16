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

import { SpotQuestion } from '../../../core/models/games';
import { GamesApiService } from '../../../core/services/games-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import {
  ClockState,
  pauseClock,
  remaining,
  resumeClock,
  startClock,
} from '../shared/game-clock.logic';
import { GameResult, MissedItem } from '../shared/game-result';
import { formatScore } from '../shared/game-score.logic';
import { GameShell } from '../shared/game-shell';
import {
  callClean,
  chooseFix,
  correctedSentence,
  inMistake,
  isMark,
  mistakeText,
  nextSpot,
  outcomeRight,
  rightFix,
  spaceBefore,
  SPOT_LIVES,
  SPOT_MS,
  SpotState,
  spotFinished,
  startSpot,
  tapToken,
  timeOut,
} from './grammar-spot.logic';

type Phase = 'loading' | 'empty' | 'members' | 'error' | 'playing' | 'done';

/** A piece of the sentence card: a word (button), a mark (text) or the fix put in. */
interface Piece {
  kind: 'word' | 'mark' | 'fix';
  text: string;
  /** The token's index; -1 for the fix. */
  index: number;
  space: boolean;
  /** The token is (part of) the mistake, shown red and struck through. */
  struck: boolean;
  /** A right word the learner tapped by mistake. */
  tapped: boolean;
}

/** The status of an API error, 0 when there is none. */
function statusOf(err: unknown): number {
  const s = (err as { status?: unknown } | null)?.status;
  return typeof s === 'number' ? s : 0;
}

/**
 * "Bắn lỗi ngữ pháp" (P5, design/de-xuat-tro-choi-2.gif part 1): 10 sentences, each with one wrong
 * word or none, 20 seconds and 3 lives. Tapping the mistake opens its three fixes in a bottom sheet;
 * every sentence ends with its grammar topic and rule.
 */
@Component({
  selector: 'lu-grammar-spot',
  imports: [GameShell, GameResult, RouterLink, Loading],
  templateUrl: './grammar-spot.html',
  styleUrl: './grammar-spot.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarSpot {
  private readonly api = inject(GamesApiService);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);

  protected readonly phase = signal<Phase>('loading');
  protected readonly state = signal<SpotState>(startSpot([]));
  private readonly clock = signal<ClockState | null>(null);
  private readonly now = signal(0);
  protected readonly best = signal<number | null>(null);
  protected readonly newBest = signal(false);
  protected readonly offline = signal(false);
  protected readonly announce = signal('');

  private readonly sheet = viewChild<ElementRef<HTMLElement>>('sheet');
  private readonly nextButton = viewChild<ElementRef<HTMLElement>>('next');
  private readonly card = viewChild<ElementRef<HTMLElement>>('card');

  protected readonly format = formatScore;
  protected readonly lifeSlots = Array.from({ length: SPOT_LIVES }, (_, i) => i);

  /** The sentence on screen: the current one, kept while its outcome is shown. */
  protected readonly question = computed(() => {
    const s = this.state();
    return s.questions[s.index] ?? null;
  });
  protected readonly left = computed(() => {
    const c = this.clock();
    return c ? remaining(c, this.now()) : SPOT_MS;
  });
  protected readonly progress = computed(() => this.left() / SPOT_MS);
  protected readonly right = computed(() => outcomeRight(this.state().outcome));
  protected readonly mistake = computed(() => {
    const q = this.question();
    return q ? mistakeText(q) : '';
  });
  protected readonly fix = computed(() => {
    const q = this.question();
    return q ? rightFix(q) : '';
  });
  /** Whether "Tiếp" ends the round. */
  protected readonly last = computed(() => spotFinished(nextSpot(this.state())));

  protected readonly pieces = computed<Piece[]>(() => {
    const q = this.question();
    const s = this.state();
    if (!q) {
      return [];
    }
    const reveal = s.stage !== 'ask' && !!q.wrong;
    const fixed = s.outcome === 'fixed';
    const out: Piece[] = [];
    q.tokens.forEach((text, index) => {
      const space = spaceBefore(q.tokens, index);
      const struck = reveal && inMistake(q, index);
      if (fixed && struck) {
        if (index === q.wrong!.from) {
          out.push({ kind: 'fix', text: this.fix(), index: -1, space, struck: false, tapped: false });
        }
        return;
      }
      out.push({
        kind: isMark(text) ? 'mark' : 'word',
        text,
        index,
        space,
        struck,
        tapped: s.stage === 'shown' && s.tapped === index && !inMistake(q, index),
      });
      if (struck && s.stage === 'shown' && index === q.wrong!.to - 1) {
        out.push({ kind: 'fix', text: this.fix(), index: -1, space: true, struck: false, tapped: false });
      }
    });
    return out;
  });

  protected readonly heading = computed(() => {
    switch (this.state().outcome) {
      case 'fixed':
        return `Đúng! +${this.state().gained}`;
      case 'clean':
        return `Câu đúng! +${this.state().gained}`;
      case 'timeout':
        return 'Hết giờ';
      case 'missed':
        return 'Câu này có lỗi';
      case 'false-alarm':
        return 'Câu này không có lỗi';
      default:
        return 'Chưa đúng';
    }
  });

  protected readonly missed = computed<MissedItem[]>(() =>
    this.state().missed.map((q) => ({ text: correctedSentence(q), detail: q.topicVi })),
  );

  private timer: ReturnType<typeof setInterval> | null = null;
  private warned = false;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimer());
    this.load();
  }

  protected load(): void {
    this.phase.set('loading');
    this.stopTimer();
    this.api.grammarRound().subscribe({
      next: (r) => this.start(r.questions),
      error: (err: unknown) => {
        const status = statusOf(err);
        this.phase.set(status === 409 ? 'empty' : status === 403 ? 'members' : 'error');
      },
    });
  }

  private start(questions: SpotQuestion[]): void {
    this.state.set(startSpot(questions));
    this.best.set(null);
    this.newBest.set(false);
    this.offline.set(false);
    this.phase.set('playing');
    this.startSentence();
    this.timer = setInterval(() => this.tick(), 200);
  }

  private startSentence(): void {
    const t = performance.now();
    this.now.set(t);
    this.clock.set(startClock(SPOT_MS, t));
    this.warned = false;
  }

  private tick(): void {
    if (this.state().stage === 'shown') {
      return;
    }
    this.now.set(performance.now());
    const left = this.left();
    if (left <= 5000 && !this.warned) {
      this.warned = true;
      this.announce.set('Còn 5 giây');
    }
    if (left <= 0) {
      this.apply(timeOut(this.state()));
    }
  }

  /** Whether the learner can answer now (playing, not paused). */
  private canAnswer(): boolean {
    return this.phase() === 'playing' && this.clock()?.since !== null;
  }

  protected tap(i: number): void {
    if (!this.canAnswer()) {
      return;
    }
    this.apply(tapToken(this.state(), i, this.left()));
  }

  protected choose(i: number): void {
    if (!this.canAnswer()) {
      return;
    }
    this.apply(chooseFix(this.state(), i, this.left()));
  }

  protected noMistake(): void {
    if (!this.canAnswer()) {
      return;
    }
    this.apply(callClean(this.state(), this.left()));
  }

  /** Sets the new state, tells the screen reader and moves the focus. */
  private apply(next: SpotState): void {
    const before = this.state();
    if (next === before) {
      return;
    }
    this.state.set(next);
    if (next.stage === 'fix') {
      this.announce.set(`Chọn cách sửa “${this.mistake()}”`);
      this.focus(() => this.sheet()?.nativeElement.querySelector<HTMLElement>('button'));
      return;
    }
    if (next.stage === 'shown') {
      this.clock.update((c) => (c ? pauseClock(c, performance.now()) : c));
      const q = this.question()!;
      const fix = q.wrong ? ` Sửa “${this.mistake()}” thành “${this.fix()}”.` : '';
      this.announce.set(`${this.heading()}.${outcomeRight(next.outcome) ? '' : fix} ${q.ruleVi}`);
      this.focus(() => this.nextButton()?.nativeElement);
    }
  }

  private focus(target: () => HTMLElement | null | undefined): void {
    afterNextRender(() => target()?.focus(), { injector: this.injector });
  }

  protected goOn(): void {
    const s = nextSpot(this.state());
    this.state.set(s);
    if (spotFinished(s)) {
      this.end(true);
      return;
    }
    this.announce.set(`Câu ${s.index + 1}`);
    this.startSentence();
    this.focus(() => this.card()?.nativeElement);
  }

  protected pause(): void {
    const c = this.clock();
    if (c && this.state().stage !== 'shown') {
      this.clock.set(pauseClock(c, performance.now()));
    }
  }

  protected resume(): void {
    const c = this.clock();
    if (c && this.state().stage !== 'shown') {
      const t = performance.now();
      this.now.set(t);
      this.clock.set(resumeClock(c, t));
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
    this.stopTimer();
    const s = this.state();
    if (completed) {
      this.phase.set('done');
    }
    this.api
      .finish('grammar', { score: s.score, completed, wrongWords: [], wrongVerbs: [] })
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
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
}
