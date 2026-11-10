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

import { VerbProgress } from '../../../core/models/verbs';
import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { VERB_EXAMPLES } from '../verb-lessons.data';
import {
  changedPart,
  exampleParts,
  QuickGrade,
  quickWaitDays,
  spokenForm,
  threeForms,
} from '../verb-study.logic';
import { VERB_USAGE } from '../verb-usage.data';

/** One verb of the round with its first grade. */
export interface QuickResult {
  verb: IrregularVerb;
  grade: QuickGrade;
}

const GRADES: readonly { grade: QuickGrade; label: string }[] = [
  { grade: 'forgot', label: 'Quên' },
  { grade: 'hard', label: 'Hơi nhớ' },
  { grade: 'good', label: 'Nhớ' },
];

/**
 * "Ôn nhanh" (design/de-xuat-dong-tu.png part 5, added 2026-11-10): the learner sees a verb's
 * picture, V1 and meaning, says V2 and V3 to themselves, flips the card and grades themselves:
 * Quên, Hơi nhớ or Nhớ, each with when the verb comes back. A forgotten verb is asked once more at
 * the end of the round; its first grade is the one kept.
 */
@Component({
  selector: 'lu-verb-flip-round',
  imports: [Icon],
  templateUrl: './verb-flip-round.html',
  styleUrl: './verb-flip-round.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbFlipRound {
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);
  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  readonly verbs = input.required<readonly IrregularVerb[]>();
  readonly progress = input<ReadonlyMap<string, VerbProgress>>(new Map());
  readonly pictures = input<ReadonlyMap<string, string>>(new Map());
  readonly finished = output<QuickResult[]>();

  protected readonly canSpeak = this.speech.supported;
  protected readonly grades = GRADES;
  /** The verbs to show, a forgotten one added again at the end. */
  private readonly extra = signal<IrregularVerb[]>([]);
  protected readonly queue = computed(() => [...this.verbs(), ...this.extra()]);
  protected readonly index = signal(0);
  protected readonly flipped = signal(false);
  private readonly results = signal<QuickResult[]>([]);

  protected readonly current = computed(() => this.queue()[this.index()] ?? null);
  protected readonly isRetry = computed(() => this.index() >= this.verbs().length);
  protected readonly cuts = computed(() => {
    const v = this.current();
    if (!v) {
      return [];
    }
    const [base, past, participle] = threeForms(v);
    return [changedPart(past, base), changedPart(base, past), changedPart(base, participle)];
  });
  protected readonly example = computed(() => {
    const v = this.current();
    return v ? (VERB_USAGE[v.base]?.perfect ?? VERB_EXAMPLES[v.base] ?? null) : null;
  });

  protected parts(text: string, v: IrregularVerb): { text: string; verb: boolean }[] {
    return exampleParts(text, v);
  }

  /** When the verb comes back after a grade, "1 ngày" or "6 ngày"; '' on a retry (not recorded). */
  protected waitLabel(v: IrregularVerb, grade: QuickGrade): string {
    if (this.isRetry()) {
      return '';
    }
    return grade === 'forgot'
      ? 'hỏi lại'
      : `${quickWaitDays(this.progress().get(v.base), grade)} ngày`;
  }

  protected listen(v: IrregularVerb): void {
    this.speech.speak(
      threeForms(v)
        .map((f, i) => spokenForm(v.base, f, i))
        .join(', '),
      0.85,
    );
  }

  protected flip(): void {
    this.flipped.set(true);
  }

  protected grade(g: QuickGrade): void {
    const v = this.current();
    if (!v) {
      return;
    }
    if (!this.isRetry()) {
      this.results.update((r) => [...r, { verb: v, grade: g }]);
      if (g === 'forgot') {
        this.extra.update((e) => [...e, v]);
      }
    }
    if (this.index() + 1 >= this.queue().length) {
      this.finished.emit(this.results());
      return;
    }
    this.index.update((i) => i + 1);
    this.flipped.set(false);
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }
}
