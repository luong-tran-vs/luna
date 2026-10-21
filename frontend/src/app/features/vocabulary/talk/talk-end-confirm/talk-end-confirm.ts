import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  input,
  output,
  viewChild,
} from '@angular/core';

/**
 * Asks before ending a conversation while turns are left (added 2026-10-21): Kết thúc sits next to
 * the history button and is easy to hit by mistake. Nói tiếp, Escape or the backdrop go back.
 */
@Component({
  selector: 'lu-talk-end-confirm',
  template: `
    <div class="backdrop" (click)="cancelled.emit()" aria-hidden="true"></div>
    <div
      class="confirm"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="end-title"
      aria-describedby="end-desc"
      (keydown.escape)="cancelled.emit()"
    >
      <h2 id="end-title">Kết thúc buổi trò chuyện?</h2>
      <p id="end-desc">
        Bạn mới nói {{ said() }}/{{ maxTurns() }} lượt. Leo sẽ nhận xét dựa trên {{ said() }} câu
        này.
      </p>
      <button type="button" class="btn-primary" (click)="confirmed.emit()">
        Kết thúc và xem nhận xét
      </button>
      <button #keepTalking type="button" class="btn" (click)="cancelled.emit()">Nói tiếp</button>
    </div>
  `,
  styles: `
    .backdrop {
      position: fixed;
      inset: 0;
      z-index: 20;
      background: color-mix(in srgb, var(--color-text) 40%, transparent);
    }
    .confirm {
      position: fixed;
      top: 50%;
      left: 50%;
      z-index: 21;
      display: grid;
      gap: var(--space-2);
      width: min(360px, calc(100% - 2 * var(--space-5)));
      padding: var(--space-5);
      background: var(--color-surface);
      border-radius: 20px;
      transform: translate(-50%, -50%);
    }
    h2 {
      font: var(--text-lg) var(--font-ui);
    }
    p {
      margin-bottom: var(--space-2);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkEndConfirm {
  private readonly keepTalking = viewChild<ElementRef<HTMLButtonElement>>('keepTalking');

  /** Lines said so far, out of maxTurns. */
  readonly said = input.required<number>();
  readonly maxTurns = input.required<number>();
  readonly confirmed = output();
  readonly cancelled = output();

  constructor() {
    afterNextRender(() => this.keepTalking()?.nativeElement.focus());
  }
}
