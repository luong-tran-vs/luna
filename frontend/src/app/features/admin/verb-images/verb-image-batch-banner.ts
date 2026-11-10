import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { engineLabel } from '../../../core/models/word-bank';
import { VerbImageBatchService } from './verb-image-batch.service';

/**
 * The label on every admin page while the server draws verb pictures in the background (added
 * 2026-11-10): how far it got, with a link to the page; once done, the result until hidden.
 */
@Component({
  selector: 'lu-verb-image-batch-banner',
  imports: [RouterLink],
  template: `
    <div role="status">
      @if (batches.status(); as b) {
        @if (b.running) {
          <a class="banner running" routerLink="/admin/verb-images">
            <span class="dot" aria-hidden="true"></span>
            <span class="text">
              {{ b.stopping ? 'Đang dừng tạo ảnh động từ' : 'Máy chủ đang tạo ảnh động từ ngầm' }}
              bằng {{ engine() }}: xong <strong>{{ b.done }}/{{ b.total }}</strong> ảnh
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </span>
            <span class="go">Xem</span>
          </a>
        } @else if (batches.result(); as r) {
          <div class="banner" [class.bad]="!!r.stopMessage">
            <a class="text" routerLink="/admin/verb-images">
              {{
                r.stopMessage
                  ? 'Tạo ảnh động từ dừng sớm'
                  : r.stopped
                    ? 'Đã dừng tạo ảnh động từ'
                    : 'Đã tạo ảnh động từ xong'
              }}: {{ r.saved.length }}/{{ r.total }} ảnh
              @if (r.failures.length) {
                · {{ r.failures.length }} lỗi
              }
            </a>
            <button type="button" class="hide" (click)="batches.hideResult()">
              Ẩn<span class="visually-hidden"> thông báo tạo ảnh động từ</span>
            </button>
          </div>
        }
      }
    </div>
  `,
  styleUrl: '../word-bank/image-batch-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbImageBatchBanner {
  protected readonly batches = inject(VerbImageBatchService);
  protected readonly engine = computed(() =>
    engineLabel(this.batches.status()?.engine === 'codex' ? 'codex' : 'gemini'),
  );

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }
}
