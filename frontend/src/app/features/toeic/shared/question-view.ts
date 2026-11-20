import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { letter } from './score.logic';

/** A question as the views need it (the exam room's or the answer detail's). */
export interface QuestionLike {
  id: string;
  number: number;
  prompt?: string;
  options: readonly string[];
}

/**
 * A question with its options A–D (A–C in Part 2). In the exam room the learner picks one and may
 * flag the question; read-only, it shows the right answer and the learner's choice with ✓ / ✗.
 */
@Component({
  selector: 'lu-toeic-question',
  template: `
    <div
      class="q"
      role="group"
      [class.current]="current()"
      [attr.id]="'q-' + question().id"
      [attr.aria-labelledby]="'qh-' + question().id"
    >
      <div class="q-head">
        <span class="num">{{ question().number }}</span>
        @if (question().prompt) {
          <span class="prompt" lang="en" [attr.id]="'qh-' + question().id">{{ question().prompt }}</span>
        } @else {
          <span class="prompt muted-prompt" [attr.id]="'qh-' + question().id">Chọn đáp án cho chỗ trống ({{ question().number }})</span>
        }
        @if (!readonly()) {
          <button
            type="button"
            class="flag"
            [class.on]="flagged()"
            [attr.aria-pressed]="flagged()"
            [attr.aria-label]="(flagged() ? 'Bỏ đánh dấu câu ' : 'Đánh dấu xem lại câu ') + question().number"
            (click)="flagToggled.emit()"
          >
            ⚑
          </button>
        }
      </div>
      <div class="options" role="radiogroup" [attr.aria-label]="'Câu ' + question().number">
        @for (o of options(); track $index) {
          <button
            type="button"
            role="radio"
            class="opt"
            [attr.aria-checked]="chosen() === $index"
            [attr.data-state]="o.state"
            [disabled]="readonly()"
            (click)="selected.emit($index)"
          >
            <span class="letter">{{ o.letter }}</span>
            <span class="text" lang="en">{{ o.text }}</span>
            @if (o.sign) {
              <span class="sign" [attr.aria-label]="o.signLabel">{{ o.sign }}</span>
            }
          </button>
        }
      </div>
    </div>
  `,
  styleUrl: './question-view.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class QuestionView {
  readonly question = input.required<QuestionLike>();
  readonly chosen = input<number | null>(null);
  readonly flagged = input(false);
  readonly current = input(false);
  readonly readonly = input(false);
  /** The right option, shown when read-only. */
  readonly answer = input<number | null>(null);
  /** Hide the words of the options (Part 1/2 are only heard). */
  readonly hideText = input(false);

  readonly selected = output<number>();
  readonly flagToggled = output<void>();

  protected readonly options = computed(() => {
    const answer = this.answer();
    const chosen = this.chosen();
    const ro = this.readonly() && answer !== null;
    return this.question().options.map((text, i) => {
      let state = chosen === i ? 'chosen' : '';
      let sign = '';
      let signLabel = '';
      if (ro && i === answer) {
        state = 'right';
        sign = '✓';
        signLabel = 'Đáp án đúng';
      } else if (ro && i === chosen) {
        state = 'wrong';
        sign = '✗';
        signLabel = 'Bạn chọn, sai';
      }
      return { letter: letter(i), text: this.hideText() ? '' : text, state, sign, signLabel };
    });
  });
}
