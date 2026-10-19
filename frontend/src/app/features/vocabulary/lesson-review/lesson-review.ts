import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  Injector,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, forkJoin, map, of } from 'rxjs';

import { PracticeView } from '../../../core/models/practice';
import { DueCard, LessonResult, LessonVocabulary, VocabItem } from '../../../core/models/vocab';
import { FocusMode } from '../../../core/services/focus-mode.service';
import { SpeechService } from '../../../core/services/speech.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { Asker } from '../../../shared/components/asker/asker';
import { ChoiceList } from '../../../shared/components/choice-list/choice-list';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { SpeakButton } from '../../../shared/directives/speak-button';
import { intervalLabel } from '../../../shared/utils/interval-label';
import { ReadingApiService } from '../../lesson/reading-api.service';
import { settleReviewOffer } from '../../lesson/review-offer';
import {
  Answer,
  Bank,
  buildBank,
  checkTiles,
  daysAgo,
  dueWords,
  minutesFor,
  missedWords,
  Part,
  PART_NAMES,
  partCount,
  partOf,
  PARTS,
  Question,
  questionsWithIds,
  resultAfter,
  RoundKind,
  roundOf,
  scores,
} from './lesson-review-logic';

interface LessonData {
  title: string;
  bank: Bank;
  /** What the questions are made from, to make others for "Ôn lại cả bài". */
  items: VocabItem[];
  due: DueCard[];
  practice: PracticeView | null;
}

/** How the question on screen went: right, wrong, or skipped ("Quên" / "Bỏ qua"). */
type Verdict = 'correct' | 'wrong' | 'skipped';

/** One part of the progress bar: its share of the round and how much of it is done. */
interface Segment {
  part: Part;
  size: number;
  done: number;
}

/**
 * Reviewing one learned lesson (design/on-tap-theo-bai.png): what the round holds, then its words
 * (pick the meaning), lines of its dialogue (pick the missing word) and sentences to translate
 * (build them from tiles), then the score of each part and the questions got wrong. A due card of
 * the lesson is rated by the answer (right: Good, else Again); other words got wrong come back for
 * review (practice misses). The round takes the whole screen, like a lesson.
 */
@Component({
  selector: 'lu-lesson-review',
  imports: [Asker, ChoiceList, Icon, Loading, RouterLink, SpeakButton],
  templateUrl: './lesson-review.html',
  styleUrl: './lesson-review.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(keydown)': 'onKeydown($event)' },
})
export class LessonReview {
  private readonly reading = inject(ReadingApiService);
  private readonly api = inject(VocabApiService);
  private readonly speech = inject(SpeechService);
  private readonly focusMode = inject(FocusMode);
  private readonly injector = inject(Injector);
  private readonly route = inject(ActivatedRoute);
  protected readonly lessonId = this.route.snapshot.paramMap.get('id') ?? '';
  /**
   * The new lesson that offered this review (?next=): leaving or finishing the review goes on to
   * it instead of back to Ôn tập.
   */
  protected readonly nextLesson = this.route.snapshot.queryParamMap.get('next');
  protected readonly exitLink = this.nextLesson
    ? ['/lessons', this.nextLesson]
    : ['/vocabulary/review'];

  protected readonly partNames = PART_NAMES;
  protected readonly data = signal<LessonData | null>(null);
  protected readonly loadError = signal(false);
  protected readonly stage = signal<'intro' | 'quiz' | 'result'>('intro');
  /** The parts the learner keeps for the round. */
  protected readonly parts = signal<Part[]>([...PARTS]);

  protected readonly questions = signal<Question[]>([]);
  protected readonly index = signal(0);
  protected readonly answers = signal<Answer[]>([]);
  protected readonly picked = signal<number | null>(null);
  /** Translation: the tiles chosen, by index into the question's tiles. */
  protected readonly chosen = signal<number[]>([]);
  protected readonly verdict = signal<Verdict | null>(null);
  /** The latest round's score, kept on the server; null before the first round. */
  protected readonly lastResult = signal<LessonResult | null>(null);
  protected readonly roundKind = signal<RoundKind>('full');
  protected readonly saveError = signal(false);
  /** The questions got wrong last time that the lesson still has. */
  protected readonly lastWrong = computed(() => {
    const d = this.data();
    const r = this.lastResult();
    return d && r ? questionsWithIds(d.bank, r.wrong) : [];
  });

  protected readonly available = computed(() => {
    const d = this.data();
    return d ? PARTS.filter((p) => partCount(d.bank, p) > 0) : [];
  });
  protected readonly roundSize = computed(() => {
    const d = this.data();
    return d ? roundOf(d.bank, this.parts()).length : 0;
  });
  protected readonly minutes = computed(() => minutesFor(this.roundSize()));
  /** The words of the lesson due now: asked in every round, Từ vựng chosen or not. */
  protected readonly dueCount = computed(() => {
    const d = this.data();
    return d ? dueWords(d.bank) : 0;
  });
  protected readonly question = computed<Question | null>(
    () => this.questions()[this.index()] ?? null,
  );
  protected readonly part = computed(() => {
    const q = this.question();
    return q ? partOf(q) : null;
  });
  /** "2/4" within the part on screen. */
  protected readonly partPosition = computed(() => {
    const q = this.question();
    if (!q) {
      return '';
    }
    const of = this.questions().filter((x) => x.kind === q.kind);
    return `${of.indexOf(q) + 1}/${of.length}`;
  });
  protected readonly segments = computed<Segment[]>(() => {
    const qs = this.questions();
    const done = this.index() + (this.verdict() ? 1 : 0);
    return PARTS.map((part) => {
      const at = qs.findIndex((q) => q.kind === part);
      const size = qs.filter((q) => q.kind === part).length;
      return { part, size, done: at < 0 ? 0 : Math.min(size, Math.max(0, done - at)) };
    }).filter((s) => s.size > 0);
  });
  protected readonly canSubmit = computed(() => {
    const q = this.question();
    return q?.kind === 'translate' ? this.chosen().length > 0 : this.picked() !== null;
  });
  /** Translation: the tiles in the answer area. */
  protected readonly built = computed(() => {
    const q = this.question();
    return q?.kind === 'translate' ? this.chosen().map((i) => q.tiles[i]) : [];
  });
  /** Fill-in: the word in the gap, the one picked or, once answered, the right one. */
  protected readonly gapText = computed(() => {
    const q = this.question();
    if (q?.kind !== 'fill') {
      return '';
    }
    const p = this.picked();
    return this.verdict() ? q.answer : p === null ? '' : q.choices.options[p];
  });
  protected readonly scores = computed(() => scores(this.answers()));
  protected readonly correct = computed(() => this.answers().filter((a) => a.correct).length);
  protected readonly wrong = computed(() => this.answers().filter((a) => !a.correct));

  private readonly nextButton = viewChild<ElementRef<HTMLButtonElement>>('nextButton');
  private readonly submitButton = viewChild<ElementRef<HTMLButtonElement>>('submitButton');

  constructor() {
    effect(() => this.focusMode.on.set(this.stage() === 'quiz'));
    inject(DestroyRef).onDestroy(() => this.focusMode.on.set(false));
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.data.set(null);
    const id = this.lessonId;
    const empty: LessonVocabulary = { available: false, items: [] };
    forkJoin({
      lesson: this.reading.getLesson(id),
      vocabulary: this.reading.vocabulary(id).pipe(catchError(() => of(empty))),
      practice: this.reading.practice(id).pipe(catchError(() => of<PracticeView | null>(null))),
      due: this.api.due(200).pipe(
        map((list) => list.cards.filter((c) => c.lessonId === id)),
        catchError(() => of<DueCard[]>([])),
      ),
      result: this.api.lessonStats().pipe(
        map((list) => list.find((s) => s.lessonId === id)?.result ?? null),
        catchError(() => of<LessonResult | null>(null)),
      ),
    }).subscribe({
      next: ({ lesson, vocabulary, practice, due, result }) => {
        this.lastResult.set(result);
        const seed = Math.floor(Math.random() * 0x7fffffff);
        const bank = buildBank(
          vocabulary.items,
          due,
          practice?.fill ?? null,
          practice?.translations ?? [],
          seed,
        );
        this.data.set({ title: lesson.title, bank, items: vocabulary.items, due, practice });
        this.parts.set(PARTS.filter((p) => partCount(bank, p) > 0));
      },
      error: () => this.loadError.set(true),
    });
  }

  protected count(part: Part): number {
    const d = this.data();
    return d ? partCount(d.bank, part) : 0;
  }

  protected togglePart(part: Part): void {
    this.parts.update((ps) => (ps.includes(part) ? ps.filter((p) => p !== part) : [...ps, part]));
  }

  /** "Ôn cả bài": a round of every part chosen. */
  protected start(): void {
    const d = this.data();
    if (d && this.roundSize() > 0) {
      this.begin(roundOf(d.bank, this.parts()), 'full');
    }
  }

  /** "Ôn lại câu sai" on the intro: the questions got wrong in the latest round. */
  protected startWrong(): void {
    if (this.lastWrong().length > 0) {
      this.begin(this.lastWrong(), 'wrong');
    }
  }

  /**
   * "Ôn lại cả bài": a new round of the same parts, the questions drawn and shuffled again. The due
   * cards were rated in the round just done, so their words come back without a rating.
   */
  protected restartAll(): void {
    const d = this.data();
    if (!d) {
      return;
    }
    const seed = Math.floor(Math.random() * 0x7fffffff);
    const drawn = buildBank(
      d.items,
      d.due,
      d.practice?.fill ?? null,
      d.practice?.translations ?? [],
      seed,
    );
    const bank = {
      ...drawn,
      vocab: drawn.vocab.map((q) => (q.kind === 'vocab' ? { ...q, card: null } : q)),
    };
    this.data.set({ ...d, bank });
    this.begin(roundOf(bank, this.parts()), 'full');
  }

  /** "Ôn lại câu sai": a new round of the questions got wrong; their cards are already rated. */
  protected retryWrong(): void {
    this.begin(
      this.wrong().map((a) =>
        a.question.kind === 'vocab' ? { ...a.question, card: null } : a.question,
      ),
      'wrong',
    );
  }

  private begin(questions: Question[], kind: RoundKind): void {
    this.roundKind.set(kind);
    this.saveError.set(false);
    this.questions.set(questions);
    this.answers.set([]);
    this.index.set(0);
    this.resetQuestion();
    this.stage.set('quiz');
    this.focusSubmit();
  }

  protected pick(i: number): void {
    if (!this.verdict()) {
      this.picked.set(i);
    }
  }

  protected addTile(i: number): void {
    if (!this.verdict() && !this.chosen().includes(i)) {
      this.chosen.update((c) => [...c, i]);
    }
  }

  protected removeTile(pos: number): void {
    if (!this.verdict()) {
      this.chosen.update((c) => c.filter((_, i) => i !== pos));
    }
  }

  protected tileUsed(i: number): boolean {
    return this.chosen().includes(i);
  }

  /** "Nộp": checks the answer. */
  protected submit(): void {
    const q = this.question();
    if (!q || this.verdict() || !this.canSubmit()) {
      return;
    }
    const right =
      q.kind === 'translate'
        ? checkTiles(this.built(), q.answer)
        : this.picked() === q.choices.answer;
    this.reveal(right ? 'correct' : 'wrong');
  }

  /** "Quên" / "Bỏ qua": shows the answer without one. */
  protected skip(): void {
    if (this.question() && !this.verdict()) {
      this.picked.set(null);
      this.reveal('skipped');
    }
  }

  private reveal(v: Verdict): void {
    const q = this.question();
    this.verdict.set(v);
    if (q?.kind === 'vocab' && q.card) {
      // A due card of the lesson is rated by the answer, as in the free review.
      const card = q.card;
      this.api
        .review(card.id, {
          rating: v === 'correct' ? 3 : 1,
          mode: 'flip',
          reps: card.reps,
          context: 'free',
        })
        .subscribe({ error: () => undefined });
    }
    this.focusNext();
  }

  /** "Tiếp": keeps the answer and moves on, or ends the round. */
  protected next(): void {
    const q = this.question();
    const v = this.verdict();
    if (!q || !v) {
      return;
    }
    this.answers.update((a) => [
      ...a,
      { question: q, correct: v === 'correct', given: this.givenText(q) },
    ]);
    if (this.index() + 1 < this.questions().length) {
      this.index.update((i) => i + 1);
      this.resetQuestion();
      this.focusSubmit();
      return;
    }
    this.finish();
  }

  private finish(): void {
    this.stage.set('result');
    if (this.nextLesson) {
      // The review the new lesson offered is done: it is not offered again.
      settleReviewOffer(this.nextLesson);
    }
    // The lesson's score: shown at once, then kept on the server.
    // Ids the lesson no longer has (its content changed) are not held against it.
    const last = this.lastResult();
    const previous = last && { ...last, wrong: this.lastWrong().map((q) => q.id) };
    const result = resultAfter(this.roundKind(), this.answers(), previous);
    this.lastResult.set({ ...result, reviewedAt: new Date().toISOString() });
    this.api.saveLessonResult(this.lessonId, result).subscribe({
      next: (saved) => this.lastResult.set(saved),
      error: () => this.saveError.set(true),
    });
    const words = missedWords(this.answers());
    if (words.length > 0) {
      this.api.practiceMisses(this.lessonId, words).subscribe({ error: () => undefined });
    }
  }

  private givenText(q: Question): string | null {
    if (this.verdict() === 'skipped') {
      return null;
    }
    if (q.kind === 'translate') {
      return this.built().join(' ');
    }
    const p = this.picked();
    return p === null ? null : q.choices.options[p];
  }

  /** When a due card comes back after this answer, e.g. "10 phút"; empty for other questions. */
  protected comesBack(): string {
    const q = this.question();
    if (q?.kind !== 'vocab' || !q.card) {
      return '';
    }
    return intervalLabel(q.card.intervals[this.verdict() === 'correct' ? 'good' : 'again']);
  }

  protected ago(iso: string): string {
    return daysAgo(iso, Date.now());
  }

  protected say(text: string): void {
    this.speech.speak(text, 1);
  }

  protected answerText(a: Answer): string {
    const q = a.question;
    return q.kind === 'vocab' ? q.meaningVi : q.kind === 'fill' ? q.answer : q.answer.join(' ');
  }

  protected promptText(a: Answer): string {
    const q = a.question;
    return q.kind === 'vocab'
      ? q.word
      : q.kind === 'fill'
        ? `${q.before}___${q.after}`
        : `"${q.vi}"`;
  }

  protected onKeydown(event: KeyboardEvent): void {
    if (
      this.stage() !== 'quiz' ||
      ((event.target as HTMLElement).tagName === 'BUTTON' && event.key === 'Enter')
    ) {
      return;
    }
    const q = this.question();
    if (event.key === 'Enter') {
      event.preventDefault();
      if (this.verdict()) {
        this.next();
      } else {
        this.submit();
      }
      return;
    }
    const n = Number(event.key);
    if (q && q.kind !== 'translate' && !this.verdict() && n >= 1 && n <= q.choices.options.length) {
      event.preventDefault();
      this.picked.set(n - 1);
    }
  }

  private resetQuestion(): void {
    this.picked.set(null);
    this.chosen.set([]);
    this.verdict.set(null);
  }

  private focusNext(): void {
    afterNextRender(() => this.nextButton()?.nativeElement.focus(), { injector: this.injector });
  }

  private focusSubmit(): void {
    afterNextRender(() => this.submitButton()?.nativeElement.focus(), { injector: this.injector });
  }
}
