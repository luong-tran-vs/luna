import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/**
 * Loading indicator in place of a "Đang tải…" line: a small spinning arc in the theme colors,
 * centered in the screen (or, with class "inline", in the flow of a section).
 * The label stays in the page for screen readers (role="status"); it is shown as text instead of
 * the spinner when the system asks for reduced motion. The spinner fades in after a short delay
 * so fast loads do not flash it.
 */
@Component({
  selector: 'lu-loading',
  template: `<svg class="spinner" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <circle class="track" cx="12" cy="12" r="9" />
      <circle class="arc" cx="12" cy="12" r="9" />
    </svg>
    <span class="label">{{ label() }}</span>`,
  styles: `
    /* Page loading: centered in the screen, above the tab bar, letting clicks through. */
    :host {
      position: fixed;
      inset: 0 0 var(--tabbar-h, 0px);
      z-index: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      color: var(--color-text-muted);
      pointer-events: none;
      animation: loading-in 0.2s ease-out 0.15s both;
    }
    /* Inside a section: in the flow, at the start of the line. */
    :host(.inline) {
      position: static;
      justify-content: flex-start;
      min-height: 44px;
      padding: var(--space-2, 8px) 0;
    }
    .spinner {
      width: 32px;
      height: 32px;
      animation: loading-spin 0.9s linear infinite;
    }
    :host(.inline) .spinner {
      width: 20px;
      height: 20px;
    }
    circle {
      fill: none;
      stroke-width: 3;
    }
    .track {
      stroke: var(--color-track);
    }
    .arc {
      stroke: var(--color-primary);
      stroke-linecap: round;
      stroke-dasharray: 18 57;
    }
    .label {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip-path: inset(50%);
      white-space: nowrap;
    }
    @keyframes loading-spin {
      to {
        transform: rotate(360deg);
      }
    }
    @keyframes loading-in {
      from {
        opacity: 0;
      }
    }
    @media (prefers-reduced-motion: reduce) {
      .spinner {
        display: none;
      }
      .label {
        position: static;
        width: auto;
        height: auto;
        overflow: visible;
        clip-path: none;
        font: var(--text-sm) var(--font-ui);
      }
    }
  `,
  host: { role: 'status' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Loading {
  /** What is loading, read by screen readers. */
  readonly label = input('Đang tải…');
}
