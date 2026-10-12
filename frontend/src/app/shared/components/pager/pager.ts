import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

/** A page number, or a gap ("…") between far pages. */
type PageItem = number | 'gap';

/**
 * Trước / page numbers / Sau under a list cut into pages (pages from 1). Shows nothing for a
 * single page. Far pages fold into "…": 1 … 4 5 6 … 12.
 */
@Component({
  selector: 'lu-pager',
  template: `
    @if (pageCount() > 1) {
      <nav class="pager" [attr.aria-label]="label()">
        <button type="button" class="btn" [disabled]="page() <= 1" (click)="go(page() - 1)">← Trước</button>
        <ul class="pages">
          @for (p of items(); track $index) {
            <li>
              @if (p === 'gap') {
                <span class="gap" aria-hidden="true">…</span>
              } @else {
                <button
                  type="button"
                  class="num"
                  [attr.aria-current]="p === page() ? 'page' : null"
                  [attr.aria-label]="'Trang ' + p"
                  (click)="go(p)"
                >
                  {{ p }}
                </button>
              }
            </li>
          }
        </ul>
        <button type="button" class="btn" [disabled]="page() >= pageCount()" (click)="go(page() + 1)">Sau →</button>
        @if (summary()) {
          <span class="summary">{{ summary() }}</span>
        }
      </nav>
    }
  `,
  styles: `
    .pager {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: center;
      gap: var(--space-2);
    }

    .pages {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-1);
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .num {
      min-width: 44px;
      min-height: 44px;
      padding: 0 var(--space-2);
      font: var(--text-base) var(--font-ui);
      font-variant-numeric: tabular-nums;
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
      cursor: pointer;
      transition: border-color 0.15s ease, background-color 0.15s ease;
    }

    .num:hover {
      border-color: var(--color-primary);
    }

    .num[aria-current='page'] {
      font-weight: 700;
      color: var(--color-primary-ink);
      background: var(--color-primary-soft);
      border: 2px solid var(--color-primary);
    }

    .gap {
      display: inline-flex;
      align-items: center;
      min-height: 44px;
      color: var(--color-text-muted);
    }

    .summary {
      flex-basis: 100%;
      font: var(--text-sm) var(--font-ui);
      font-variant-numeric: tabular-nums;
      color: var(--color-text-muted);
      text-align: center;
    }

    @media (prefers-reduced-motion: reduce) {
      .num {
        transition: none;
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Pager {
  readonly page = input.required<number>();
  readonly pageCount = input.required<number>();
  /** Names the navigation for screen readers, e.g. "Phân trang bài học". */
  readonly label = input('Phân trang');
  /** A line under the buttons, e.g. "21–40 / 57 bài". */
  readonly summary = input('');
  readonly pageChange = output<number>();

  protected readonly items = computed<PageItem[]>(() => pageItems(this.page(), this.pageCount()));

  protected go(page: number): void {
    const next = Math.min(Math.max(1, page), this.pageCount());
    if (next !== this.page()) {
      this.pageChange.emit(next);
    }
  }
}

/** The first and last page, the current one with one on each side, and gaps between. */
export function pageItems(page: number, count: number): PageItem[] {
  const shown = [...new Set([1, page - 1, page, page + 1, count])].filter((p) => p >= 1 && p <= count).sort((a, b) => a - b);
  const items: PageItem[] = [];
  for (const p of shown) {
    const last = items.at(-1);
    if (typeof last === 'number' && p - last > 1) {
      // A single missing page is shown instead of "…".
      items.push(p - last === 2 ? last + 1 : 'gap');
    }
    items.push(p);
  }
  return items;
}

/** The items of one page (pages from 1); the page is kept inside the page count. */
export function pageSlice<T>(items: readonly T[], page: number, size: number): T[] {
  const start = (clampPage(page, items.length, size) - 1) * size;
  return items.slice(start, start + size);
}

/** Number of pages for `total` items, at least 1. */
export function pageCountOf(total: number, size: number): number {
  return Math.max(1, Math.ceil(total / size));
}

/** `page` kept between 1 and the page count. */
export function clampPage(page: number, total: number, size: number): number {
  return Math.min(Math.max(1, page), pageCountOf(total, size));
}

/** "21–40 / 57" for page 2 of 20. */
export function rangeText(page: number, total: number, size: number): string {
  if (total === 0) {
    return '0';
  }
  const start = (clampPage(page, total, size) - 1) * size;
  return `${start + 1}–${Math.min(start + size, total)} / ${total}`;
}
