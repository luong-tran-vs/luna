import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spellings, spokenForms } from '../irregular-verbs.logic';

/** Reading speed of the forms: a little slower, so each one is clear. */
const RATE = 0.85;

type Form = 'past' | 'participle';

/**
 * The verbs as a table on wide screens and as one line each on phones ("begin · began · begun",
 * the meaning under it). Each verb is read aloud; a verb the learner knows has a tick.
 */
@Component({
  selector: 'lu-verb-table',
  imports: [Icon],
  templateUrl: './verb-table.html',
  styleUrl: './verb-table.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbTable {
  private readonly speech = inject(SpeechService);

  /** Verbs by letter; a group without a letter has no letter row (search results). */
  readonly groups = input.required<{ letter: string; verbs: IrregularVerb[] }[]>();
  /** Base forms of the verbs the learner knows. */
  readonly known = input<ReadonlySet<string>>(new Set());

  protected readonly canSpeak = this.speech.supported;
  protected readonly forms: readonly Form[] = ['past', 'participle'];
  protected readonly spellings = spellings;
  protected readonly spoken = spokenForms;

  /** The text being read aloud, to mark its row. */
  private readonly playing = computed(() => this.speech.playing() ?? this.speech.preparing());

  protected formOf(v: IrregularVerb, form: Form): string {
    return form === 'past' ? v.past : v.participle;
  }

  protected isPlaying(v: IrregularVerb): boolean {
    return this.playing() === spokenForms(v);
  }

  /** Reads "go, went, gone"; pressed again while reading, stops. */
  protected listen(v: IrregularVerb): void {
    if (this.isPlaying(v)) {
      this.speech.stop();
    } else {
      this.speech.speak(spokenForms(v), RATE);
    }
  }
}
