import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { AppUpdateService } from '../../../core/services/app-update.service';

/**
 * "Đã có phiên bản mới": shown at the top of every page once the service worker has downloaded a
 * new build (AppUpdateService). Cập nhật reloads into it; Để sau hides the bar until the next
 * visit (the app still updates by itself on the next page change).
 */
@Component({
  selector: 'lu-update-banner',
  template: `
    @if (shown()) {
      <div class="bar" role="status">
        <p class="text">Đã có phiên bản mới của Luna.</p>
        <button type="button" class="btn-primary update" (click)="updates.update()">
          Cập nhật
        </button>
        <button type="button" class="btn later" (click)="dismissed.set(true)">Để sau</button>
      </div>
    }
  `,
  styles: `
    .bar {
      position: fixed;
      top: calc(var(--space-2) + env(safe-area-inset-top, 0px));
      left: 50%;
      z-index: 100;
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: var(--space-2);
      width: max-content;
      max-width: calc(100vw - 2 * var(--space-4));
      padding: var(--space-2) var(--space-3);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-left: var(--space-1) solid var(--color-primary);
      border-radius: var(--radius-md);
      box-shadow: 0 6px 24px rgb(0 0 0 / 15%);
      transform: translateX(-50%);
    }
    .text {
      flex: 1;
      margin: 0;
      font-weight: 600;
    }
    .update,
    .later {
      min-height: 40px;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UpdateBanner {
  protected readonly updates = inject(AppUpdateService);
  protected readonly dismissed = signal(false);
  protected readonly shown = computed(() => this.updates.ready() && !this.dismissed());
}
