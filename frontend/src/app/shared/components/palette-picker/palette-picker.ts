import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { PALETTES, PaletteService } from '../../../core/services/palette.service';

/**
 * Picks the app's color palette (admin Giao diện tab, learner Màu giao diện page). Each option carries
 * data-palette, so its swatches show that palette's own colors in the current light/dark mode.
 * A choice applies to the whole app at once.
 */
@Component({
  selector: 'lu-palette-picker',
  template: `<div class="palettes" role="radiogroup" [attr.aria-labelledby]="labelledBy()">
    @for (p of palettes; track p.id) {
      <label class="palette" [attr.data-palette]="p.id">
        <input
          type="radio"
          name="palette"
          [value]="p.id"
          [checked]="service.palette() === p.id"
          (change)="service.set(p.id)"
        />
        <span class="swatches" aria-hidden="true">
          <span style="background: var(--color-bg)"></span>
          <span style="background: var(--color-surface)"></span>
          <span style="background: var(--color-primary)"></span>
          <span style="background: var(--color-primary-soft)"></span>
          <span style="background: var(--color-accent)"></span>
          <span style="background: var(--color-text)"></span>
        </span>
        <span class="palette-name">{{ p.name }}</span>
        <span class="palette-note">{{ p.note }}</span>
      </label>
    }
  </div>`,
  styles: `
    .palettes {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
      gap: var(--space-3);
    }

    .palette {
      position: relative;
      display: grid;
      align-content: start;
      gap: var(--space-1);
      padding: var(--space-3);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
      cursor: pointer;
    }

    .palette:hover {
      border-color: var(--color-primary);
    }

    .palette input {
      position: absolute;
      opacity: 0;
      pointer-events: none;
    }

    /* Chosen: a thicker primary border plus a check mark, not color alone. */
    .palette:has(input:checked) {
      border-color: var(--color-primary);
      box-shadow: 0 0 0 1px var(--color-primary);
    }

    .palette:has(input:checked) .palette-name::after {
      content: ' ✓';
      color: var(--color-primary-ink);
    }

    .palette:has(input:focus-visible) {
      outline: 2px solid var(--color-primary);
      outline-offset: 2px;
    }

    .swatches {
      display: grid;
      grid-template-columns: repeat(6, 1fr);
      height: 28px;
      margin-bottom: var(--space-1);
      overflow: hidden;
      border: 1px solid var(--color-line);
      border-radius: var(--radius-sm);
    }

    .palette-name {
      font-weight: 600;
    }

    .palette-note {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PalettePicker {
  /** Id of the heading that names the group. */
  readonly labelledBy = input.required<string>();

  protected readonly service = inject(PaletteService);
  protected readonly palettes = PALETTES;
}
