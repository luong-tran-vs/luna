import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';

import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { VocabApiService } from '../../../../core/services/vocab-api.service';
import { Icon } from '../../../../shared/components/icon/icon';

/** Step 1: the lesson's words with IPA, meaning, word audio and an example sentence to hear. */
@Component({
  selector: 'lu-vocab-step',
  imports: [Icon],
  templateUrl: './vocab-step.html',
  styleUrls: ['../practice.css', './vocab-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VocabStep {
  private readonly vocab = inject(VocabApiService);

  /** The lesson's words; empty when the lesson has none. */
  readonly words = input.required<VocabItem[]>();
  /** Example sentences by lemma; a word without one shows no example. */
  readonly examples = input<PracticeExample[]>([]);

  /** Asks the page to play an audio URL. */
  readonly playAudio = output<string>();

  protected readonly open = signal(true);

  protected readonly rows = computed(() => {
    const byLemma = new Map(this.examples().map((e) => [e.lemma.toLowerCase(), e]));
    return this.words().map((w) => ({
      word: w,
      example: byLemma.get(w.lemma.toLowerCase()) ?? null,
    }));
  });

  protected playWord(text: string): void {
    this.playAudio.emit(this.vocab.wordAudioUrl(text));
  }
}
