import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { batchScope, ImageBatchService } from './image-batch.service';

/**
 * The label on every admin page while the server draws word pictures in the background (added
 * 2026-11-04): how far it got, with a link to the word bank; once done, the result until hidden.
 */
@Component({
  selector: 'lu-image-batch-banner',
  imports: [RouterLink],
  template: `
    <div role="status">
      @if (batches.status(); as b) {
        @if (b.running) {
          <a class="banner running" routerLink="/admin/words">
            <span class="dot" aria-hidden="true"></span>
            <span class="text">
              {{ b.stopping ? 'Đang dừng sinh ảnh ngầm' : 'Máy chủ đang sinh ảnh ngầm' }} cho
              <strong>{{ scope() }}</strong> bằng {{ engine() }}: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> từ, còn
              <strong>{{ b.total - b.done }}</strong> từ
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </span>
            <span class="go">Xem</span>
          </a>
        } @else if (batches.result(); as r) {
          <div class="banner" [class.bad]="!!r.stopMessage">
            <a class="text" routerLink="/admin/words">
              {{
                r.stopMessage
                  ? 'Sinh ảnh ngầm dừng sớm'
                  : r.stopped
                    ? 'Đã dừng sinh ảnh ngầm'
                    : 'Đã sinh ảnh ngầm xong'
              }}
              cho <strong>{{ scope() }}</strong
              >: {{ r.saved.length }}/{{ r.total }} ảnh
              @if (r.failures.length) {
                · {{ r.failures.length }} lỗi
              }
            </a>
            <button type="button" class="hide" (click)="batches.hideResult()">
              Ẩn<span class="visually-hidden"> thông báo sinh ảnh</span>
            </button>
          </div>
        }
      }
    </div>
  `,
  styleUrl: './image-batch-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ImageBatchBanner {
  protected readonly batches = inject(ImageBatchService);
  protected readonly scope = computed(() => {
    const b = this.batches.status();
    return b ? batchScope(b) : '';
  });
  protected readonly engine = computed(() =>
    this.batches.status()?.engine === 'codex' ? 'Codex CLI' : 'Gemini',
  );

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }
}
