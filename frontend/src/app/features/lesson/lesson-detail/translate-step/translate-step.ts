import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  linkedSignal,
  output,
} from '@angular/core';

import { PracticeTranslation } from '../../../../core/models/practice';
import { Icon } from '../../../../shared/components/icon/icon';
import { checkTranslation } from '../practice-logic';

/**
 * Step 4, one sentence: build the English translation by tapping tiles in order. Tiles can be
 * removed one by one or all at once (Làm lại); Kiểm tra compares the tiles with the answer in order.
 * A new sentence starts empty.
 */
@Component({
  selector: 'lu-translate-step',
  imports: [Icon],
  templateUrl: './translate-step.html',
  styleUrls: ['../practice.css', './translate-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TranslateStep {
  readonly translation = input.required<PracticeTranslation>();
  /** The tiles, already shuffled by the page. */
  readonly tiles = input.required<string[]>();
  /** Position of the sentence (from 0) and number of sentences. */
  readonly index = input(0);
  readonly count = input(1);

  readonly playAudio = output<string>();
  readonly checked = output<boolean>();

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
