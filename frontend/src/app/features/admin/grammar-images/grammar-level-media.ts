import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import { GrammarLevelHints } from './grammar-level-hints';
import { GrammarLevelImages } from './grammar-level-images';
import { GrammarLevelPictures } from './grammar-level-pictures';

/**
 * The pictures of a level's grammar lessons in one card (added 2026-11-16, three cards were too
 * many): the "Nhìn ảnh chọn từ" exercises, the example pictures and the pictures of the choice
 * sentences, one row each.
 */
@Component({
  selector: 'lu-grammar-level-media',
  imports: [GrammarLevelPictures, GrammarLevelImages, GrammarLevelHints],
  template: `
    <section class="card media" aria-labelledby="level-media-title">
      <h2 class="media-title" id="level-media-title">Ảnh minh hoạ {{ level() }}</h2>
      <lu-grammar-level-pictures [level]="level()" [pointIds]="drawableIds()" />
      <lu-grammar-level-images [level]="level()" [pointIds]="drawableIds()" [titles]="titles()" />
      <lu-grammar-level-hints [level]="level()" [pointIds]="lessonIds()" />
    </section>
  `,
  styles: `
    .media {
      display: grid;
      gap: 0;
    }

    .media-title {
      margin-bottom: var(--space-2);
      font: var(--text-lg) var(--font-ui);
      font-weight: 600;
    }

    .media > :not(.media-title) {
      display: block;
      padding-block: var(--space-3);
    }

    .media > :not(.media-title) + :not(.media-title) {
      border-top: 1px solid var(--color-line);
    }

    .media > :last-child {
      padding-bottom: 0;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLevelMedia {
  readonly level = input.required<string>();
  /** The points of the level that can be drawn and have a lesson. */
  readonly drawableIds = input.required<readonly string[]>();
  /** Every point of the level that has a lesson. */
  readonly lessonIds = input.required<readonly string[]>();
  readonly titles = input<Readonly<Record<string, string>>>({});
}
