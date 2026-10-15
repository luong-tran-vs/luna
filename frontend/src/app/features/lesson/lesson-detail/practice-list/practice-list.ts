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
        <!-- The whole row opens the step (not only its "Làm" at the end). -->
        <li>
          <button
            type="button"
            class="item"
            [class.done]="item.result !== null"
            [attr.aria-label]="(item.result === null ? 'Làm ' : 'Làm lại ') + item.name + ': ' + (item.result ?? 'chưa làm')"
            (click)="go.emit(item.key)"
          >
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
            <span class="go" [class.todo]="item.result === null" aria-hidden="true">
              {{ item.result === null ? 'Làm' : 'Làm lại' }} <lu-icon name="chevron-right" />
            </span>
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
      width: 100%;
      min-height: 56px;
      padding: var(--space-2) 0;
      font: inherit;
      color: var(--color-text);
      text-align: left;
      background: none;
      border: 0;
      cursor: pointer;
    }

    li + li .item {
      border-top: 1px solid var(--color-line);
    }

    .item:hover .name {
      color: var(--color-primary-ink);
    }

    .item:focus-visible {
      outline: 3px solid var(--color-primary);
      outline-offset: 2px;
      border-radius: var(--radius-sm);
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

    /* "Làm" (to do, filled) or "Làm lại" (done, quiet) with an arrow: the row is the button. */
    .go {
      --icon-size: 16px;

      display: inline-flex;
      flex: none;
      align-items: center;
      gap: 2px;
      padding: 6px var(--space-2) 6px var(--space-3);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      color: var(--color-primary-ink);
      background: var(--color-primary-soft);
      border-radius: var(--radius-full);
    }

    .go.todo {
      color: var(--color-on-primary);
      background: var(--color-primary);
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
