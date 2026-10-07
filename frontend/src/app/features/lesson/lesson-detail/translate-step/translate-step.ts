import {
  inject,
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  linkedSignal,
  output,
} from '@angular/core';

import { PracticeTranslation } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { checkTranslation } from '../practice-logic';
import { SpeakButton } from '../../../../shared/directives/speak-button';

/**
 * Step 4, one sentence: build the English translation by tapping tiles in order. Tiles can be
 * removed one by one or all at once (Làm lại); Kiểm tra compares the tiles with the answer in order.
 * A new sentence starts empty.
 */
@Component({
  selector: 'lu-translate-step',
  imports: [Icon, SpeakButton],
  templateUrl: './translate-step.html',
  styleUrls: ['../practice.css', './translate-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TranslateStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  /** False when the browser has no voice: the listen buttons are hidden. */
  protected readonly canSpeak = inject(SpeechService).supported;
  readonly translation = input.required<PracticeTranslation>();
  /** The tiles, already shuffled by the page. */
  readonly tiles = input.required<string[]>();
  /** Position of the sentence (from 0) and number of sentences. */
  readonly index = input(0);
  readonly count = input(1);

  /** Text to read aloud with the browser's voice. */
  readonly readAloud = output<string>();
  readonly checked = output<boolean>();
  /** ← and →, Câu tiếp theo: the sentence to show (the page keeps the index). */
  readonly go = output<number>();
  /** Hoàn thành after the last sentence: go to the next step. */
  readonly done = output<void>();
  /** Làm lại câu này: the sentence is cleared; the page forgets its result until checked again. */
  readonly redone = output<void>();
  /** Làm lại bước này: the page clears every result and goes back to the first sentence. */
  readonly restart = output<void>();

  /** Tile indexes in the order they were tapped. */
  protected readonly chosen = linkedSignal<PracticeTranslation, number[]>({
    source: this.translation,
    computation: () => [],
  });
  protected readonly result = linkedSignal<PracticeTranslation, boolean | null>({
    source: this.translation,
    computation: () => null,
  });

  protected readonly used = computed(() => new Set(this.chosen()));
  protected readonly answerText = computed(() => this.translation().answer.join(' '));

  protected pick(tile: number): void {
    if (this.result() === null && !this.used().has(tile)) {
      this.chosen.update((c) => [...c, tile]);
    }
  }

  protected remove(position: number): void {
    if (this.result() === null) {
      this.chosen.update((c) => c.filter((_, i) => i !== position));
    }
  }

  protected clear(): void {
    if (this.result() === null) {
      this.chosen.set([]);
    }
  }

  /** Làm lại câu này, after a check. */
  protected redo(): void {
    this.chosen.set([]);
    this.result.set(null);
    this.redone.emit();
  }

  /** Làm lại bước này: this sentence is cleared here, the others by the page. */
  protected restartAll(): void {
    this.chosen.set([]);
    this.result.set(null);
    this.restart.emit();
  }

  protected check(): void {
    if (this.result() !== null || this.chosen().length === 0) {
      return;
    }
    const tiles = this.tiles();
    const ok = checkTranslation(
      this.chosen().map((i) => tiles[i]),
      this.translation().answer,
    );
    this.result.set(ok);
    this.checked.emit(ok);
  }
}
