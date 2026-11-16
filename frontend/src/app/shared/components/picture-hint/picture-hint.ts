import { ChangeDetectionStrategy, Component, input, linkedSignal } from '@angular/core';

import { Icon } from '../icon/icon';

/**
 * A hint by pictures for a practice question (added 2026-11-16): a button that shows the pictures
 * of the words the answer needs, so the learner recalls them from their meaning rather than reading
 * them. Nothing shows when no word has a picture; new pictures close the hint again. The pictures are
 * decoration for screen readers (an alt would give the answer away).
 */
@Component({
  selector: 'lu-picture-hint',
  imports: [Icon],
  template: `
    @if (urls().length) {
      @if (open()) {
        <ul class="pics" aria-label="Gợi ý bằng ảnh">
          @for (u of urls(); track u) {
            <li>
              <img
                class="pic"
                crossorigin="use-credentials"
                referrerpolicy="no-referrer"
                alt=""
                decoding="async"
                width="96"
                height="72"
                [src]="u"
              />
            </li>
          }
        </ul>
      } @else {
        <button type="button" class="btn hint-btn" (click)="open.set(true)">
          <lu-icon name="bulb" /> Gợi ý bằng ảnh
        </button>
      }
    }
  `,
  styles: `
    :host {
      display: block;
    }
    :host:empty {
      display: none;
    }
    .hint-btn {
      gap: var(--space-1);
    }
    .pics {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-2);
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .pic {
      display: block;
      width: 96px;
      height: 72px;
      border-radius: var(--radius-md);
      object-fit: cover;
      background: var(--color-primary-soft);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PictureHint {
  /** The pictures to show, already server URLs; empty shows nothing. */
  readonly urls = input<readonly string[]>([]);
  /** Closed again when the pictures change (by content, not by array identity). */
  protected readonly open = linkedSignal({ source: () => this.urls().join(' '), computation: () => false });
}
