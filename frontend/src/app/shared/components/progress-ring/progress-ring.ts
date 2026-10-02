import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

const RADIUS = 42;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

/** A round progress meter with the percentage in the middle; the label names it for screen readers. */
@Component({
  selector: 'lu-progress-ring',
  template: `
    <div
      class="ring"
      role="progressbar"
      [attr.aria-label]="label()"
      aria-valuemin="0"
      aria-valuemax="100"
      [attr.aria-valuenow]="percent()"
      [attr.aria-valuetext]="text()"
    >
      <svg viewBox="0 0 100 100" aria-hidden="true">
        <circle class="track" cx="50" cy="50" [attr.r]="radius" />
        <circle
          class="fill"
          cx="50"
          cy="50"
          [attr.r]="radius"
          [attr.stroke-dasharray]="circumference"
          [attr.stroke-dashoffset]="offset()"
          [style.--ring-empty]="circumference"
        />
      </svg>
      <span class="value" aria-hidden="true">{{ text() }}</span>
    </div>
  `,
  styles: `
    :host {
      display: inline-grid;
      flex: none;
      width: var(--ring-size, 72px);
      height: var(--ring-size, 72px);
    }
    .ring {
      position: relative;
      display: grid;
      place-items: center;
    }
    svg {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
      transform: rotate(-90deg);
    }
    circle {
      fill: none;
      stroke-width: 9;
    }
    .track {
      stroke: var(--color-track);
    }
    /* Fills up from empty when shown, then slides to new values (off with reduced motion, styles/motion.css). */
    .fill {
      stroke: var(--ring-color, var(--color-primary));
      stroke-linecap: round;
      transition: stroke-dashoffset var(--motion-slow, 400ms) var(--ease-out, ease-out);
      animation: ring-fill 700ms var(--ease-out, ease-out) both;
    }
    @keyframes ring-fill {
      from {
        stroke-dashoffset: var(--ring-empty);
      }
    }
    .value {
      font: var(--text-sm) var(--font-ui);
      font-weight: 700;
      font-variant-numeric: tabular-nums;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProgressRing {
  /** 0–100, or null when there is nothing to measure yet ("—"). */
  readonly value = input.required<number | null>();
  readonly label = input.required<string>();

  protected readonly radius = RADIUS;
  protected readonly circumference = CIRCUMFERENCE;
  protected readonly percent = computed(() => {
    const v = this.value();
    return v === null ? 0 : Math.round(Math.min(100, Math.max(0, v)));
  });
  protected readonly text = computed(() => (this.value() === null ? '—' : `${this.percent()}%`));
  protected readonly offset = computed(() => CIRCUMFERENCE * (1 - this.percent() / 100));
}
