import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { fakeWaveform } from '../../utils/fake-waveform';

/** Width of one bar slot in viewBox units: a 1-unit bar and a 2-unit gap. */
const SLOT = 3;
const BAR = 1;
const HEIGHT = 24;

/**
 * Bars that look like the waveform of a spoken sentence, filled up to `progress` (0–1). Purely
 * decorative: the shape comes from the text (see fakeWaveform), so it is hidden from screen
 * readers and cannot be used to seek. Drawn in SVG so the colors follow the theme tokens.
 *
 * The played layer is the same bars clipped with CSS clip-path, which the browser animates: the
 * speech progress comes in steps (timer ticks, word events) and the transition turns them into a
 * smooth sweep. Going back to the start (a new playback) jumps without animation.
 */
@Component({
  selector: 'lu-waveform',
  template: `<svg [attr.viewBox]="'0 0 ' + width() + ' ' + height" preserveAspectRatio="none" focusable="false">
      <path class="track" [attr.d]="path()" />
    </svg>
    <svg
      class="played"
      [class.restart]="share() === 0"
      [style.clip-path]="clip()"
      [attr.viewBox]="'0 0 ' + width() + ' ' + height"
      preserveAspectRatio="none"
      focusable="false"
    >
      <path [attr.d]="path()" />
    </svg>`,
  styles: `
    :host {
      position: relative;
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
      position: absolute;
      inset: 0;
      fill: var(--color-primary);
      transition: clip-path 0.2s linear;
    }
    .played.restart {
      transition: none;
    }
    @media (prefers-reduced-motion: reduce) {
      .played {
        transition: none;
      }
    }
  `,
  host: { 'aria-hidden': 'true' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Waveform {
  readonly text = input.required<string>();
  /** Part already played, from 0 to 1. */
  readonly progress = input(0);
  readonly bars = input(48);

  protected readonly height = HEIGHT;
  protected readonly width = computed(() => this.bars() * SLOT);
  protected readonly share = computed(() => Math.min(1, Math.max(0, this.progress())));
  /** Hides the part of the played layer not reached yet. */
  protected readonly clip = computed(() => `inset(0 ${(100 - this.share() * 100).toFixed(2)}% 0 0)`);

  /** One rectangle per bar, centered vertically. */
  protected readonly path = computed(() =>
    fakeWaveform(this.text(), this.bars())
      .map((h, i) => {
        const bar = Math.max(2, h * (HEIGHT - 2));
        return `M${i * SLOT} ${((HEIGHT - bar) / 2).toFixed(2)}h${BAR}v${bar.toFixed(2)}h-${BAR}z`;
      })
      .join(''),
  );
}
