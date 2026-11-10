import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';

import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IrregularVerb } from '../irregular-verbs.data';
import { VerbLesson } from '../verb-lessons.data';
import { changedPart, choiceQuestion, hasPattern, threeForms } from '../verb-study.logic';

/** A form cut where it changes: [same, changed]. */
type Cut = [string, string];

interface Row {
  verb: IrregularVerb;
  cells: [Cut, Cut, Cut];
}

/** The forms of a verb, each cut where it differs: V1 from V2, V2 and V3 from V1. */
function cuts(v: IrregularVerb): [Cut, Cut, Cut] {
  const [base, past, participle] = threeForms(v);
  return [changedPart(past, base), changedPart(base, past), changedPart(base, participle)];
}

/**
 * The start of a lesson's "Làm quen" (design/de-xuat-dong-tu.png part 1, added 2026-11-10): the rule
 * of the lesson on all its verbs at once, what changes coloured alike, then a guess of the last verb's
 * V2 from the rule before seeing it. The guess is not graded. A lesson whose verbs share no rule
 * ("Hay gặp", "Còn lại", "Hiếm") shows the table only.
 */
@Component({
  selector: 'lu-verb-pattern',
  imports: [NgOptimizedImage, Icon],
  templateUrl: './verb-pattern.html',
  styleUrl: './verb-pattern.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbPattern {
  readonly lesson = input.required<VerbLesson>();
  readonly verbs = input.required<readonly IrregularVerb[]>();
  /** Picture URL by base form; a verb without one shows none. */
  readonly pictures = input<ReadonlyMap<string, string>>(new Map());
  readonly finished = output();

  private readonly seed = newSeed();
  protected readonly stage = signal<'rule' | 'guess'>('rule');
  protected readonly chosen = signal<string | null>(null);

  /** The verb to guess: the last one of a lesson with a rule. */
  protected readonly guessVerb = computed(() => {
    const vs = this.verbs();
    return hasPattern(this.lesson()) && vs.length >= 2 ? vs[vs.length - 1] : null;
  });
  protected readonly question = computed(() => {
    const v = this.guessVerb();
    return v ? choiceQuestion(v, 'past', this.verbs(), this.seed) : null;
  });
  protected readonly rows = computed<Row[]>(() =>
    this.verbs().map((v) => ({ verb: v, cells: cuts(v) })),
  );
  protected readonly right = computed(() => {
    const q = this.question();
    return !!q && this.chosen() === q.answer;
  });

  /** The guessed verb's forms stay hidden until the guess. */
  protected hidden(v: IrregularVerb): boolean {
    return v === this.guessVerb() && this.chosen() === null;
  }

  protected toGuess(): void {
    if (this.guessVerb()) {
      this.stage.set('guess');
    } else {
      this.finished.emit();
    }
  }

  protected choose(option: string): void {
    if (this.chosen() === null) {
      this.chosen.set(option);
    }
  }
}
