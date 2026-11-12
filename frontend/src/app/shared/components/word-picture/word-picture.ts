import { ChangeDetectionStrategy, Component, computed, input, linkedSignal } from '@angular/core';

import { serverUrl } from '../../../core/api-url';

/**
 * The picture of a word in a review (added 2026-11-12): a small box above the word, kept short so
 * the answers stay on a phone's screen. Nothing is shown without a picture, or when it fails to
 * load. The meaning or the word is written beside it, so the picture is decoration (no alt), and
 * an alt would give the answer away.
 */
@Component({
  selector: 'lu-word-picture',
  template: `
    @if (src() && !failed()) {
      <img
        class="picture"
        crossorigin="use-credentials"
        referrerpolicy="no-referrer"
        alt=""
        decoding="async"
        [src]="src()"
        [class.loaded]="loaded()"
        (load)="loaded.set(true)"
        (error)="failed.set(true)"
      />
    }
  `,
  styles: `
    :host {
      display: block;
    }
    :host:empty {
      display: none;
    }
    .picture {
      display: block;
      height: clamp(96px, 22dvh, 180px);
      max-width: 100%;
      margin: 0 auto;
      aspect-ratio: 4 / 3;
      object-fit: cover;
      background: var(--color-primary-soft);
      border-radius: var(--radius-md);
      opacity: 0;
      transition: opacity var(--motion-base) var(--ease-out);
    }
    .picture.loaded {
      opacity: 1;
    }
    @media (prefers-reduced-motion: reduce) {
      .picture {
        transition: none;
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WordPicture {
  /** The picture's API path ('' or missing: nothing shown). */
  readonly url = input<string | null | undefined>('');

  protected readonly src = computed(() => {
    const u = this.url();
    return u ? serverUrl(u) : '';
  });
  /** Both start again with each word, so the previous word's picture never shows. */
  protected readonly loaded = linkedSignal({ source: this.src, computation: () => false });
  protected readonly failed = linkedSignal({ source: this.src, computation: () => false });
}
