import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  input,
  linkedSignal,
  output,
  untracked,
} from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';

import { GrammarExercise } from '../../../core/models/grammar-study';
import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { correctAnswerText, gradeChoice, gradeFill, gradeReorder, shuffleWords, splitBlank } from '../grammar-logic';
import { ReportExercise } from '../report-exercise/report-exercise';

/**
 * One grammar exercise (choice, fill or reorder), graded in the browser. With `reveal` the result
 * ("Đúng"/"Sai", the right answer and the explanation) shows at once; without it (a mastery test)
 * only "Đã ghi nhận" shows. Emits `answered` once, when the learner checks the answer. A new
 * `exercise` input starts it afresh.
 */
@Component({
  selector: 'lu-grammar-exercise',
  imports: [ReactiveFormsModule, Icon, ReportExercise],
  templateUrl: './grammar-exercise.html',
  styleUrls: ['../../lesson/lesson-detail/practice.css', './grammar-exercise.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarExerciseView {
  readonly exercise = input.required<GrammarExercise>();
  readonly reveal = input(true);
  /** F21: the point the exercise belongs to; with it, "Báo lỗi câu này" shows after the answer. */
  readonly pointId = input('');
  readonly answered = output<boolean>();
  /** What the learner answered, as text; emitted just before `answered`. */
  readonly given = output<string>();

  /** null until the answer is checked. */
  protected readonly result = linkedSignal<GrammarExercise, boolean | null>({
    source: this.exercise,
    computation: () => null,
  });
  protected readonly chosen = linkedSignal<GrammarExercise, number | null>({
    source: this.exercise,
    computation: () => null,
  });
  /** Indexes into the shuffled tiles, in the order tapped. */
  protected readonly picked = linkedSignal<GrammarExercise, number[]>({
    source: this.exercise,
    computation: () => [],
  });
  private readonly seed = linkedSignal<GrammarExercise, number>({
    source: this.exercise,
    computation: () => newSeed(),
  });

  protected readonly typed = new FormControl('', { nonNullable: true });

  protected readonly options = computed(() => this.exercise().options ?? []);
  protected readonly blank = computed(() => splitBlank(this.exercise().text ?? ''));
  protected readonly tiles = computed(() => shuffleWords(this.exercise().words ?? [], this.seed()));
  protected readonly built = computed(() => this.picked().map((i) => this.tiles()[i]));
  protected readonly isDone = computed(() => this.result() !== null);
  /** The right answer as text, shown after a wrong one. */
  protected readonly correctText = computed(() => correctAnswerText(this.exercise()));

  constructor() {
    effect(() => {
      this.exercise();
      untracked(() => {
        this.typed.reset('');
        this.typed.enable();
      });
    });
  }

  protected choose(index: number): void {
    if (this.isDone()) {
      return;
    }
    this.chosen.set(index);
    this.given.emit(this.options()[index] ?? '');
    this.finish(gradeChoice(this.exercise(), index));
  }

  protected checkFill(): void {
    if (this.isDone() || this.typed.value.trim() === '') {
      return;
    }
    this.typed.disable();
    this.given.emit(this.typed.value.trim());
    this.finish(gradeFill(this.exercise(), this.typed.value));
  }

  protected pick(tile: number): void {
    if (!this.isDone() && !this.picked().includes(tile)) {
      this.picked.update((p) => [...p, tile]);
    }
  }

  protected remove(position: number): void {
    if (!this.isDone()) {
      this.picked.update((p) => p.filter((_, i) => i !== position));
    }
  }

  protected clear(): void {
    if (!this.isDone()) {
      this.picked.set([]);
    }
  }

  protected checkReorder(): void {
    if (this.isDone() || this.picked().length === 0) {
      return;
    }
    this.given.emit(this.built().join(' '));
    this.finish(gradeReorder(this.exercise(), this.built()));
  }

  private finish(correct: boolean): void {
    this.result.set(correct);
    this.answered.emit(correct);
  }
}
