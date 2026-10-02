import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { fakeWaveform } from '../../utils/fake-waveform';

/** Width of one bar slot in viewBox units: a 2-unit bar and a 1-unit gap. */
const SLOT = 3;
const HEIGHT = 24;

let nextId = 0;

/**
 * Bars that look like the waveform of a spoken sentence, filled up to `progress` (0–1). Purely
 * decorative: the shape comes from the text (see fakeWaveform), so it is hidden from screen
 * readers and cannot be used to seek. Drawn in SVG so the colors follow the theme tokens.
 */
@Component({
  selector: 'lu-waveform',
  template: `<svg
    [attr.viewBox]="'0 0 ' + width() + ' ' + height"
    preserveAspectRatio="none"
    aria-hidden="true"
    focusable="false"
  >
    <defs>
      <clipPath [id]="clipId">
        <rect x="0" y="0" [attr.width]="played()" [attr.height]="height" />
      </clipPath>
    </defs>
    <path class="track" [attr.d]="path()" />
    <path class="played" [attr.d]="path()" [attr.clip-path]="'url(#' + clipId + ')'" />
  </svg>`,
  styles: `
    :host {
      display: block;
      height: 28px;
    }
    svg {
      display: block;
      width: 100%;
      height: 100%;
    }
    .track {
      fill: var(--color-track);
    }
    .played {
      fill: var(--color-primary);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Waveform {
  readonly text = input.required<string>();
  /** Part already played, from 0 to 1. */
  readonly progress = input(0);
  readonly bars = input(48);

  protected readonly height = HEIGHT;
  protected readonly clipId = `lu-waveform-${nextId++}`;
  protected readonly width = computed(() => this.bars() * SLOT);
  protected readonly played = computed(() => Math.min(1, Math.max(0, this.progress())) * this.width());

  /** One rectangle per bar, centered vertically. */
  protected readonly path = computed(() =>
    fakeWaveform(this.text(), this.bars())
      .map((h, i) => {
        const bar = Math.max(2, h * (HEIGHT - 2));
        return `M${i * SLOT} ${((HEIGHT - bar) / 2).toFixed(2)}h2v${bar.toFixed(2)}h-2z`;
      })
      .join(''),
  );
}
