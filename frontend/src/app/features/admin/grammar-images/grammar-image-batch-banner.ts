import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { engineLabel } from '../../../core/models/word-bank';
import { GrammarImageBatchService } from './grammar-image-batch.service';

/**
 * The label on every admin page while the server draws grammar example pictures in the background
 * (added 2026-11-06): how far it got, with a link to where it was started; once done, the result
 * until hidden.
 */
@Component({
  selector: 'lu-grammar-image-batch-banner',
  imports: [RouterLink],
  template: `
    <div role="status">
      @if (batches.status(); as b) {
        @if (b.running) {
          <a class="banner running" [routerLink]="link()">
            <span class="dot" aria-hidden="true"></span>
            <span class="text">
              {{ b.stopping ? 'Đang dừng tạo ảnh ví dụ' : 'Máy chủ đang tạo ảnh ví dụ ngầm' }} cho
              <strong>{{ b.label }}</strong> bằng {{ engine() }}: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> ảnh
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </span>
            <span class="go">Xem</span>
          </a>
        } @else if (batches.result(); as r) {
          <div class="banner" [class.bad]="!!r.stopMessage">
            <a class="text" [routerLink]="link()">
              {{
                r.stopMessage
                  ? 'Tạo ảnh ví dụ dừng sớm'
                  : r.stopped
                    ? 'Đã dừng tạo ảnh ví dụ'
                    : 'Đã tạo ảnh ví dụ xong'
              }}
              cho <strong>{{ r.label }}</strong
              >: {{ r.saved.length }}/{{ r.total }} ảnh
              @if (r.failures.length) {
                · {{ r.failures.length }} lỗi
              }
            </a>
            <button type="button" class="hide" (click)="batches.hideResult()">
              Ẩn<span class="visually-hidden"> thông báo tạo ảnh ví dụ</span>
            </button>
          </div>
        }
      }
    </div>
  `,
  styleUrl: '../word-bank/image-batch-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarImageBatchBanner {
  protected readonly batches = inject(GrammarImageBatchService);
  protected readonly engine = computed(() =>
    engineLabel(this.batches.status()?.engine === 'codex' ? 'codex' : 'gemini'),
  );
  /** The lesson's page for a lesson's batch, the grammar list for a level's. */
  protected readonly link = computed(() => {
    const key = this.batches.status()?.key ?? '';
    return key.startsWith('point:')
      ? ['/admin/grammar', key.slice('point:'.length)]
      : ['/admin/grammar'];
  });

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }
}
