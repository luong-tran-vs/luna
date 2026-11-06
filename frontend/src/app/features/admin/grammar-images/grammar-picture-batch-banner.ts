import { ChangeDetectionStrategy, Component, DestroyRef, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { GrammarPictureBatchService } from './grammar-picture-batch.service';

/**
 * The label on every admin page while the server adds picture exercises to grammar lessons in the
 * background (added 2026-11-06), linking to the grammar list; once done, the result until hidden.
 */
@Component({
  selector: 'lu-grammar-picture-batch-banner',
  imports: [RouterLink],
  template: `
    <div role="status">
      @if (batches.status(); as b) {
        @if (b.running) {
          <a class="banner running" routerLink="/admin/grammar">
            <span class="dot" aria-hidden="true"></span>
            <span class="text">
              {{
                b.stopping ? 'Đang dừng thêm bài nhìn ảnh' : 'Máy chủ đang thêm bài nhìn ảnh ngầm'
              }}
              cho <strong>{{ b.label }}</strong
              >: xong <strong>{{ b.done }}/{{ b.total }}</strong> bài
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </span>
            <span class="go">Xem</span>
          </a>
        } @else if (batches.result(); as r) {
          <div class="banner" [class.bad]="!!r.stopMessage">
            <a class="text" routerLink="/admin/grammar">
              {{
                r.stopMessage
                  ? 'Thêm bài nhìn ảnh dừng sớm'
                  : r.stopped
                    ? 'Đã dừng thêm bài nhìn ảnh'
                    : 'Đã thêm bài nhìn ảnh xong'
              }}
              cho <strong>{{ r.label }}</strong
              >: {{ r.saved.length }}/{{ r.total }} bài
              @if (r.failures.length) {
                · {{ r.failures.length }} lỗi
              }
            </a>
            <button type="button" class="hide" (click)="batches.hideResult()">
              Ẩn<span class="visually-hidden"> thông báo thêm bài nhìn ảnh</span>
            </button>
          </div>
        }
      }
    </div>
  `,
  styleUrl: '../word-bank/image-batch-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarPictureBatchBanner {
  protected readonly batches = inject(GrammarPictureBatchService);

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }
}
