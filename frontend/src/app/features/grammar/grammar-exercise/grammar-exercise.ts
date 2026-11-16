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
import { toSignal } from '@angular/core/rxjs-interop';
import { FormControl, ReactiveFormsModule } from '@angular/forms';

import { serverUrl } from '../../../core/api-url';
import { GrammarExercise, GrammarStructure } from '../../../core/models/grammar-study';
import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import {
  correctAnswerText,
  gradeChoice,
  gradeFill,
  gradeFix,
  gradeReorder,
  hintRemovedOption,
  shuffleOptions,
  shuffleWords,
  splitBlank,
} from '../grammar-logic';
import { ReportExercise } from '../report-exercise/report-exercise';

/**
 * One grammar exercise (choice, fill, reorder or fix), graded in the browser. With `reveal` the result
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
  /** Whether a fill or reorder has its own Kiểm tra button; off when the page's bar calls check(). */
  readonly inlineCheck = input(true);
  /** The point's patterns, shown by "Gợi ý" while practising (added 2026-11-16). */
  readonly rules = input<GrammarStructure[]>([]);
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
  /** "Gợi ý" was opened for this exercise. */
  protected readonly hintShown = linkedSignal<GrammarExercise, boolean>({
    source: this.exercise,
    computation: () => false,
  });

  protected readonly typed = new FormControl('', { nonNullable: true });
  private readonly typedValue = toSignal(this.typed.valueChanges, { initialValue: '' });

  /** The options in a random order; each keeps its stored index, which grading uses. */
  protected readonly options = computed(() =>
    shuffleOptions(this.exercise().options ?? [], this.seed()),
  );
  protected readonly blank = computed(() => splitBlank(this.exercise().text ?? ''));
  protected readonly tiles = computed(() => shuffleWords(this.exercise().words ?? [], this.seed()));
  protected readonly built = computed(() => this.picked().map((i) => this.tiles()[i]));
  protected readonly isDone = computed(() => this.result() !== null);
  /** The right answer as text, shown after a wrong one. */
  protected readonly correctText = computed(() => correctAnswerText(this.exercise()));
  /** The wrong option "Gợi ý" crosses out on a choice, null when there is none to cross out. */
  protected readonly removedOption = computed(() =>
    this.hintShown() ? hintRemovedOption(this.exercise(), this.seed()) : null,
  );
  /**
   * "Gợi ý" (added 2026-11-16) shows only while practising (a test shows no result, so no hint) and
   * before the answer: on a choice it crosses out one wrong option and shows the point's patterns;
   * on a fill only the patterns, so it needs some.
   */
  protected readonly canHint = computed(() => {
    if (!this.reveal() || this.isDone() || this.hintShown()) {
      return false;
    }
    const e = this.exercise();
    const choice = e.kind === 'choice' || e.kind === 'picture';
    return (
      (choice && hintRemovedOption(e, 0) !== null) ||
      ((choice || e.kind === 'fill') && this.rules().length > 0)
    );
  });
  /** The point's patterns shown under the exercise once "Gợi ý" is open. */
  protected readonly hintRules = computed(() => (this.hintShown() ? this.rules() : []));

  /** The picture of the scene of a practice choice sentence, shown above it (added 2026-11-16). */
  protected readonly sceneSrc = computed(() => {
    const u = this.exercise().hintImageUrl;
    return u ? serverUrl(u) : '';
  });
  /** The picture of a picture exercise (added 2026-11-06). */
  protected readonly pictureSrc = computed(() => {
    const u = this.exercise().imageUrl;
    return u ? serverUrl(u) : '';
  });
  /** A fill, reorder or fix has an answer to check (a choice is checked when tapped); a fix must be changed. */
  readonly canCheck = computed(() => {
    if (this.isDone()) {
      return false;
    }
    const kind = this.exercise().kind;
    switch (kind) {
      case 'fill':
        return this.typedValue().trim() !== '';
      case 'fix':
        return (
          this.typedValue().trim() !== '' &&
          this.typedValue().trim() !== (this.exercise().text ?? '').trim()
        );
      case 'reorder':
        return this.picked().length > 0;
      default:
        return false;
    }
  });

  constructor() {
    effect(() => {
      const e = this.exercise();
      untracked(() => {
        // A fix starts from the sentence with the mistake, which the learner corrects in place.
        this.typed.reset(e.kind === 'fix' ? (e.text ?? '') : '');
        this.typed.enable();
      });
    });
  }

  protected showHint(): void {
    if (this.canHint()) {
      this.hintShown.set(true);
    }
  }

  protected choose(index: number): void {
    if (this.isDone() || index === this.removedOption()) {
      return;
    }
    this.chosen.set(index);
    this.given.emit(this.exercise().options?.[index] ?? '');
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

  protected checkFix(): void {
    if (!this.canCheck()) {
      return;
    }
    this.typed.disable();
    this.given.emit(this.typed.value.trim());
    this.finish(gradeFix(this.exercise(), this.typed.value));
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

  /** Checks a fill, reorder or fix answer: the Kiểm tra button, here or in the page's bar. */
  check(): void {
    switch (this.exercise().kind) {
      case 'fill':
        this.checkFill();
        break;
      case 'reorder':
        this.checkReorder();
        break;
      case 'fix':
        this.checkFix();
        break;
    }
  }

  private finish(correct: boolean): void {
    this.result.set(correct);
    this.answered.emit(correct);
  }
}
