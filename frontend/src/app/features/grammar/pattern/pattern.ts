import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import { PatternRow } from '../grammar-logic';

/**
 * A structure's pattern, one line per question or form: the pieces in their colours joined by "+",
 * then the sample answers (after "->" in the pattern) as plain lines.
 */
@Component({
  selector: 'lu-pattern',
  template: `
    @for (row of rows(); track $index) {
      @if (row.answer) {
        <p class="answer" lang="en"><span class="arrow" aria-hidden="true">→</span> {{ row.parts[0].text }}</p>
      } @else {
        <p class="line" lang="en">
          @for (part of row.parts; track $index; let first = $first) {
            @if (!first) {
              <span class="plus">+</span>
            }
            <span class="piece" [class]="'tone-' + part.tone">{{ part.text }}</span>
          }
        </p>
      }
    }
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-2);
    }

    .line {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: var(--space-1);
      font: var(--text-base) var(--font-ui);
      font-weight: 600;
    }

    /* An answer sits in under the question it answers. */
    .answer {
      margin-top: calc(-1 * var(--space-1));
      padding-left: var(--space-4);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      color: var(--color-text-muted);
    }

    .arrow {
      color: var(--color-primary);
    }

    .plus {
      color: var(--color-text-muted);
    }

    .piece {
      padding: 1px var(--space-2);
      border-radius: var(--radius-sm);
      color: var(--color-tone-0);
      background: var(--color-tone-0-soft);
    }

    .piece.tone-1 {
      color: var(--color-tone-1);
      background: var(--color-tone-1-soft);
    }

    .piece.tone-2 {
      color: var(--color-tone-2);
      background: var(--color-tone-2-soft);
    }

    .piece.tone-3 {
      color: var(--color-tone-3);
      background: var(--color-tone-3-soft);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Pattern {
  readonly rows = input.required<readonly PatternRow[]>();
}
