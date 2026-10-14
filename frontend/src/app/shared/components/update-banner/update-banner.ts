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
 * "Đã có phiên bản mới": a dialog in the middle of the screen once the service worker has
 * downloaded a new build (AppUpdateService). A native <dialog> opened with showModal(), so the page
 * behind it cannot be used; it only offers Cập nhật, which reloads into the new version, and
 * Escape does not close it.
 */
@Component({
  selector: 'lu-update-banner',
  template: `
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
