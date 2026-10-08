import { ChangeDetectionStrategy, Component, computed, inject, input, linkedSignal } from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spellings, spokenForms } from '../irregular-verbs.logic';

/** Reading speed of the forms: a little slower, so each one is clear. */
const RATE = 0.85;

type Form = 'past' | 'participle';

/**
 * The verbs as a table on wide screens and as cards on phones. Each verb is read aloud; in
 * self-test mode V2 and V3 are hidden until tapped.
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
  readonly testing = input(false);

  protected readonly canSpeak = this.speech.supported;
  protected readonly forms: readonly Form[] = ['past', 'participle'];
  protected readonly spellings = spellings;
  protected readonly spoken = spokenForms;

  /** Cells shown in self-test mode, as "base:past"; turning the mode on or off hides them again. */
  protected readonly revealed = linkedSignal<boolean, ReadonlySet<string>>({
    source: this.testing,
    computation: () => new Set(),
  });

  /** The text being read aloud, to mark its row. */
  private readonly playing = computed(() => this.speech.playing() ?? this.speech.preparing());

  protected formOf(v: IrregularVerb, form: Form): string {
    return form === 'past' ? v.past : v.participle;
  }

  protected isHidden(v: IrregularVerb, form: Form): boolean {
    return this.testing() && !this.revealed().has(`${v.base}:${form}`);
  }

  protected reveal(v: IrregularVerb, form: Form): void {
    this.revealed.update((s) => new Set(s).add(`${v.base}:${form}`));
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
