import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { engineLabel } from '../../../core/models/word-bank';
import { lessonOfKey, levelOfKey, SceneBatchService } from './scene-batch.service';

/**
 * The label on every admin page while the server draws scene pictures in the background (added
 * 2026-11-06): how far it got, with a link to the course page; once done, the result until hidden.
 */
@Component({
  selector: 'lu-scene-batch-banner',
  imports: [RouterLink],
  template: `
    <div role="status">
      @if (batches.status(); as b) {
        @if (b.running) {
          <a class="banner running" [routerLink]="link()" [queryParams]="query()">
            <span class="dot" aria-hidden="true"></span>
            <span class="text">
              {{ b.stopping ? 'Đang dừng tạo ảnh cảnh' : 'Máy chủ đang tạo ảnh cảnh ngầm' }} cho
              <strong>{{ b.label }}</strong> bằng {{ engine() }}: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> bài
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </span>
            <span class="go">Xem</span>
          </a>
        } @else if (batches.result(); as r) {
          <div class="banner" [class.bad]="!!r.stopMessage">
            <a class="text" [routerLink]="link()" [queryParams]="query()">
              {{
                r.stopMessage
                  ? 'Tạo ảnh cảnh dừng sớm'
                  : r.stopped
                    ? 'Đã dừng tạo ảnh cảnh'
                    : 'Đã tạo ảnh cảnh xong'
              }}
              cho <strong>{{ r.label }}</strong
              >: {{ r.saved.length }}/{{ r.total }} bài
              @if (r.failures.length) {
                · {{ r.failures.length }} lỗi
              }
            </a>
            <button type="button" class="hide" (click)="batches.hideResult()">
              Ẩn<span class="visually-hidden"> thông báo tạo ảnh cảnh</span>
            </button>
          </div>
        }
      }
    </div>
  `,
  styleUrl: '../word-bank/image-batch-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SceneBatchBanner {
  protected readonly batches = inject(SceneBatchService);
  protected readonly engine = computed(() =>
    engineLabel(this.batches.status()?.engine === 'codex' ? 'codex' : 'gemini'),
  );
  /** Where the batch was started: the lesson's page, or the course page of its level. */
  protected readonly link = computed(() => {
    const id = lessonOfKey(this.batches.status()?.key ?? '');
    return id ? ['/admin/lessons', id] : ['/admin/curriculum'];
  });
  protected readonly query = computed(() => {
    const level = levelOfKey(this.batches.status()?.key ?? '');
    return level ? { level } : {};
  });

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }
}
