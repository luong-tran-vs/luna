import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { byPart, cellState, ExamQuestion, ExamState } from './exam-state.logic';

const CELL_LABELS = {
  current: 'đang xem',
  flagged: 'đã đánh dấu',
  answered: 'đã làm',
  empty: 'chưa làm',
} as const;

/**
 * The question grid by part (FR-007): answered, flagged, empty and current cells differ by colour
 * and by a sign, and a tap goes to the question.
 */
@Component({
  selector: 'lu-toeic-grid',
  template: `
    @for (g of groups(); track g.part) {
      <section class="part">
        <h3 class="part-title">Part {{ g.part }}</h3>
        <div class="cells">
          @for (c of g.cells; track c.id) {
            <button
              type="button"
              class="cell"
              [attr.data-state]="c.state"
              [attr.aria-label]="'Câu ' + c.number + ', ' + c.label"
              [attr.aria-current]="c.state === 'current' ? 'true' : null"
              (click)="picked.emit(c.id)"
            >
              {{ c.number }}
            </button>
          }
        </div>
      </section>
    }
    <p class="legend" aria-hidden="true">
      <span><i data-state="answered"></i>Đã làm</span>
      <span><i data-state="flagged"></i>⚑ Đánh dấu</span>
      <span><i data-state="empty"></i>Chưa làm</span>
    </p>
  `,
  styleUrl: './question-grid.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class QuestionGrid {
  readonly questions = input.required<readonly ExamQuestion[]>();
  readonly state = input.required<ExamState>();
  readonly picked = output<string>();

  protected readonly groups = computed(() => {
    const s = this.state();
    return byPart(this.questions()).map((g) => ({
      part: g.part,
      cells: g.questions.map((q) => {
        const st = cellState(s, q.id);
        return { id: q.id, number: q.number, state: st, label: CELL_LABELS[st] };
      }),
    }));
  });
}
