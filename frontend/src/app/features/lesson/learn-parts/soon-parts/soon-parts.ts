import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/** A part with no lesson yet, as the "Học" page lists it. */
export interface SoonPart {
  number: number;
  name: string;
  levelName: string;
  summary: string;
  milestone: { level: string };
}

/**
 * The parts with no lesson yet, together in one small box after the parts' cards: one line each
 * (Phần N, its name and what it is about), no card, picture or lock.
 */
@Component({
  selector: 'lu-soon-parts',
  template: `
    <section class="soon" aria-labelledby="soon-heading">
      <h2 class="soon-title" id="soon-heading">Sắp có · {{ parts().length }} phần</h2>
      <ul class="soon-list">
        @for (p of parts(); track p.milestone.level) {
          <li class="soon-item">
            <span class="soon-pill">Phần {{ p.number }}</span
            >&ngsp;
            <span class="soon-text">
              <span class="soon-name"
                >{{ p.name
                }}<span class="visually-hidden">
                  · {{ p.milestone.level }} {{ p.levelName }}</span
                ></span
              >&ngsp;
              <span class="soon-summary">{{ p.summary }}</span>
            </span>
          </li>
        }
      </ul>
    </section>
  `,
  styles: `
    .soon {
      margin-top: var(--space-3);
      padding: var(--space-2) var(--space-4);
      background: var(--color-surface);
      border: 1px dashed var(--color-line);
      border-radius: 18px;
    }

    .soon-title {
      margin: 0;
      padding: var(--space-1) 0;
      font: var(--text-xs) var(--font-ui);
      font-weight: 700;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: var(--color-text-muted);
    }

    .soon-list {
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .soon-item {
      display: flex;
      align-items: center;
      gap: var(--space-2);
      padding: var(--space-2) 0;
      border-top: 1px solid var(--color-line);
    }

    .soon-pill {
      flex: none;
      padding: 2px var(--space-2);
      font: var(--text-xs) var(--font-ui);
      font-weight: 700;
      color: var(--color-warn);
      background: color-mix(in srgb, var(--color-warn) 10%, transparent);
      border-radius: 999px;
    }

    .soon-text {
      display: grid;
      min-width: 0;
    }

    .soon-name {
      font: var(--text-sm) var(--font-ui);
      font-weight: 700;
    }

    .soon-summary {
      overflow: hidden;
      font: var(--text-xs) var(--font-ui);
      color: var(--color-text-muted);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SoonParts {
  readonly parts = input.required<SoonPart[]>();
}
