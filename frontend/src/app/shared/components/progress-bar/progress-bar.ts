import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

export type ProgressTone = 'primary' | 'read' | 'listen' | 'write';

/** A labelled progress bar with "value/max unit" text, readable by screen readers (F6). */
@Component({
  selector: 'lu-progress-bar',
  template: `
    <div class="head">
      <span class="label">{{ label() }}</span>
      <span class="count">{{ value() }}/{{ max() }}{{ unit() ? ' ' + unit() : '' }}</span>
    </div>
    <div
      class="track"
      role="progressbar"
      [attr.aria-label]="label()"
      aria-valuemin="0"
      [attr.aria-valuemax]="max()"
      [attr.aria-valuenow]="value()"
      [attr.aria-valuetext]="value() + '/' + max() + (unit() ? ' ' + unit() : '')"
    >
      <div class="fill" [style.width.%]="percent()"></div>
    </div>
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-1);
      --fill: var(--color-primary);
    }
    :host([data-tone='read']) {
      --fill: var(--color-skill-read);
    }
    :host([data-tone='listen']) {
      --fill: var(--color-skill-listen);
    }
    :host([data-tone='write']) {
      --fill: var(--color-skill-write);
    }
    .head {
      display: flex;
      justify-content: space-between;
      gap: var(--space-2);
      font: var(--text-sm) var(--font-ui);
    }
    .label {
      overflow-wrap: anywhere;
    }
    .count {
      flex: none;
      color: var(--color-text-muted);
      font-variant-numeric: tabular-nums;
    }
    .track {
      height: var(--bar-height, 8px);
      overflow: hidden;
      background: var(--color-track);
      border-radius: 999px;
    }
    .fill {
      height: 100%;
      background: var(--fill);
      border-radius: inherit;
    }
  `,
  host: { '[attr.data-tone]': 'tone()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProgressBar {
  readonly label = input.required<string>();
  readonly value = input.required<number>();
  readonly max = input.required<number>();
  /** Shown after the count, e.g. "bài" or "bước". */
  readonly unit = input('');
  readonly tone = input<ProgressTone>('primary');

  protected readonly percent = computed(() => {
    const max = this.max();
    return max > 0 ? Math.min(100, Math.max(0, (this.value() / max) * 100)) : 0;
  });
}
