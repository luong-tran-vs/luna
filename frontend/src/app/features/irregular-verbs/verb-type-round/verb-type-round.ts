import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  input,
  linkedSignal,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { PATTERNS, patternOf, spokenForms } from '../irregular-verbs.logic';
import { gradeForm } from '../verb-study.logic';

/** Reading speed of the forms, as in the list. */
const RATE = 0.85;

/** The first answer of a verb in the round, with what was typed. */
export interface TypedResult {
  verb: IrregularVerb;
  right: boolean;
  past: string;
  participle: string;
}

/**
 * A round where the learner types V2 and V3 of each verb, graded per field. A verb got wrong is
 * asked again once at the end. Emits `finished` with the first answer of each verb.
 */
@Component({
  selector: 'lu-verb-type-round',
  imports: [Icon],
  templateUrl: './verb-type-round.html',
  styleUrl: './verb-type-round.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbTypeRound {
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);

  readonly verbs = input.required<IrregularVerb[]>();
  /** Shown before "x/y", e.g. "Bước 3 · Gõ". */
  readonly label = input('');
  readonly finished = output<TypedResult[]>();

  private readonly pastInput = viewChild<ElementRef<HTMLInputElement>>('pastInput');
  private readonly nextButton = viewChild<ElementRef<HTMLButtonElement>>('nextButton');

  protected readonly canSpeak = this.speech.supported;

  /** The verbs to ask, a wrong one added again at the end; a new list of verbs starts over. */
  protected readonly queue = linkedSignal(() => [...this.verbs()]);
  protected readonly index = linkedSignal<IrregularVerb[], number>({ source: this.verbs, computation: () => 0 });
  private readonly results = linkedSignal<IrregularVerb[], TypedResult[]>({
    source: this.verbs,
    computation: () => [],
  });
  protected readonly past = signal('');
  protected readonly participle = signal('');
  /** null until the answer on screen is checked. */
  protected readonly checked = signal<{ past: boolean; participle: boolean } | null>(null);

  protected readonly current = computed(() => this.queue()[this.index()] ?? null);
  protected readonly isLast = computed(() => this.index() + 1 >= this.queue().length);
  protected readonly isRetry = computed(() => this.index() >= this.verbs().length);
  protected readonly canCheck = computed(() => this.past().trim() !== '' && this.participle().trim() !== '');
  protected readonly right = computed(() => {
    const c = this.checked();
    return !!c && c.past && c.participle;
  });

  constructor() {
    this.focusInput();
  }

  protected patternLabel(v: IrregularVerb): string {
    return PATTERNS.find((p) => p.key === patternOf(v))?.label ?? '';
  }

  protected patternHint(v: IrregularVerb): string {
    return PATTERNS.find((p) => p.key === patternOf(v))?.hintVi ?? '';
  }

  protected forms(v: IrregularVerb): string {
    return `${v.base} – ${v.past} – ${v.participle}`;
  }

  protected onPast(event: Event): void {
    this.past.set((event.target as HTMLInputElement).value);
  }

  protected onParticiple(event: Event): void {
    this.participle.set((event.target as HTMLInputElement).value);
  }

  protected check(): void {
    const v = this.current();
    if (!v || this.checked() || !this.canCheck()) {
      return;
    }
    const result = { past: gradeForm(this.past(), v.past), participle: gradeForm(this.participle(), v.participle) };
    this.checked.set(result);
    if (!this.results().some((r) => r.verb.base === v.base)) {
      const right = result.past && result.participle;
      this.results.update((r) => [
        ...r,
        { verb: v, right, past: this.past().trim(), participle: this.participle().trim() },
      ]);
      if (!right) {
        this.queue.update((q) => [...q, v]);
      }
    }
    afterNextRender(() => this.nextButton()?.nativeElement.focus(), { injector: this.injector });
  }

  protected next(): void {
    if (!this.checked()) {
      return;
    }
    if (this.isLast()) {
      this.finished.emit(this.results());
      return;
    }
    this.index.update((i) => i + 1);
    this.past.set('');
    this.participle.set('');
    this.checked.set(null);
    this.focusInput();
  }

  protected listen(text: string): void {
    this.speech.speak(text, RATE);
  }

  protected listenForms(v: IrregularVerb): void {
    this.speech.speak(spokenForms(v), RATE);
  }

  private focusInput(): void {
    afterNextRender(() => this.pastInput()?.nativeElement.focus(), { injector: this.injector });
  }
}
