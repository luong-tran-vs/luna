import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  linkedSignal,
  output,
} from '@angular/core';

import { PracticeFill } from '../../../../core/models/practice';
import { Icon } from '../../../../shared/components/icon/icon';
import { checkFill, FillCheck, nextEmptyBlank, Score } from '../practice-logic';

type Part = { kind: 'text'; text: string } | { kind: 'blank'; index: number };

/**
 * Step 3: the dialogue with blanks. The selected blank takes the next word tapped in the bank;
 * a filled blank can be cleared (✕) or selected and replaced. Kiểm tra checks every blank at once.
 */
@Component({
  selector: 'lu-fill-step',
  imports: [Icon],
  templateUrl: './fill-step.html',
  styleUrls: ['../practice.css', './fill-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FillStep {
  readonly fill = input.required<PracticeFill>();
  /** The word bank, already shuffled by the page. */
  readonly bank = input.required<string[]>();
  readonly speakers = input<string[]>([]);
  /** Audio of each dialogue turn, by turn index. */
  readonly turnAudio = input<(string | null)[]>([]);
  readonly grammarTip = input('');

  readonly playAudio = output<string>();
  readonly checked = output<Score>();

  /** Bank index put in each blank, or null. Indexes keep tiles with the same word apart. */
  protected readonly answers = linkedSignal<(number | null)[]>(() =>
    this.fill().blanks.map(() => null),
  );
  /** The blank that takes the next word; -1 for none. */
  protected readonly selected = linkedSignal<number>(() =>
    this.fill().blanks.length > 0 ? 0 : -1,
  );
  protected readonly result = linkedSignal<PracticeFill, FillCheck | null>({
    source: this.fill,
    computation: () => null,
  });

  protected readonly words = computed(() =>
    this.answers().map((i) => (i === null ? null : (this.bank()[i] ?? null))),
  );
  protected readonly used = computed(() => new Set(this.answers().filter((i) => i !== null)));
  protected readonly complete = computed(() => this.answers().every((i) => i !== null));

  protected readonly turns = computed(() =>
    this.fill().turns.map((t) => ({
      ...t,
      parts: t.parts.map((p): Part =>
        'blank' in p ? { kind: 'blank', index: p.blank } : { kind: 'text', text: p.text },
      ),
    })),
  );

  protected speaker(index: number): string {
    return this.speakers()[index] ?? '';
  }

  protected audio(turnIndex: number): string | null {
    return this.turnAudio()[turnIndex] ?? null;
  }

  protected blankLabel(index: number): string {
    return `Ô trống ${index + 1}: ${this.words()[index] ?? 'trống'}`;
  }

  protected select(index: number): void {
    if (!this.result()) {
      this.selected.set(index);
    }
  }

  protected pick(bankIndex: number): void {
    const target = this.selected();
    if (this.result() || target < 0 || this.used().has(bankIndex)) {
      return;
    }
    const answers = [...this.answers()];
    answers[target] = bankIndex;
    this.answers.set(answers);
    this.selected.set(nextEmptyBlank(answers, target));
  }

  protected remove(index: number): void {
    if (this.result()) {
      return;
    }
    this.answers.update((a) => a.map((v, i) => (i === index ? null : v)));
    this.selected.set(index);
  }

  protected check(): void {
    if (!this.complete() || this.result()) {
      return;
    }
    const r = checkFill(this.words(), this.fill().blanks);
    this.result.set(r);
    this.selected.set(-1);
    this.checked.emit({ correct: r.correct, total: r.total });
  }
}
