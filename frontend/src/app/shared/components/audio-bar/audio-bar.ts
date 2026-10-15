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
 * is an estimate from the text length (see estimateSeconds). While the audio is being prepared
 * (natural voice, browser voice off) the button shows a spinner and the time says so.
 */
@Component({
  selector: 'lu-audio-bar',
  imports: [Icon],
  template: `
    <button
      type="button"
      class="play"
      [class.preparing]="preparing()"
      [attr.aria-busy]="preparing() || null"
      [attr.aria-label]="playing() ? stopLabel() : playLabel()"
      [disabled]="disabled()"
      (click)="playPressed.emit()"
    >
      <lu-icon [name]="playing() ? 'pause' : 'play'" />
    </button>
    <div class="meter">
      <!--
        Moved with transforms, updated on every frame while playing: no layout work, so the bar glides
        even when it is long. Transitions only smooth a change while stopped (the jump to the end).
      -->
      <div class="track-box">
        <div class="track" aria-hidden="true" [class.animate]="!playing() && share() > 0">
          <div class="played" [style.transform]="'scaleX(' + share() + ')'"></div>
          <div class="head" [style.transform]="'translateX(' + share() * 100 + '%)'">
            <span class="knob"></span>
          </div>
        </div>
      </div>
      <span class="time">
        @if (preparing()) {
          <span aria-hidden="true">Đang chuẩn bị…</span>
        } @else {
          <span class="visually-hidden">Đã nghe </span><span>{{ elapsed() }}</span
          ><span class="sep"> / </span><span>{{ total() }}</span>
        }
      </span>
    </div>
    <!-- Said once, not on every tick of the time. -->
    <span class="visually-hidden" role="status">{{ preparing() ? 'Đang chuẩn bị giọng đọc' : '' }}</span>
  `,
  styles: `
    :host {
      display: flex;
      align-items: center;
      gap: var(--space-3);
      min-width: 0;
    }
    /* Track and time side by side; in the large player the time goes under the track. */
    .meter {
      display: contents;
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
    /*
     * Clips what the head moves past the track (it is as wide as the track, so near the end it
     * would reach far past the screen and widen the page), keeping 8px around for the knob.
     * The margin and padding cancel out: the track stays where it was.
     */
    .track-box {
      flex: 1;
      min-width: 48px;
      margin: -8px;
      padding: 8px;
      overflow: hidden;
    }
    .track {
      position: relative;
      height: 6px;
      background: var(--color-track);
      border-radius: var(--radius-full);
    }
    .played {
      height: 100%;
      background: var(--color-primary);
      border-radius: inherit;
      transform-origin: left center;
      will-change: transform;
    }
    /* As wide as the track: translateX(n%) puts its left edge, the knob's centre, at n% of the track. */
    .head {
      position: absolute;
      inset: 0;
      pointer-events: none;
      will-change: transform;
    }
    .knob {
      position: absolute;
      top: 50%;
      left: 0;
      width: 14px;
      height: 14px;
      background: var(--color-surface);
      border: 3px solid var(--color-primary);
      border-radius: var(--radius-full);
      transform: translate(-50%, -50%);
    }
    .animate .played,
    .animate .head {
      transition: transform 0.15s ease-out;
    }
    .time {
      flex: none;
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
      font-variant-numeric: tabular-nums;
    }
    /* Large (design, Bước Nghe): a rounded square button, the time at both ends under the track. */
    :host(.large) {
      gap: var(--space-4);
      padding: var(--space-3);
      background: var(--color-bg);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-lg);
    }
    :host(.large) .play {
      width: 56px;
      height: 56px;
      background: linear-gradient(135deg, var(--color-primary), var(--color-primary-ink));
      border-radius: var(--radius-lg);
      box-shadow: 0 6px 14px color-mix(in srgb, var(--color-primary) 30%, transparent);

      --icon-size: 22px;
    }
    :host(.large) .meter {
      display: grid;
      flex: 1;
      gap: var(--space-2);
      min-width: 0;
    }
    :host(.large) .time {
      display: flex;
      justify-content: space-between;
      font-size: var(--text-xs);
    }
    :host(.large) .time > :nth-child(2) {
      color: var(--color-primary-ink);
      font-weight: 600;
    }
    :host(.large) .sep {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip-path: inset(50%);
      white-space: nowrap;
    }
    @media (prefers-reduced-motion: reduce) {
      .animate .played,
      .animate .head {
        transition: none;
      }
    }
  `,
  host: { '[class.large]': "size() === 'large'" },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AudioBar {
  /** "large": the bigger player of the Listening step. */
  readonly size = input<'normal' | 'large'>('normal');
  /** Part already played, from 0 to 1. */
  readonly progress = input(0);
  /** Length of the audio in seconds. */
  readonly duration = input(0);
  readonly playing = input(false);
  /** The audio is being generated before it plays. */
  readonly preparing = input(false);
  readonly disabled = input(false);
  readonly playLabel = input('Phát');
  readonly stopLabel = input('Dừng');

  /** The button: play when stopped, stop when playing. */
  readonly playPressed = output<void>();

  protected readonly share = computed(() => Math.min(1, Math.max(0, this.progress())));
  protected readonly elapsed = computed(() => clock(this.share() * this.duration()));
  protected readonly total = computed(() => clock(this.duration()));
}
