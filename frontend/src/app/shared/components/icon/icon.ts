import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

/** Line icons on a 24×24 grid, drawn with the current text color. */
const PATHS = {
  home: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z',
  book: 'M2 5h6a4 4 0 0 1 4 4v12a3 3 0 0 0-3-3H2zM22 5h-6a4 4 0 0 0-4 4v12a3 3 0 0 1 3-3h7z',
  review: 'M21 12a9 9 0 1 1-2.6-6.4L21 8M21 3v5h-5',
  user: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4 21a8 8 0 0 1 16 0',
  bell: 'M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a2 2 0 0 0 3.4 0',
  'chevron-right': 'M9 6l6 6-6 6',
  'arrow-left': 'M19 12H5M12 19l-7-7 7-7',
  'arrow-right': 'M5 12h14M12 5l7 7-7 7',
  lock: 'M6 11h12v10H6zM8 11V7a4 4 0 0 1 8 0v4',
  check: 'M5 12.5l4.5 4.5L19 7.5',
  x: 'M18 6 6 18M6 6l12 12',
  flame: 'M12 22c4 0 7-3 7-7 0-4-3-6-4-9-1 2-2 3-3 3 0-2-1-4-3-6 0 4-4 6-4 12 0 4 3 7 7 7z',
  cards: 'M3 7h13v14H3zM7 3h14v14',
  target: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 12h.01',
  chart: 'M4 20V10M10 20V4M16 20v-7M22 20H2',
  pencil: 'M4 20h4L19 9l-4-4L4 16zM14 6l4 4',
  notebook: 'M5 3h12a2 2 0 0 1 2 2v16H7a2 2 0 0 1-2-2zM9 7h6M9 11h6',
  sliders: 'M4 6h9M17 6h3M4 12h3M11 12h9M4 18h11M19 18h1M15 4v4M9 10v4M17 16v4',
  palette: 'M12 21a9 9 0 1 1 9-9c0 2.2-1.8 3.5-3.6 3.5h-2.2a2 2 0 0 0-1.5 3.3A1.4 1.4 0 0 1 12 21zM7.5 11h.01M10 7h.01M15 7.5h.01',
  moon: 'M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5z',
  sun: 'M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
  download: 'M12 3v12M7 10l5 5 5-5M4 21h16',
  logout: 'M15 4h4a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-4M10 17l5-5-5-5M15 12H3',
  headphones: 'M4 18v-6a8 8 0 0 1 16 0v6M4 15h3v6H4zM17 15h3v6h-3z',
  clock: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3 2',
  calendar: 'M4 5h16v16H4zM4 10h16M8 3v4M16 3v4',
  trophy: 'M8 4h8v6a4 4 0 0 1-8 0zM8 6H4a4 4 0 0 0 4 5M16 6h4a4 4 0 0 1-4 5M12 14v4M8 21h8',
  layers: 'M12 3 2 8l10 5 10-5zM2 13l10 5 10-5',
  play: 'M7 4.5v15a1 1 0 0 0 1.5.9l12-7.5a1 1 0 0 0 0-1.8l-12-7.5A1 1 0 0 0 7 4.5z',
  pause: 'M7 4h3v16H7zM14 4h3v16h-3z',
  grammar: 'M4 7V5h16v2M12 5v14M9 19h6',
  volume: 'M4 9h4l5-4v14l-5-4H4zM16.5 8.5a5 5 0 0 1 0 7M19.5 5.5a9 9 0 0 1 0 13',
  mic: 'M12 3a3 3 0 0 0-3 3v6a3 3 0 0 0 6 0V6a3 3 0 0 0-3-3zM5 11a7 7 0 0 0 14 0M12 18v3M8 21h8',
} as const;

export type IconName = keyof typeof PATHS;

/** A decorative icon: always hidden from screen readers, so the text next to it carries the meaning. */
@Component({
  selector: 'lu-icon',
  template: `<svg viewBox="0 0 24 24" focusable="false"><path [attr.d]="path()" /></svg>`,
  styles: `
    :host {
      display: inline-flex;
      flex: none;
      width: var(--icon-size, 20px);
      height: var(--icon-size, 20px);
    }
    svg {
      width: 100%;
      height: 100%;
      fill: none;
      stroke: currentColor;
      stroke-width: 2;
      stroke-linecap: round;
      stroke-linejoin: round;
    }
  `,
  host: { 'aria-hidden': 'true' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Icon {
  readonly name = input.required<IconName>();

  protected readonly path = computed(() => PATHS[this.name()]);
}
