import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { Icon } from '../../../../shared/components/icon/icon';
import type { PracticeItem } from '../lesson-detail';

/**
 * The lesson's practice, one row each: what was done (a score, "Đã nghe hết"), or "Chưa làm" with a
 * button into it, never a score for a step that was skipped. Used by the summary and by Luyện thêm
 * on the done card.
 */
@Component({
  selector: 'lu-practice-list',
  imports: [Icon],
  template: `
    @if (heading()) {
      <h3 class="list-heading">{{ heading() }}</h3>
    }
    <ul class="items">
      @for (item of items(); track item.key) {
        <li class="item" [class.done]="item.result !== null">
          <span class="mark" aria-hidden="true">
            @if (item.result !== null) {
              <lu-icon name="check" />
            } @else {
              –
            }
          </span>
          <span class="text">
            <span class="name">{{ item.name }}</span>
            <span class="result">{{ item.result ?? 'Chưa làm' }}</span>
          </span>
          <button
            type="button"
            [class]="item.result === null ? 'btn-primary go' : 'btn go'"
            (click)="go.emit(item.key)"
          >
            {{ item.result === null ? 'Làm' : 'Làm lại' }}<span class="visually-hidden"> {{ item.name }}</span>
          </button>
        </li>
      }
    </ul>
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-2);
    }

    .list-heading {
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      color: var(--color-text-muted);
    }

    .items {
      display: grid;
      margin: 0;
      padding: 0 var(--space-3);
      list-style: none;
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }

    .item {
      display: flex;
      align-items: center;
      gap: var(--space-3);
      padding: var(--space-2) 0;
    }

    .item + .item {
      border-top: 1px solid var(--color-line);
    }

    .mark {
      display: grid;
      flex: none;
      place-items: center;
      width: 28px;
      height: 28px;
      color: var(--color-text-muted);
      background: var(--color-primary-soft);
      border-radius: 50%;

      --icon-size: 16px;
    }

    .done .mark {
      color: var(--color-ok);
    }

    .text {
      display: grid;
      flex: 1;
      min-width: 0;
    }

    .name {
      font-weight: 600;
    }

    .result {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }

    .go {
      flex: none;
      min-height: 40px;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeList {
  readonly heading = input('');
  readonly items = input.required<PracticeItem[]>();
  /** The step to open: "Làm" or "Làm lại". */
  readonly go = output<PracticeItem['key']>();
}
