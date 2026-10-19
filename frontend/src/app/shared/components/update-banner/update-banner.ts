import {
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  inject,
  viewChild,
} from '@angular/core';

import { AppUpdateService } from '../../../core/services/app-update.service';

/**
 * "Đang tải bản mới…": a small pill at the top while a new version downloads, which leaves the page
 * usable (design/tat-ca-de-xuat.png (mục thong-bao-cap-nhat)).
 *
 * "Đã có phiên bản mới": a dialog in the middle of the screen once the service worker has
 * downloaded a new build (AppUpdateService). A native <dialog> opened with showModal(), so the page
 * behind it cannot be used; it only offers Cập nhật, which reloads into the new version, and
 * Escape does not close it.
 */
@Component({
  selector: 'lu-update-banner',
  template: `
    <p class="pill" role="status">
      @if (updates.downloading()) {
        <span class="spinner" aria-hidden="true"></span>
        Đang tải bản mới…
        <span class="track" aria-hidden="true"><span class="run"></span></span>
      }
    </p>
    <dialog
      #dialog
      class="dialog"
      aria-labelledby="update-title"
      aria-describedby="update-text"
      (cancel)="$event.preventDefault()"
    >
      <h2 class="title" id="update-title">Đã có phiên bản mới</h2>
      <p class="text" id="update-text">Cập nhật để dùng bản mới nhất của Luna.</p>
      <button type="button" class="btn-primary update" autofocus (click)="updates.update()">
        Cập nhật
      </button>
    </dialog>
  `,
  styles: `
    .pill {
      position: fixed;
      top: calc(env(safe-area-inset-top, 0px) + var(--space-3));
      left: 50%;
      z-index: 20;
      display: flex;
      align-items: center;
      gap: var(--space-2);
      margin: 0;
      padding: var(--space-2) var(--space-4) calc(var(--space-2) + 2px);
      overflow: hidden;
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      white-space: nowrap;
      color: var(--color-surface);
      background: var(--color-text);
      border-radius: var(--radius-full);
      box-shadow: var(--shadow-card);
      transform: translateX(-50%);
      pointer-events: none;
      animation: drop 0.25s ease-out;
    }
    .pill:empty {
      display: none;
    }
    .spinner {
      flex: none;
      width: 14px;
      height: 14px;
      border: 2px solid color-mix(in srgb, var(--color-surface) 30%, transparent);
      border-top-color: var(--color-surface);
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }
    .track {
      position: absolute;
      right: var(--space-4);
      bottom: 4px;
      left: var(--space-4);
      height: 2px;
      overflow: hidden;
      background: color-mix(in srgb, var(--color-surface) 20%, transparent);
      border-radius: 2px;
    }
    .run {
      display: block;
      width: 35%;
      height: 100%;
      background: var(--color-primary-soft);
      border-radius: inherit;
      animation: run 1.2s ease-in-out infinite;
    }
    @keyframes drop {
      from {
        opacity: 0;
        transform: translate(-50%, -12px);
      }
    }
    @keyframes spin {
      to {
        transform: rotate(1turn);
      }
    }
    @keyframes run {
      from {
        transform: translateX(-100%);
      }
      to {
        transform: translateX(290%);
      }
    }
    @media (prefers-reduced-motion: reduce) {
      .pill,
      .spinner,
      .run {
        animation: none;
      }
    }
    .dialog {
      width: min(360px, calc(100vw - 32px));
      padding: var(--space-5) var(--space-4) var(--space-4);
      color: var(--color-text);
      text-align: center;
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-lg);
    }
    .dialog::backdrop {
      background: color-mix(in srgb, var(--color-text) 45%, transparent);
    }
    .title {
      margin: 0 0 var(--space-2);
      font: var(--text-lg) var(--font-ui);
    }
    .text {
      margin: 0 0 var(--space-4);
    }
    .update {
      width: 100%;
      min-height: 44px;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UpdateBanner {
  protected readonly updates = inject(AppUpdateService);
  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');

  constructor() {
    effect(() => {
      const el = this.dialog().nativeElement;
      if (this.updates.ready() && !el.open) {
        el.showModal();
      }
    });
  }
}
