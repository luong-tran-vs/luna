import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { Icon } from '../icon/icon';

/** `mm:ss` for a number of seconds, rounded down. */
export function clock(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds));
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
}

/**
 * Audio player bar: a play/stop button, the played part of the track with a knob, and the time
 * as "00:03 / 00:12". The browser voice cannot seek, so the track only shows progress; the time
 * is an estimate from the text length (see estimateSeconds).
 */
@Component({
  selector: 'lu-audio-bar',
  imports: [Icon],
  template: `
    <button
      type="button"
      class="play"
      [attr.aria-label]="playing() ? stopLabel() : playLabel()"
      [disabled]="disabled()"
      (click)="playPressed.emit()"
    >
      <lu-icon [name]="playing() ? 'pause' : 'play'" />
    </button>
    <div class="track" aria-hidden="true">
      <div class="played" [class.restart]="share() === 0" [style.width.%]="share() * 100"></div>
      <span class="knob" [class.restart]="share() === 0" [style.left.%]="share() * 100"></span>
    </div>
    <span class="time">
      <span class="visually-hidden">Đã nghe </span>{{ elapsed() }} / {{ total() }}
    </span>
  `,
  styles: `
    :host {
      display: flex;
      align-items: center;
      gap: var(--space-3);
      min-width: 0;
    }
    .play {
      display: inline-flex;
      flex: none;
      align-items: center;
      justify-content: center;
      width: 44px;
      height: 44px;
      padding: 0;
      color: var(--color-on-primary);
      background: var(--color-primary);
      border: 0;
      border-radius: var(--radius-full);
      cursor: pointer;

      --icon-size: 18px;
    }
    .play:disabled {
      color: var(--color-text-muted);
      background: var(--color-track);
      cursor: not-allowed;
    }
    .track {
      position: relative;
      flex: 1;
      min-width: 48px;
      height: 6px;
      background: var(--color-track);
      border-radius: var(--radius-full);
    }
    .played {
      height: 100%;
      background: var(--color-primary);
      border-radius: inherit;
      transition: width 0.2s linear;
    }
    .knob {
      position: absolute;
      top: 50%;
      width: 14px;
      height: 14px;
      background: var(--color-surface);
      border: 3px solid var(--color-primary);
      border-radius: var(--radius-full);
      transform: translate(-50%, -50%);
      transition: left 0.2s linear;
    }
    .restart {
      transition: none;
    }
    .time {
      flex: none;
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
      font-variant-numeric: tabular-nums;
    }
    @media (prefers-reduced-motion: reduce) {
      .played,
      .knob {
        transition: none;
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AudioBar {
  /** Part already played, from 0 to 1. */
  readonly progress = input(0);
  /** Length of the audio in seconds. */
  readonly duration = input(0);
  readonly playing = input(false);
  readonly disabled = input(false);
  readonly playLabel = input('Phát');
  readonly stopLabel = input('Dừng');

  /** The button: play when stopped, stop when playing. */
  readonly playPressed = output<void>();

  protected readonly share = computed(() => Math.min(1, Math.max(0, this.progress())));
  protected readonly elapsed = computed(() => clock(this.share() * this.duration()));
  protected readonly total = computed(() => clock(this.duration()));
}
