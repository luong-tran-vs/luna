import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  input,
  linkedSignal,
  output,
} from '@angular/core';

import { PracticeFill } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { checkFill, FillCheck, missedBlanks, nextEmptyBlank, Score } from '../practice-logic';

type Part = { kind: 'text'; text: string } | { kind: 'blank'; index: number };

const key = (word: string) => word.trim().toLowerCase();

/**
 * Step 3: the dialogue with blanks to type in. A word tapped in the bank (the hints) goes into the
 * blank last focused. Kiểm tra checks every blank at once, ignoring case.
 */
@Component({
  selector: 'lu-fill-step',
  imports: [Icon],
  templateUrl: './fill-step.html',
  styleUrls: ['../practice.css', './fill-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FillStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  /** False when the browser has no voice: the listen buttons are hidden. */
  protected readonly canSpeak = inject(SpeechService).supported;
  readonly fill = input.required<PracticeFill>();
  /** The word bank, already shuffled by the page. */
  readonly bank = input.required<string[]>();
  readonly speakers = input<string[]>([]);
  /** Full text of each dialogue turn, to read aloud. */
  readonly turnTexts = input<string[]>([]);
  readonly grammarTip = input('');

  /** Text to read aloud with the browser's voice. */
  readonly readAloud = output<string>();
  readonly checked = output<Score>();
  /** The words of the blanks that were wrong, sent after each check. */
  readonly missed = output<string[]>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  /** What the learner typed in each blank (updated 2026-10-02: the blanks are text inputs). */
  protected readonly answers = linkedSignal<string[]>(() => this.fill().blanks.map(() => ''));
  /** The blank a word of the bank goes to: the last one focused; -1 for none. */
  protected readonly selected = linkedSignal<number>(() =>
    this.fill().blanks.length > 0 ? 0 : -1,
  );
  protected readonly result = linkedSignal<PracticeFill, FillCheck | null>({
    source: this.fill,
    computation: () => null,
  });

  /** Bank tiles whose word is already in a blank, one tile per blank holding it. */
  protected readonly used = computed(() => {
    const left = new Map<string, number>();
    for (const a of this.answers()) {
      if (a.trim()) {
        left.set(key(a), (left.get(key(a)) ?? 0) + 1);
      }
    }
    const out = new Set<number>();
    this.bank().forEach((w, i) => {
      const n = left.get(key(w)) ?? 0;
      if (n > 0) {
        out.add(i);
        left.set(key(w), n - 1);
      }
    });
    return out;
  });
  protected readonly complete = computed(() => this.answers().every((a) => a.trim() !== ''));
  /** Every input as wide as the longest answer, so the width does not give one away. */
  protected readonly inputSize = computed(() =>
    Math.max(8, ...this.fill().blanks.map((b) => b.answer.length + 2)),
  );

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

  protected turnText(turnIndex: number): string | null {
    return this.canSpeak ? (this.turnTexts()[turnIndex] ?? null) : null;
  }

  protected select(index: number): void {
    if (!this.result()) {
      this.selected.set(index);
    }
  }

  protected type(index: number, event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.answers.update((a) => a.map((v, i) => (i === index ? value : v)));
  }

  /** Enter goes to the next blank; on the last one it checks once every blank is filled. */
  protected enter(index: number, event: Event): void {
    event.preventDefault();
    if (index + 1 < this.answers().length) {
      this.focusBlank(index + 1);
    } else {
      this.check();
    }
  }

  /**
   * A word of the bank goes into the focused blank when it is empty, else into the next empty
   * blank; with every blank filled it replaces the focused one. Then the next empty blank is focused.
   */
  protected pick(bankIndex: number): void {
    const from = this.selected();
    if (this.result() || from < 0 || this.used().has(bankIndex)) {
      return;
    }
    const empty = nextEmptyBlank(this.answers(), from - 1);
    const target = empty < 0 ? from : empty;
    const answers = this.answers().map((v, i) => (i === target ? this.bank()[bankIndex] : v));
    this.answers.set(answers);
    const next = nextEmptyBlank(answers, target);
    if (next >= 0) {
      this.focusBlank(next);
    } else {
      this.selected.set(target);
    }
  }

  protected check(): void {
    if (!this.complete() || this.result()) {
      return;
    }
    const r = checkFill(this.answers(), this.fill().blanks);
    this.result.set(r);
    this.selected.set(-1);
    this.checked.emit({ correct: r.correct, total: r.total });
    this.missed.emit(missedBlanks(this.fill().blanks, r.results));
  }

  private focusBlank(index: number): void {
    this.selected.set(index);
    this.host.nativeElement.querySelector<HTMLInputElement>(`#fill-blank-${index}`)?.focus();
  }
}
