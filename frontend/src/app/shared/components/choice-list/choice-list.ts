import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { Icon } from '../icon/icon';

/**
 * Up to four answers to pick one from (design/on-tap-theo-bai.png): a numbered row each, the picked
 * one outlined. Once `answer` is given the question is over: the right row turns green, a wrong pick
 * red, the others fade, and nothing can be picked any more. Keys 1–4 are the parent's job.
 */
@Component({
  selector: 'lu-choice-list',
  imports: [Icon],
  template: `
    <div class="choices" role="radiogroup" [attr.aria-label]="label()">
      @for (option of options(); track $index; let i = $index) {
        <button
          type="button"
          role="radio"
          class="choice"
          [class.picked]="picked() === i && answer() === null"
          [class.right]="answer() === i"
          [class.wrong]="answer() !== null && picked() === i && answer() !== i"
          [class.faded]="answer() !== null && answer() !== i && picked() !== i"
          [attr.aria-checked]="picked() === i"
          [attr.aria-disabled]="answer() !== null ? true : null"
          [attr.aria-keyshortcuts]="i + 1"
          [attr.lang]="lang()"
          (click)="pick(i)"
        >
          <span class="key" aria-hidden="true">
            @if (answer() === i) {
              <lu-icon name="check" />
            } @else if (answer() !== null && picked() === i) {
              <lu-icon name="x" />
            } @else {
              {{ i + 1 }}
            }
          </span>
          <span class="text">{{ option }}</span>
          @if (answer() === i) {
            <span class="visually-hidden">(đáp án đúng)</span>
          } @else if (answer() !== null && picked() === i) {
            <span class="visually-hidden">(bạn chọn, chưa đúng)</span>
          }
        </button>
      }
    </div>
  `,
  styleUrl: './choice-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ChoiceList {
  readonly options = input.required<readonly string[]>();
  /** The option picked, null for none. */
  readonly picked = input<number | null>(null);
  /** The right option once the answer is shown; null while the learner is still choosing. */
  readonly answer = input<number | null>(null);
  readonly label = input('Chọn đáp án');
  /** 'en' when the options are English words; empty for Vietnamese. */
  readonly lang = input('');
  readonly pickedChange = output<number>();

  protected pick(i: number): void {
    if (this.answer() === null) {
      this.pickedChange.emit(i);
    }
  }
}
