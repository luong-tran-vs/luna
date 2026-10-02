import {
  inject,
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  output,
  signal,
} from '@angular/core';

import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';

/** Step 1: the lesson's words with IPA, meaning and an example sentence to hear. */
@Component({
  selector: 'lu-vocab-step',
  imports: [Icon],
  templateUrl: './vocab-step.html',
  styleUrls: ['../practice.css', './vocab-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VocabStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  /** False when the browser has no voice: the listen buttons are hidden. */
  protected readonly canSpeak = inject(SpeechService).supported;

  /** The lesson's words; empty when the lesson has none. */
  readonly words = input.required<VocabItem[]>();
  /** Example sentences by lemma; a word without one shows no example. */
  readonly examples = input<PracticeExample[]>([]);

  /** Text to read aloud with the browser's voice. */
  readonly readAloud = output<string>();

  protected readonly open = signal(true);

  protected readonly rows = computed(() => {
    const byLemma = new Map(this.examples().map((e) => [e.lemma.toLowerCase(), e]));
    return this.words().map((w) => ({
      word: w,
      example: byLemma.get(w.lemma.toLowerCase()) ?? null,
    }));
  });

  protected playWord(text: string): void {
    this.readAloud.emit(text);
  }
}
