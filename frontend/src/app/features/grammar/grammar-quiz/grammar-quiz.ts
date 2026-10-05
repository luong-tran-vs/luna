import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { AttemptKind, AttemptResult, GrammarExercise } from '../../../core/models/grammar-study';
import { GrammarApiService } from '../../../core/services/grammar-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { GrammarExerciseView } from '../grammar-exercise/grammar-exercise';
import { correctAnswerText, missingToPass, Outcome, percent, toAttempt } from '../grammar-logic';
import { ReportExercise } from '../report-exercise/report-exercise';

/** One row of the review shown after a mastery test. */
export interface ReviewItem {
  exercise: GrammarExercise;
  correct: boolean;
  given: string;
  right: string;
  /** Position in the test, from 1. */
  number: number;
}

/**
 * A round of grammar exercises, one after another. In practice the answer is shown after each
 * exercise; in a mastery test it is not, and the score comes at the end. A full round is sent to the
 * server (POST …/attempts); a round that redoes only some exercises is not recorded.
 */
@Component({
  selector: 'lu-grammar-quiz',
  imports: [GrammarExerciseView, Icon, ReportExercise],
  templateUrl: './grammar-quiz.html',
  styleUrl: './grammar-quiz.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarQuiz {
  private readonly api = inject(GrammarApiService);
  private readonly injector = inject(Injector);

  readonly pointId = input.required<string>();
  readonly kind = input.required<AttemptKind>();
  readonly exercises = input.required<GrammarExercise[]>();
  /** Ids of the practice exercises answered wrong last time (offered as a quick redo). */
  readonly weak = input<string[]>([]);
  readonly recorded = output<AttemptResult>();

  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  /** Ids of a partial round (not recorded); null for the whole set. */
  protected readonly subset = signal<string[] | null>(null);
  protected readonly index = signal(0);
  protected readonly outcomes = signal<Outcome[]>([]);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  protected readonly result = signal<AttemptResult | null>(null);

  protected readonly items = computed(() => {
    const ids = this.subset();
    return ids ? this.exercises().filter((e) => ids.includes(e.id)) : this.exercises();
  });
  protected readonly current = computed(() => this.items()[this.index()] ?? null);
  protected readonly finished = computed(() => this.items().length > 0 && this.index() >= this.items().length);
  protected readonly answeredCurrent = computed(() => this.outcomes().length > this.index());
  protected readonly isLast = computed(() => this.index() + 1 >= this.items().length);
  protected readonly reveal = computed(() => this.kind() === 'practice');
  protected readonly recording = computed(() => this.subset() === null);
  protected readonly correct = computed(() => this.outcomes().filter((o) => o.correct).length);
  protected readonly total = computed(() => this.items().length);
  protected readonly score = computed(() => percent(this.correct(), this.total()));
  protected readonly wrongIds = computed(() => this.outcomes().filter((o) => !o.correct).map((o) => o.id));
  protected readonly missing = computed(() => missingToPass(this.correct(), this.total()));
  /** Weak exercises still in this set, offered before a round starts. */
  protected readonly weakIds = computed(() =>
    this.exercises()
      .filter((e) => this.weak().includes(e.id))
      .map((e) => e.id),
  );
  protected readonly canRetryWeak = computed(
    () => this.kind() === 'practice' && this.weakIds().length > 0 && this.subset() === null && !this.started(),
  );
  protected readonly started = computed(() => this.index() > 0 || this.outcomes().length > 0);

  /** Mastery test: every exercise with the learner's answer, wrong ones first. */
  protected readonly review = computed<ReviewItem[]>(() => {
    if (this.kind() !== 'mastery') {
      return [];
    }
    const outcomes = this.outcomes();
    const rows = this.items().flatMap((exercise, i) => {
      const o = outcomes[i];
      return o
        ? [{ exercise, correct: o.correct, given: o.given ?? '', right: correctAnswerText(exercise), number: i + 1 }]
        : [];
    });
    return [...rows.filter((r) => !r.correct), ...rows.filter((r) => r.correct)];
  });

  /** The answer of the exercise on screen, told just before `answered`. */
  private lastGiven = '';

  protected onGiven(text: string): void {
    this.lastGiven = text;
  }

  protected onAnswered(correct: boolean): void {
    const ex = this.current();
    if (ex && !this.answeredCurrent()) {
      this.outcomes.update((o) => [...o, { id: ex.id, correct, given: this.lastGiven }]);
    }
    this.lastGiven = '';
  }

  protected next(): void {
    if (!this.answeredCurrent()) {
      return;
    }
    this.index.update((i) => i + 1);
    if (this.finished() && this.recording()) {
      this.send();
    }
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }

  protected send(): void {
    this.saving.set(true);
    this.saveError.set(false);
    this.api.recordAttempt(this.pointId(), toAttempt(this.kind(), this.outcomes())).subscribe({
      next: (r) => {
        this.result.set(r);
        this.saving.set(false);
        this.recorded.emit(r);
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set(true);
      },
    });
  }

  /** A new round over the whole set. */
  protected restart(): void {
    this.subset.set(null);
    this.reset();
  }

  /** A round over only the exercises answered wrong (this round, or the last recorded one). */
  protected retryWrong(ids: string[]): void {
    this.subset.set(ids);
    this.reset();
  }

  private reset(): void {
    this.index.set(0);
    this.outcomes.set([]);
    this.result.set(null);
    this.saveError.set(false);
    this.saving.set(false);
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }
}
