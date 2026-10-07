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
import { checkFill, missedBlanks, Score } from '../practice-logic';
import { SpeakButton } from '../../../../shared/directives/speak-button';

type Part = { kind: 'text'; text: string } | { kind: 'blank'; index: number };

const key = (word: string) => word.trim().toLowerCase();

/**
 * Step 3: the dialogue with blanks to type in, one line at a time as in the Listening step. A word
 * tapped in the bank (the hints) goes into the blank last focused. Kiểm tra checks the blanks of
 * the line on screen, ignoring case; the score is reported once every line is checked.
 */
@Component({
  selector: 'lu-fill-step',
  imports: [Icon, SpeakButton],
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
  /** The score of every blank, once every line is checked. */
  readonly checked = output<Score>();
  /** The words of the blanks that were wrong, sent after each check. */
  readonly missed = output<string[]>();
  /** Hoàn thành after the last line: go to the next step. */
  readonly done = output<void>();
  /** Làm lại bước này: every line cleared; the page forgets the score until it is checked again. */
  readonly restarted = output<void>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  protected readonly turns = computed(() =>
    this.fill().turns.map((t) => {
      const parts = t.parts.map((p): Part =>
        'blank' in p ? { kind: 'blank', index: p.blank } : { kind: 'text', text: p.text },
      );
      return {
        ...t,
        parts,
        blanks: parts.flatMap((p) => (p.kind === 'blank' ? [p.index] : [])),
      };
    }),
  );

  /** The line on screen. */
  protected readonly shown = linkedSignal({ source: this.fill, computation: () => 0 });
  protected readonly turn = computed(() => this.turns()[this.shown()] ?? null);
  protected readonly isLast = computed(() => this.shown() >= this.turns().length - 1);

  /** What the learner typed in each blank (updated 2026-10-02: the blanks are text inputs). */
  protected readonly answers = linkedSignal<string[]>(() => this.fill().blanks.map(() => ''));
  /** Per blank: right, wrong, or null until its line is checked. */
  protected readonly results = linkedSignal<(boolean | null)[]>(() =>
    this.fill().blanks.map(() => null),
  );
  /** The blank a word of the bank goes to: the last one focused; -1 for none. */
  protected readonly selected = linkedSignal<number>(() => this.turns()[0]?.blanks[0] ?? -1);

  private readonly lineBlanks = computed(() => this.turn()?.blanks ?? []);
  protected readonly lineChecked = computed(() =>
    this.lineBlanks().every((i) => this.results()[i] !== null),
  );
  protected readonly lineComplete = computed(() =>
    this.lineBlanks().every((i) => this.answers()[i]?.trim() !== ''),
  );
  protected readonly lineScore = computed(() => {
    const blanks = this.lineBlanks();
    return { correct: blanks.filter((i) => this.results()[i] === true).length, total: blanks.length };
  });
  protected readonly allChecked = computed(() => this.results().every((r) => r !== null));
  protected readonly score = computed(() => ({
    correct: this.results().filter((r) => r === true).length,
    total: this.results().length,
  }));
  protected readonly checkedCount = computed(
    () => this.turns().filter((t) => t.blanks.every((i) => this.results()[i] !== null)).length,
  );

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
  /** Every input as wide as the longest answer, so the width does not give one away. */
  protected readonly inputSize = computed(() =>
    Math.max(8, ...this.fill().blanks.map((b) => b.answer.length + 2)),
  );

  protected speaker(index: number): string {
    return this.speakers()[index] ?? '';
  }

  protected turnText(turnIndex: number): string | null {
    return this.canSpeak ? (this.turnTexts()[turnIndex] ?? null) : null;
  }

  /** Shows line `index`, its first blank selected unless it is checked. */
  protected show(index: number): void {
    const turn = this.turns()[index];
    if (!turn) {
      return;
    }
    this.shown.set(index);
    const open = turn.blanks.find((i) => this.results()[i] === null);
    this.selected.set(open ?? -1);
  }

  protected select(index: number): void {
    if (this.results()[index] === null) {
      this.selected.set(index);
    }
  }

  protected type(index: number, event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.answers.update((a) => a.map((v, i) => (i === index ? value : v)));
  }

  /** Enter goes to the next blank of the line; on its last one it checks the line. */
  protected enter(index: number, event: Event): void {
    event.preventDefault();
    const blanks = this.lineBlanks();
    const next = blanks[blanks.indexOf(index) + 1];
    if (next !== undefined) {
      this.focusBlank(next);
    } else {
      this.check();
    }
  }

  /**
   * A word of the bank goes into the focused blank of the line when it is empty, else into the
   * line's next empty blank; with every blank filled it replaces the focused one. Then the next
   * empty blank of the line is focused.
   */
  protected pick(bankIndex: number): void {
    const blanks = this.lineBlanks();
    const from = blanks.includes(this.selected()) ? this.selected() : blanks[0];
    if (this.lineChecked() || from === undefined || this.used().has(bankIndex)) {
      return;
    }
    const empty = emptyFrom(blanks, this.answers(), from);
    const target = empty ?? from;
    const answers = this.answers().map((v, i) => (i === target ? this.bank()[bankIndex] : v));
    this.answers.set(answers);
    const next = emptyFrom(blanks, answers, target);
    if (next !== undefined) {
      this.focusBlank(next);
    } else {
      this.selected.set(target);
    }
  }

  /** Checks the line on screen; once every line is checked, reports the score. */
  protected check(): void {
    if (!this.lineComplete() || this.lineChecked()) {
      return;
    }
    const blanks = this.lineBlanks();
    const r = checkFill(this.answers(), this.fill().blanks);
    this.results.update((res) => res.map((v, i) => (blanks.includes(i) ? r.results[i] : v)));
    this.selected.set(-1);
    const all = this.fill().blanks;
    this.missed.emit(missedBlanks(blanks.map((i) => all[i]), blanks.map((i) => r.results[i])));
    if (this.allChecked()) {
      this.checked.emit(this.score());
    }
  }

  /** Làm lại câu này: the line on screen is cleared to be done again; the new check counts. */
  protected redoLine(): void {
    const blanks = this.lineBlanks();
    this.answers.update((a) => a.map((v, i) => (blanks.includes(i) ? '' : v)));
    this.results.update((r) => r.map((v, i) => (blanks.includes(i) ? null : v)));
    this.selected.set(blanks[0] ?? -1);
  }

  /** Làm lại bước này: every line cleared, back to the first one. */
  protected restart(): void {
    this.answers.set(this.fill().blanks.map(() => ''));
    this.results.set(this.fill().blanks.map(() => null));
    this.show(0);
    this.restarted.emit();
  }

  /** Câu tiếp theo, or Hoàn thành after the last line. */
  protected forward(): void {
    if (this.isLast()) {
      this.done.emit();
    } else {
      this.show(this.shown() + 1);
    }
  }

  private focusBlank(index: number): void {
    if (index < 0) {
      return;
    }
    this.selected.set(index);
    this.host.nativeElement.querySelector<HTMLInputElement>(`#fill-blank-${index}`)?.focus();
  }
}

/** The first empty blank of `blanks` from `from` on (wrapping round), or undefined. */
function emptyFrom(blanks: readonly number[], answers: readonly string[], from: number): number | undefined {
  const start = Math.max(0, blanks.indexOf(from));
  for (let k = 0; k < blanks.length; k++) {
    const i = blanks[(start + k) % blanks.length];
    if (!answers[i]?.trim()) {
      return i;
    }
  }
  return undefined;
}
