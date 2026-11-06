import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';

import { serverUrl } from '../../../core/api-url';

/**
 * The lesson's scene picture (added 2026-11-05): a wide banner of the story's place and people,
 * at the top of the lesson and of its Reading and Listening steps. Nothing is shown without a
 * picture, or when it fails to load. The picture is decoration beside the text, so it has no alt.
 */
@Component({
  selector: 'lu-scene-image',
  template: `
    @if (src() && !failed()) {
      <img
        class="scene"
        crossorigin="use-credentials"
        referrerpolicy="no-referrer"
        alt=""
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
    /* Without a picture nothing takes room, not even the margins a page gives the element. */
    :host:empty {
      display: none;
    }
    .scene {
      display: block;
      width: 100%;
      aspect-ratio: 16 / 9;
      object-fit: cover;
      background: var(--color-primary-soft);
      border-radius: var(--radius-lg);
      opacity: 0;
      transition: opacity var(--motion-base) var(--ease-out);
    }
    .scene.loaded {
      opacity: 1;
    }
    @media (prefers-reduced-motion: reduce) {
      .scene {
        transition: none;
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SceneImage {
  /** The picture's API path ('' or missing: nothing shown). */
  readonly url = input<string | null | undefined>('');

  protected readonly src = computed(() => {
    const u = this.url();
    return u ? serverUrl(u) : '';
  });
  protected readonly loaded = signal(false);
  protected readonly failed = signal(false);
}
