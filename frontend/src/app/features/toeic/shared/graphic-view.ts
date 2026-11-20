import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { Graphic } from '../../../core/models/toeic';

/**
 * The graphic of a Part 3/4 "Look at the graphic" question (FR-017), drawn from its data: a table,
 * schedule or price list; a floor plan on a grid; a bar chart with its values written. Readable at
 * 360 px, in light and dark, with the data as text for screen readers.
 */
@Component({
  selector: 'lu-toeic-graphic',
  template: `
    <figure class="graphic" lang="en">
      @if (graphic().title) {
        <figcaption class="g-title">{{ graphic().title }}</figcaption>
      }
      @switch (graphic().kind) {
        @case ('floor-plan') {
          <div
            class="plan"
            role="img"
            [attr.aria-label]="planLabel()"
            [style.grid-template-columns]="'repeat(' + planCols() + ', minmax(0, 1fr))'"
          >
            @for (c of graphic().cells ?? []; track $index) {
              <div
                class="room"
                [style.grid-row]="c.row + 1"
                [style.grid-column]="c.col + 1 + ' / span ' + (c.span || 1)"
              >
                {{ c.label }}
              </div>
            }
          </div>
        }
        @case ('bar-chart') {
          <div class="bars" role="img" [attr.aria-label]="barsLabel()">
            @for (b of bars(); track $index) {
              <div class="bar-col">
                <span class="bar-value">{{ b.value }}</span>
                <span class="bar" [style.height.%]="b.height"></span>
                <span class="bar-label">{{ b.label }}</span>
              </div>
            }
          </div>
        }
        @default {
          <div class="table-wrap">
            <table>
              @if (graphic().columns?.length) {
                <thead>
                  <tr>
                    @for (c of graphic().columns ?? []; track $index) {
                      <th scope="col">{{ c }}</th>
                    }
                  </tr>
                </thead>
              }
              <tbody>
                @for (r of graphic().rows ?? []; track $index) {
                  <tr>
                    @for (cell of r; track $index) {
                      <td>{{ cell }}</td>
                    }
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      }
    </figure>
  `,
  styles: `
    .graphic {
      display: grid;
      gap: var(--space-2);
      margin: 0;
      padding: var(--space-3);
      background: var(--color-surface);
      border: 2px solid var(--color-text);
      border-radius: var(--radius-md);
    }
    .g-title {
      font: var(--text-base) var(--font-ui);
      font-weight: 700;
      text-align: center;
    }
    .table-wrap {
      overflow-x: auto;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font: var(--text-sm) var(--font-ui);
    }
    th,
    td {
      padding: var(--space-1) var(--space-2);
      text-align: left;
      border: 1px solid var(--color-line);
    }
    th {
      background: var(--color-bg);
    }
    .plan {
      display: grid;
      gap: var(--space-1);
    }
    .room {
      display: grid;
      place-items: center;
      min-height: 52px;
      padding: var(--space-1);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      text-align: center;
      background: var(--color-bg);
      border: 2px solid var(--color-text-muted);
      border-radius: var(--radius-sm);
    }
    .bars {
      display: flex;
      gap: var(--space-2);
      align-items: stretch;
      height: 180px;
    }
    .bar-col {
      display: flex;
      flex: 1;
      flex-direction: column;
      gap: 2px;
      align-items: center;
      justify-content: flex-end;
      min-width: 0;
    }
    .bar {
      width: 70%;
      min-height: 2px;
      background: var(--color-primary);
      border-radius: var(--radius-sm) var(--radius-sm) 0 0;
    }
    .bar-value,
    .bar-label {
      font: var(--text-xs) var(--font-ui);
      text-align: center;
      overflow-wrap: anywhere;
    }
    .bar-label {
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GraphicView {
  readonly graphic = input.required<Graphic>();

  protected readonly planCols = computed(() =>
    Math.max(1, ...(this.graphic().cells ?? []).map((c) => c.col + (c.span || 1))),
  );
  protected readonly planLabel = computed(
    () => `Sơ đồ: ${(this.graphic().cells ?? []).map((c) => `${c.label} (hàng ${c.row + 1}, cột ${c.col + 1})`).join('; ')}`,
  );
  protected readonly bars = computed(() => {
    const list = this.graphic().bars ?? [];
    const max = Math.max(1, ...list.map((b) => b.value));
    return list.map((b) => ({ ...b, height: Math.round((b.value / max) * 80) }));
  });
  protected readonly barsLabel = computed(
    () => `Biểu đồ cột: ${(this.graphic().bars ?? []).map((b) => `${b.label} ${b.value}`).join('; ')}`,
  );
}
