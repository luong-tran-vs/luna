import { DOCUMENT } from '@angular/common';
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

import { serverUrl } from '../../../core/api-url';
import {
  DailyCompare,
  DailyQuestion,
  DailyResultBody,
  DailyResultView,
  DailyView,
  VerbMiss,
} from '../../../core/models/games';
import { errorBody, errorStatus, GamesApiService } from '../../../core/services/games-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';
import { gradeForm } from '../../irregular-verbs/verb-study.logic';
import { ClockState, clockLabel, pauseClock, remaining, resumeClock, startClock } from '../shared/game-clock.logic';
import { formatScore } from '../shared/game-score.logic';
import { GameShell } from '../shared/game-shell';
import {
  DAILY_COUNT,
  DailyMiss,
  dailyAnswers,
  DailyState,
  dayLabel,
  fixedSentence,
  hitsWrong,
  LetterTile,
  letterTiles,
  noErrorRight,
  QUESTION_MS,
  recordAnswer,
  recordTimeout,
  resolveVerbs,
  rightTile,
  startDaily,
  weekCells,
} from './daily.logic';
import { shareText } from './share.logic';

type Phase = 'loading' | 'error' | 'start' | 'playing' | 'sending' | 'result';

/** What the result screen shows: the counted round, or a replay for practice. */
interface Outcome {
  score: number;
  answers: boolean[];
  streak: number;
  compare: DailyCompare;
  practice: boolean;
  /** Something was wrong: it went back to the review queue. */
  missed: boolean;
}

/** The answer given to the current question; null before answering. */
interface Answered {
  right: boolean;
  timedOut: boolean;
}

/**
 * Thử thách hằng ngày (P6, design/de-xuat-tro-choi-2.gif part 2): ten short questions of the other
 * games, 20 seconds each. Only the first round of the day counts (FR-028); a replay is practice.
 */
@Component({
  selector: 'lu-daily-challenge',
  imports: [GameShell, RouterLink, Loading],
  templateUrl: './daily-challenge.html',
  styleUrl: './daily-challenge.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DailyChallenge {
  private readonly api = inject(GamesApiService);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);
  private readonly window = inject(DOCUMENT).defaultView;

  protected readonly phase = signal<Phase>('loading');
  protected readonly view = signal<DailyView | null>(null);
  /** Whether the round being played is a replay (practice only). */
  private readonly practice = signal(false);
  protected readonly state = signal<DailyState>(startDaily());
  protected readonly index = signal(0);
  protected readonly answered = signal<Answered | null>(null);
  protected readonly outcome = signal<Outcome | null>(null);
  protected readonly sendError = signal(false);
  protected readonly copied = signal(false);
  protected readonly showShareText = signal(false);
  protected readonly announce = signal('');

  private readonly clock = signal<ClockState | null>(null);
  private readonly now = signal(0);
  private paused = false;

  // Per question.
  protected readonly chosen = signal<number | null>(null);
  protected readonly v2 = signal('');
  protected readonly v3 = signal('');
  protected readonly placed = signal<LetterTile[]>([]);
  protected readonly tapped = signal<number | null>(null);
  protected readonly option = signal<number | null>(null);
  protected readonly saidNoError = signal(false);

  private readonly v2Input = viewChild<ElementRef<HTMLInputElement>>('v2Input');
  private readonly v3Input = viewChild<ElementRef<HTMLInputElement>>('v3Input');
  private readonly nextBtn = viewChild<ElementRef<HTMLButtonElement>>('nextBtn');

  protected readonly format = formatScore;
  protected readonly count = DAILY_COUNT;

  protected readonly dayText = computed(() => {
    const v = this.view();
    return v ? dayLabel(v.day) : '';
  });
  protected readonly week = computed(() => {
    const v = this.view();
    return v ? weekCells(v.week, v.day) : [];
  });
  protected readonly streak = computed(() => this.outcome()?.streak ?? this.view()?.streak.current ?? 0);

  protected readonly question = computed<DailyQuestion | null>(() => this.view()?.questions[this.index()] ?? null);
  private readonly verbs = computed(() => {
    const v = this.view();
    return v ? resolveVerbs(v.questions, v.seed) : [];
  });
  protected readonly verb = computed<IrregularVerb | null>(() => this.verbs()[this.index()] ?? null);
  protected readonly tiles = computed<LetterTile[]>(() => {
    const v = this.view();
    const q = this.question();
    return v && q?.kind === 'letters' && q.letters ? letterTiles(q.letters.lemma, v.seed, this.index()) : [];
  });
  protected readonly boxes = computed(() => {
    const q = this.question();
    const word = q?.letters?.lemma.toLowerCase() ?? '';
    const placed = this.placed();
    const reveal = this.answered() !== null;
    return word.split('').map((ch, i) => ({ ch: placed[i]?.ch ?? (reveal ? ch : ''), shown: i >= placed.length && reveal }));
  });
  protected readonly v2Right = computed(() => !!this.verb() && gradeForm(this.v2(), this.verb()!.past));
  protected readonly v3Right = computed(() => !!this.verb() && gradeForm(this.v3(), this.verb()!.participle));
  protected readonly fixed = computed(() => {
    const g = this.question()?.grammar;
    return g ? fixedSentence(g) : '';
  });

  protected readonly left = computed(() => {
    const c = this.clock();
    return c ? remaining(c, this.now()) : QUESTION_MS;
  });
  protected readonly timeLabel = computed(() => clockLabel(this.left()));
  protected readonly progress = computed(() => this.left() / QUESTION_MS);

  /** The ten dots of the round. */
  protected readonly dots = computed(() => {
    const answers = this.state().answers;
    return Array.from({ length: DAILY_COUNT }, (_, i) => {
      const a = answers[i];
      const status = a === undefined ? (i === this.index() ? 'current' : 'todo') : a ? 'right' : 'wrong';
      const said = { current: 'đang làm', todo: 'chưa làm', right: 'đúng', wrong: 'sai' }[status];
      return { status, name: `Câu ${i + 1}: ${said}` };
    });
  });

  protected readonly rightCount = computed(() => this.outcome()?.answers.filter(Boolean).length ?? 0);

  protected readonly share = computed(() => {
    const v = this.view();
    const o = this.outcome();
    return v && o ? shareText(v.day, o.answers, o.streak) : '';
  });

  private timer: ReturnType<typeof setInterval> | null = null;
  private pendingBody: DailyResultBody | null = null;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopTimer());
    this.load();
  }

  protected imageUrl(path: string): string {
    return serverUrl(path);
  }

  protected load(): void {
    this.phase.set('loading');
    this.api.daily().subscribe({
      next: (v) => {
        this.view.set(v);
        if (v.done && v.result) {
          this.showCounted(v.result, v.streak.current, v.compare);
        } else {
          this.outcome.set(null);
          this.phase.set('start');
        }
      },
      error: () => this.phase.set('error'),
    });
  }

  private showCounted(r: DailyResultView, streak: number, compare: DailyCompare, missed?: boolean): void {
    const answers = r.answers.map((a) => a.right);
    this.outcome.set({
      score: r.score,
      answers,
      streak,
      compare,
      practice: false,
      // Wrong grammar sentences do not go to the review queue.
      missed: missed ?? r.answers.some((a) => !a.right && a.kind !== 'grammar'),
    });
    this.copied.set(false);
    this.showShareText.set(false);
    this.phase.set('result');
  }

  /** "Bắt đầu" or "Chơi lại để luyện". */
  protected begin(): void {
    const v = this.view();
    if (!v || v.questions.length === 0) {
      return;
    }
    this.practice.set(v.done);
    this.state.set(startDaily());
    this.index.set(0);
    this.sendError.set(false);
    this.phase.set('playing');
    this.startQuestion();
  }

  private startQuestion(): void {
    this.answered.set(null);
    this.chosen.set(null);
    this.v2.set('');
    this.v3.set('');
    this.placed.set([]);
    this.tapped.set(null);
    this.option.set(null);
    this.saidNoError.set(false);
    const t = performance.now();
    this.now.set(t);
    this.clock.set(this.paused ? pauseClock(startClock(QUESTION_MS, t), t) : startClock(QUESTION_MS, t));
    this.stopTimer();
    this.timer = setInterval(() => this.tick(), 200);
    if (this.question()?.kind === 'verb') {
      this.focusLater(() => this.v2Input());
    }
  }

  private tick(): void {
    this.now.set(performance.now());
    if (this.answered() === null && this.left() <= 0) {
      this.answer(false, this.currentMiss(), true);
    }
  }

  /** What the current question brings back to the review queue when it is wrong. */
  private currentMiss(): DailyMiss {
    const q = this.question();
    switch (q?.kind) {
      case 'picture':
        return { word: q.picture?.lemma };
      case 'letters':
        return { word: q.letters?.lemma };
      case 'verb': {
        const v = this.verb();
        if (!v) {
          return {};
        }
        const verbs: VerbMiss[] = [];
        if (!this.v2Right()) {
          verbs.push({ base: v.base, form: 'past', typed: this.v2().trim() });
        }
        if (!this.v3Right()) {
          verbs.push({ base: v.base, form: 'participle', typed: this.v3().trim() });
        }
        return { verbs };
      }
      default:
        return {};
    }
  }

  private answer(right: boolean, miss: DailyMiss, timedOut = false): void {
    if (this.answered() !== null || this.phase() !== 'playing') {
      return;
    }
    const t = performance.now();
    this.now.set(t);
    const c = this.clock();
    const left = c ? remaining(c, t) : 0;
    if (c) {
      this.clock.set(pauseClock(c, t));
    }
    this.stopTimer();
    this.state.update((s) => (timedOut ? recordTimeout(s, miss) : recordAnswer(s, right, left, miss)));
    this.answered.set({ right, timedOut });
    this.announce.set(timedOut ? `Hết giờ. ${this.rightAnswerText()}` : right ? 'Đúng' : `Sai. ${this.rightAnswerText()}`);
    this.focusLater(() => this.nextBtn());
  }

  private rightAnswerText(): string {
    const q = this.question();
    switch (q?.kind) {
      case 'picture':
        return q.picture ? `Đáp án: ${q.picture.options[q.picture.answer]}` : '';
      case 'letters':
        return q.letters ? `Đáp án: ${q.letters.lemma}` : '';
      case 'verb': {
        const v = this.verb();
        return v ? `Đáp án: ${v.past}, ${v.participle}` : '';
      }
      case 'grammar':
        return this.fixed() ? `Câu đúng: ${this.fixed()}` : '';
      default:
        return '';
    }
  }

  // --- picture ---

  protected choose(i: number): void {
    const p = this.question()?.picture;
    if (!p || this.answered()) {
      return;
    }
    this.chosen.set(i);
    this.answer(i === p.answer, this.currentMiss());
  }

  // --- verb ---

  protected typeV2(e: Event): void {
    this.v2.set((e.target as HTMLInputElement).value);
  }

  protected typeV3(e: Event): void {
    this.v3.set((e.target as HTMLInputElement).value);
  }

  protected v2Enter(e: Event): void {
    e.preventDefault();
    this.v3Input()?.nativeElement.focus();
  }

  protected checkVerb(e?: Event): void {
    e?.preventDefault();
    if (this.answered() || !this.verb()) {
      return;
    }
    this.answer(this.v2Right() && this.v3Right(), this.currentMiss());
  }

  // --- letters ---

  protected isPlaced(t: LetterTile): boolean {
    return this.placed().some((p) => p.id === t.id);
  }

  protected tap(t: LetterTile): void {
    const word = this.question()?.letters?.lemma ?? '';
    if (!word || this.answered() || this.isPlaced(t)) {
      return;
    }
    if (!rightTile(word, this.placed().length, t)) {
      this.answer(false, this.currentMiss());
      return;
    }
    this.placed.update((p) => [...p, t]);
    if (this.placed().length === word.length) {
      this.answer(true, {});
    }
  }

  // --- grammar ---

  protected isWrongToken(i: number): boolean {
    const g = this.question()?.grammar;
    return !!g && this.answered() !== null && hitsWrong(g, i);
  }

  protected tapToken(i: number): void {
    const g = this.question()?.grammar;
    if (!g || this.answered() || this.tapped() !== null) {
      return;
    }
    this.tapped.set(i);
    if (!hitsWrong(g, i)) {
      this.answer(false, {});
    }
  }

  protected pickOption(i: number): void {
    const g = this.question()?.grammar;
    if (!g || this.answered()) {
      return;
    }
    this.option.set(i);
    this.answer(i === g.answer, {});
  }

  protected noError(): void {
    const g = this.question()?.grammar;
    if (!g || this.answered() || this.tapped() !== null) {
      return;
    }
    this.saidNoError.set(true);
    this.answer(noErrorRight(g), {});
  }

  // --- flow ---

  protected next(): void {
    if (this.answered() === null) {
      return;
    }
    if (this.index() + 1 >= Math.min(DAILY_COUNT, this.view()?.questions.length ?? 0)) {
      this.finish();
      return;
    }
    this.index.update((i) => i + 1);
    this.startQuestion();
  }

  private finish(): void {
    this.stopTimer();
    this.clock.set(null);
    const v = this.view();
    if (!v) {
      return;
    }
    const s = this.state();
    const missed = s.wrongWords.length > 0 || s.wrongVerbs.length > 0;
    if (this.practice()) {
      this.outcome.set({
        score: s.score,
        answers: s.answers,
        streak: v.streak.current,
        compare: v.compare,
        practice: true,
        missed,
      });
      this.phase.set('result');
      if (s.wrongWords.length || s.wrongVerbs.length) {
        this.api.dailyPractice(s.wrongWords, s.wrongVerbs).subscribe({ error: () => undefined });
      }
      return;
    }
    this.pendingBody = {
      day: v.day,
      score: s.score,
      answers: dailyAnswers(v.questions, s.answers),
      wrongWords: s.wrongWords,
      wrongVerbs: s.wrongVerbs,
    };
    this.send();
  }

  /** Sends the counted round (again after a failure). */
  protected send(): void {
    const body = this.pendingBody;
    const v = this.view();
    if (!body || !v) {
      return;
    }
    this.phase.set('sending');
    this.sendError.set(false);
    this.api.dailyResult(body).subscribe({
      next: (r) => {
        const result: DailyResultView = { score: body.score, answers: body.answers, finishedAt: '' };
        this.view.set({ ...v, done: true, result, streak: r.streak, compare: r.compare, week: this.playedToday(v) });
        this.pendingBody = null;
        this.showCounted(result, r.streak.current, r.compare, body.wrongWords.length + body.wrongVerbs.length > 0);
        this.announce.set(`Xong thử thách: ${body.answers.filter((a) => a.right).length}/${body.answers.length} câu đúng`);
      },
      error: (err: unknown) => {
        if (errorStatus(err) === 409) {
          const stored = (errorBody(err) as { result?: DailyResultView } | null)?.result;
          const result = stored ?? { score: body.score, answers: body.answers, finishedAt: '' };
          this.view.set({ ...v, done: true, result });
          this.pendingBody = null;
          this.showCounted(result, v.streak.current, v.compare);
          return;
        }
        this.sendError.set(true);
      },
    });
  }

  private playedToday(v: DailyView): boolean[] {
    const week = [...v.week];
    const today = this.week().findIndex((c) => c.today);
    if (today >= 0) {
      week[today] = true;
    }
    return week;
  }

  protected async copyShare(): Promise<void> {
    const text = this.share();
    try {
      const clip = this.window?.navigator?.clipboard;
      if (!clip) {
        throw new Error('no clipboard');
      }
      await clip.writeText(text);
      this.copied.set(true);
      this.announce.set('Đã sao chép kết quả');
    } catch {
      this.showShareText.set(true);
    }
  }

  protected compareText(n: number | null | undefined): string {
    return n === null || n === undefined ? '—' : formatScore(n);
  }

  protected pause(): void {
    this.paused = true;
    const c = this.clock();
    if (c && this.answered() === null) {
      this.clock.set(pauseClock(c, performance.now()));
    }
  }

  protected resume(): void {
    this.paused = false;
    const c = this.clock();
    if (c && this.answered() === null) {
      const t = performance.now();
      this.now.set(t);
      this.clock.set(resumeClock(c, t));
    }
  }

  /** Leaving mid-round sends nothing. */
  protected quit(): void {
    this.stopTimer();
    void this.router.navigateByUrl('/games');
  }

  private focusLater(target: () => ElementRef<HTMLElement> | undefined): void {
    afterNextRender(() => target()?.nativeElement.focus(), { injector: this.injector });
  }

  private stopTimer(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
}
