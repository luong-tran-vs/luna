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

import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IrregularVerb } from '../irregular-verbs.data';
import {
  ChoiceQuestion,
  choiceQuestion,
  Form,
  SentenceQuestion,
  sentenceQuestion,
} from '../verb-study.logic';
import { VERB_USAGE } from '../verb-usage.data';

/** A question of the round: in a sentence, or the plain one when the verb has no sentence. */
type Question = { kind: 'sentence'; q: SentenceQuestion } | { kind: 'plain'; q: ChoiceQuestion };

/**
 * "Chọn" of a lesson (design/de-xuat-dong-tu.png part 4, added 2026-11-10): one question per verb,
 * V2 and V3 in turn, each a sentence with the verb blanked out and what tells the form (yesterday,
 * has…) marked; the learner picks V1, V2, V3 or the regular guess. A wrong pick is explained by the
 * signal of the sentence. Nothing is saved: the lesson goes on to typing.
 */
@Component({
  selector: 'lu-verb-sentence-round',
  imports: [Icon],
  templateUrl: './verb-sentence-round.html',
  styleUrl: './verb-sentence-round.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbSentenceRound {
  private readonly injector = inject(Injector);
  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  readonly verbs = input.required<readonly IrregularVerb[]>();
  readonly label = input('Bước 2 · Chọn trong câu');
  readonly finished = output();

  private readonly seed = newSeed();
  protected readonly index = signal(0);
  protected readonly chosen = signal<string | null>(null);

  protected readonly questions = computed<Question[]>(() => {
    const verbs = this.verbs();
    return verbs.map((v, i) => {
      const form: Form = i % 2 === 0 ? 'past' : 'participle';
      const usage = VERB_USAGE[v.base];
      const s = sentenceQuestion(
        v,
        form,
        form === 'past' ? usage?.past : usage?.perfect,
        this.seed + i,
      );
      return s
        ? { kind: 'sentence', q: s }
        : { kind: 'plain', q: choiceQuestion(v, form, verbs, this.seed + i) };
    });
  });
  protected readonly current = computed(() => this.questions()[this.index()]);
  protected readonly right = computed(() => this.chosen() === this.current()?.q.answer);

  /** Why the answer is the right one, from the form asked. */
  protected rule(form: Form): string {
    return form === 'past'
      ? 'Câu có dấu hiệu quá khứ (yesterday, last…, … ago) nên dùng V2.'
      : 'Sau have / has / had dùng V3.';
  }

  protected choose(option: string): void {
    if (this.chosen() === null) {
      this.chosen.set(option);
    }
  }

  protected next(): void {
    if (this.index() + 1 < this.questions().length) {
      this.index.update((i) => i + 1);
      this.chosen.set(null);
      afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
      return;
    }
    this.finished.emit();
  }
}
