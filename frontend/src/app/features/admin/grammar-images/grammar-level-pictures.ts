import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { PICTURE_MASTERY, PICTURE_PRACTICE } from '../../../core/models/grammar-admin';
import { AdminApiService } from '../admin-api.service';
import { levelBatchKey } from './grammar-image-batch.service';
import { GrammarPictureBatchService } from './grammar-picture-batch.service';

/**
 * "Thêm bài nhìn ảnh" for every grammar lesson of a level (added 2026-11-06), on the grammar list:
 * the server has the AI write the missing "Nhìn ảnh chọn từ" exercises of each lesson that can be
 * drawn, one lesson after another in the background, keeping the rest of every lesson.
 */
@Component({
  selector: 'lu-grammar-level-pictures',
  imports: [RouterLink],
  template: `
    <section class="card level-batch" aria-labelledby="level-pictures-title">
      <div class="line">
        <div class="text">
          <h2 class="title" id="level-pictures-title">Bài nhìn ảnh</h2>
          <p class="field-hint">
            Thêm cho đủ {{ practice }} bài “Nhìn ảnh chọn từ” ở Luyện tập và {{ mastery }} bài ở
            Kiểm tra vào {{ pointIds().length }} bài ngữ pháp {{ level() }} vẽ được, lần lượt từng
            bài, chạy ngầm trên máy chủ. Phần còn lại của bài giữ nguyên. Thêm xong thì tạo ảnh ví
            dụ để người học thấy các bài này.
          </p>
        </div>
        @if (mine(); as b) {
          @if (b.running) {
            <button type="button" class="btn" [disabled]="b.stopping || busy()" (click)="stop()">
              {{ b.stopping ? 'Đang dừng…' : 'Dừng' }}
            </button>
          }
        } @else {
          <button
            type="button"
            class="btn"
            [disabled]="busy() || pointIds().length === 0 || !!other()"
            (click)="start()"
          >
            {{ busy() ? 'Đang gửi…' : 'Thêm bài nhìn ảnh ' + level() }}
          </button>
        }
      </div>

      @if (mine(); as b) {
        <div class="status" [class.bad]="!b.running && !!b.stopMessage" role="status">
          @if (b.running) {
            <p>
              <span class="dot" aria-hidden="true"></span>
              Máy chủ đang thêm bài nhìn ảnh ngầm: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> bài
              @if (b.current; as c) {
                · đang soạn “{{ c.title }}”
              }
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </p>
            <progress
              [max]="b.total"
              [value]="b.done"
              aria-label="Tiến độ thêm bài nhìn ảnh"
            ></progress>
            <p class="field-hint">
              Có thể đóng trang hay làm việc khác, bài nào soạn xong là lưu ngay.
            </p>
          } @else {
            <p>
              {{
                b.stopMessage
                  ? 'Thêm bài nhìn ảnh dừng sớm: ' + b.stopMessage
                  : b.stopped
                    ? 'Đã dừng thêm bài nhìn ảnh.'
                    : 'Đã thêm bài nhìn ảnh xong.'
              }}
              {{ b.saved.length }}/{{ b.total }} bài, {{ added(b.saved) }} bài tập mới
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
              <button type="button" class="btn btn-sm" (click)="batches.hideResult()">Ẩn</button>
            </p>
            @if (b.failures.length) {
              <ul class="failures">
                @for (f of b.failures; track f.pointId) {
                  <li>
                    <a [routerLink]="['/admin/grammar', f.pointId]">{{ f.title }}</a
                    >:
                    {{ f.message }}
                  </li>
                }
              </ul>
            }
          }
        </div>
      } @else if (other(); as o) {
        <p class="field-hint">
          Máy chủ đang thêm bài nhìn ảnh ngầm cho {{ o.label }} (xong {{ o.done }}/{{ o.total }}).
          Chờ lượt đó xong hoặc bấm Dừng.
        </p>
      }
      @if (error(); as message) {
        <p class="form-alert" role="alert">{{ message }}</p>
      }
    </section>
  `,
  styleUrl: './grammar-level-batch.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLevelPictures {
  private readonly api = inject(AdminApiService);
  protected readonly batches = inject(GrammarPictureBatchService);

  readonly level = input.required<string>();
  /** The points of the level that can be drawn and have a lesson. */
  readonly pointIds = input.required<readonly string[]>();

  protected readonly practice = PICTURE_PRACTICE;
  protected readonly mastery = PICTURE_MASTERY;
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  /** The batch of this level: running, or finished and not hidden yet. */
  protected readonly mine = computed(() => {
    const b = this.batches.status();
    if (!b || b.key !== levelBatchKey(this.level())) {
      return null;
    }
    return b.running ? b : this.batches.result();
  });
  /** Another batch runs (one runs at a time). */
  protected readonly other = computed(() => {
    const b = this.batches.status();
    return b?.running && b.key !== levelBatchKey(this.level()) ? b : null;
  });

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }

  protected added(saved: readonly { added: number }[]): number {
    return saved.reduce((n, s) => n + s.added, 0);
  }

  protected async start(): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    try {
      this.batches.set(
        await firstValueFrom(
          this.api.startGrammarPictureBatch({
            pointIds: [...this.pointIds()],
            key: levelBatchKey(this.level()),
            label: `các bài ngữ pháp ${this.level()}`,
          }),
        ),
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu thêm bài nhìn ảnh được, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }

  protected async stop(): Promise<void> {
    this.busy.set(true);
    try {
      this.batches.set(await firstValueFrom(this.api.stopGrammarPictureBatch()));
    } catch {
      // The next status read shows where it is.
    } finally {
      this.busy.set(false);
    }
  }
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field = body?.fields ? Object.values(body.fields)[0] : undefined;
    if (field) {
      return field;
    }
    return typeof body?.message === 'string' && body.message ? body.message : null;
  }
  return null;
}
