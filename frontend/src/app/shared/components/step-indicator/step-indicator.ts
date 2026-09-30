import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { Step, StepState, STEPS } from '../../../core/models/study';
import { ProgressBar } from '../progress-bar/progress-bar';

const LABELS: Record<Step, string> = { review: 'Ôn', read: 'Đọc', listen: 'Nghe' };
const MARKS: Record<StepState, string> = { done: '✓', current: '●', locked: '🔒' };
const STATES: Record<StepState, string> = { done: 'đã xong', current: 'đang làm', locked: 'chưa mở' };

/** The steps of today's lesson (Ôn · Đọc · Nghe) with done / current / locked states (L). */
@Component({
  selector: 'lu-step-indicator',
  imports: [ProgressBar],
  template: `
    <ol class="steps" aria-label="Các bước của bài">
      @for (s of items(); track s.step) {
        <li class="step" [attr.data-state]="s.state" [attr.aria-current]="s.state === 'current' ? 'step' : null">
          <span class="mark" aria-hidden="true">{{ s.mark }}</span>
          <span class="label">{{ s.label }}</span>
          <span class="visually-hidden">({{ s.stateLabel }})</span>
        </li>
      }
    </ol>
    <lu-progress-bar class="progress" label="Tiến độ bài" [value]="doneCount()" [max]="items().length" unit="bước" />
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-2);
    }
    .steps {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: var(--space-2);
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .step {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: var(--space-1);
      min-height: 44px;
      padding: var(--space-2);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }
    .step[data-state='done'] {
      color: var(--color-text);
    }
    .step[data-state='done'] .mark {
      color: var(--color-ok);
    }
    .step[data-state='current'] {
      font-weight: 600;
      color: var(--color-primary-ink);
      background: var(--color-primary-soft);
      border: 2px solid var(--color-primary);
    }
    .progress {
      --bar-height: 6px;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StepIndicator {
  readonly steps = input.required<Record<Step, StepState>>();

  protected readonly items = computed(() =>
    STEPS.map((step) => {
      const state = this.steps()[step];
      return { step, state, label: LABELS[step], mark: MARKS[state], stateLabel: STATES[state] };
    }),
  );
  protected readonly doneCount = computed(() => this.items().filter((i) => i.state === 'done').length);
}
