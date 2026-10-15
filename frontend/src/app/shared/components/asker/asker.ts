import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/**
 * The boy asking something in a message bubble (design, 2026-10-15): the question of a reading
 * question, the line to fill in, the sentence to translate. The content goes in the bubble, under
 * a small label ("Câu 1/4"); once answered the bubble turns green (right) or red (wrong). The
 * picture is decoration only: screen readers skip it, the content is the text.
 */
@Component({
  selector: 'lu-asker',
  imports: [NgOptimizedImage],
  template: `
    <div class="bubble" [class.right]="tone() === 'right'" [class.wrong]="tone() === 'wrong'">
      @if (label()) {
        <p class="counter">{{ label() }}</p>
      }
      <ng-content />
    </div>
    <img class="boy" ngSrc="images/boy-student.webp" width="71" height="112" alt="" />
  `,
  styleUrl: './asker.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Asker {
  /** A small line over the content, e.g. "Câu 1/4" or "Câu 2/8 · Minh"; empty for none. */
  readonly label = input('');
  /** Once answered: the bubble's colour. */
  readonly tone = input<'right' | 'wrong' | null>(null);
}
