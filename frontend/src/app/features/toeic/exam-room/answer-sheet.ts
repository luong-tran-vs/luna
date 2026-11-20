import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { letter } from '../shared/score.logic';
import { byPart, ExamQuestion, ExamState } from './exam-state.logic';

/**
 * The answer sheet for phones: a row of round bubbles A–D (A–C in Part 2) per question, like the
 * paper one. Tapping a bubble answers; tapping the number goes to the question.
 */
@Component({
  selector: 'lu-toeic-answer-sheet',
  template: `
    @for (g of groups(); track g.part) {
      <section class="part">
        <h3 class="part-title">Part {{ g.part }}</h3>
        @for (r of g.rows; track r.id) {
          <div class="row" [class.current]="r.current">
            <button
              type="button"
              class="num"
              [attr.aria-label]="'Đi tới câu ' + r.number + (r.flagged ? ', đã đánh dấu' : '')"
              (click)="picked.emit(r.id)"
            >
              {{ r.number }}@if (r.flagged) {<span aria-hidden="true">⚑</span>}
            </button>
            <div class="bubbles" role="radiogroup" [attr.aria-label]="'Câu ' + r.number">
              @for (b of r.bubbles; track b.index) {
                <button
                  type="button"
                  role="radio"
                  class="bubble"
                  [attr.aria-checked]="b.on"
                  [attr.aria-label]="'Câu ' + r.number + ' chọn ' + b.letter"
                  (click)="chosen.emit({ id: r.id, choice: b.index })"
                >
                  {{ b.letter }}
                </button>
              }
            </div>
          </div>
        }
      </section>
    }
  `,
  styleUrl: './answer-sheet.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AnswerSheet {
  readonly questions = input.required<readonly ExamQuestion[]>();
  readonly state = input.required<ExamState>();
  readonly picked = output<string>();
  readonly chosen = output<{ id: string; choice: number }>();

  protected readonly groups = computed(() => {
    const s = this.state();
    return byPart(this.questions()).map((g) => ({
      part: g.part,
      rows: g.questions.map((q) => ({
        id: q.id,
        number: q.number,
        current: s.position === q.id,
        flagged: s.flags.includes(q.id),
        bubbles: Array.from({ length: q.part === 2 ? 3 : 4 }, (_, i) => ({
          index: i,
          letter: letter(i),
          on: s.answers[q.id] === i,
        })),
      })),
    }));
  });
}
