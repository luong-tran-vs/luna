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
import { engineLabel, ImageEngine } from '../../../core/models/word-bank';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { AdminApiService } from '../admin-api.service';
import { GrammarImageBatchService, levelBatchKey } from './grammar-image-batch.service';

/**
 * "Tạo ảnh ví dụ" for every grammar lesson of a level (added 2026-11-06), on the grammar list: the
 * admin picks Gemini or Codex, and the server draws the examples without a picture one after
 * another in the background, with its progress here and in the banner of every admin page.
 */
@Component({
  selector: 'lu-grammar-level-images',
  imports: [ConfirmDialog, EngineDialog, RouterLink],
  template: `
    <section class="card level-batch" aria-labelledby="level-images-title">
      <div class="line">
        <div class="text">
          <h2 class="title" id="level-images-title">Ảnh ví dụ</h2>
          <p class="field-hint">
            Vẽ ảnh còn thiếu (câu ví dụ và bài “Nhìn ảnh chọn từ”) của {{ pointIds().length }} bài
            ngữ pháp {{ level() }} vẽ được (có nhãn “Vẽ được”), lần lượt từng ảnh, chạy ngầm trên
            máy chủ.
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
            (click)="open.set(true)"
          >
            Tạo ảnh ví dụ {{ level() }}
          </button>
          <button
            type="button"
            class="btn"
            [disabled]="busy() || pointIds().length === 0 || !!other()"
            (click)="askRemove()"
          >
            Gỡ ảnh ví dụ {{ level() }}
          </button>
        }
      </div>
      @if (note(); as n) {
        <p class="field-hint" role="status">{{ n }}</p>
      }

      @if (mine(); as b) {
        <div class="status" [class.bad]="!b.running && !!b.stopMessage" role="status">
          @if (b.running) {
            <p>
              <span class="dot" aria-hidden="true"></span>
              Máy chủ đang tạo ảnh ví dụ ngầm bằng {{ engineOf(b.engine) }}: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> ảnh của {{ b.points }} bài
              @if (b.drawing.length) {
                · đang vẽ
                @for (d of b.drawing; track d.pointId + d.key; let last = $last) {
                  “<span lang="en">{{ d.en }}</span
                  >”{{ last ? '' : ',' }}
                }
              }
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </p>
            <progress
              [max]="b.total"
              [value]="b.done"
              aria-label="Tiến độ tạo ảnh ví dụ"
            ></progress>
            <p class="field-hint">
              Có thể đóng trang hay làm việc khác, ảnh nào vẽ xong là lưu ngay.
            </p>
          } @else {
            <p>
              {{
                b.stopMessage
                  ? 'Tạo ảnh ví dụ dừng sớm: ' + b.stopMessage
                  : b.stopped
                    ? 'Đã dừng tạo ảnh ví dụ.'
                    : 'Đã tạo ảnh ví dụ xong.'
              }}
              {{ b.saved.length }}/{{ b.total }} ảnh mới bằng {{ engineOf(b.engine) }}
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
              <button type="button" class="btn btn-sm" (click)="batches.hideResult()">Ẩn</button>
            </p>
            @if (b.failures.length) {
              <ul class="failures">
                @for (f of b.failures; track f.pointId + f.key) {
                  <li>
                    <a [routerLink]="['/admin/grammar', f.pointId]" lang="en">{{ f.en }}</a
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
          Máy chủ đang tạo ảnh ví dụ ngầm cho {{ o.label }} (xong {{ o.done }}/{{ o.total }}). Chờ
          lượt đó xong hoặc bấm Dừng.
        </p>
      }
      @if (error(); as message) {
        <p class="form-alert" role="alert">{{ message }}</p>
      }
    </section>

    <lu-engine-dialog
      [open]="open()"
      [heading]="'Tạo ảnh ví dụ cho các bài ngữ pháp ' + level()"
      [message]="
        'Vẽ ảnh còn thiếu của ' +
        pointIds().length +
        ' bài ngữ pháp vẽ được: câu ví dụ và bài “Nhìn ảnh chọn từ”, lần lượt từng ảnh.'
      "
      confirmLabel="Tạo ảnh"
      [codexAvailable]="codex()"
      (confirmed)="start($event)"
      (cancelled)="open.set(false)"
    />

    <lu-confirm-dialog
      [open]="removal() !== null"
      [title]="'Gỡ mọi ảnh ví dụ của ngữ pháp ' + level() + '?'"
      [message]="
        (removal()?.pictures ?? 0) +
        ' ảnh của ' +
        (removal()?.points ?? 0) +
        ' bài sẽ bị gỡ: ảnh câu ví dụ và ảnh bài “Nhìn ảnh chọn từ” (người học không thấy các bài này cho tới khi tạo lại ảnh). Không hoàn tác được.'
      "
      [confirmLabel]="'Gỡ ' + (removal()?.pictures ?? 0) + ' ảnh'"
      [danger]="true"
      (confirmed)="remove()"
      (cancelled)="removal.set(null)"
    />
  `,
  styleUrl: './grammar-level-batch.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLevelImages {
  private readonly api = inject(AdminApiService);
  protected readonly batches = inject(GrammarImageBatchService);

  readonly level = input.required<string>();
  /** The points of the level that have a lesson. */
  readonly pointIds = input.required<readonly string[]>();

  protected readonly open = signal(false);
  /** The pictures "Gỡ ảnh ví dụ" is about to remove; null while its dialog is closed. */
  protected readonly removal = signal<{ pictures: number; points: number } | null>(null);
  protected readonly note = signal<string | null>(null);
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly codex = computed(() => this.batches.status()?.codexImages ?? false);

  /** The batch of this level: running, or finished and not hidden yet. */
  protected readonly mine = computed(() => {
    const b = this.batches.status();
    if (!b || b.key !== levelBatchKey(this.level())) {
      return null;
    }
    return b.running ? b : this.batches.result();
  });
  /** Another picture batch runs (one runs at a time). */
  protected readonly other = computed(() => {
    const b = this.batches.status();
    return b?.running && b.key !== levelBatchKey(this.level()) ? b : null;
  });

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
  }

  protected engineOf(engine: string): string {
    return engineLabel(engine === 'codex' ? 'codex' : 'gemini');
  }

  protected async start(engine: ImageEngine): Promise<void> {
    this.open.set(false);
    if (this.busy()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    try {
      this.batches.set(
        await firstValueFrom(
          this.api.startGrammarImageBatch({
            pointIds: [...this.pointIds()],
            engine,
            key: levelBatchKey(this.level()),
            label: `các bài ngữ pháp ${this.level()}`,
          }),
        ),
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu tạo ảnh được, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }

  /** "Gỡ ảnh ví dụ" (added 2026-11-12): counts the pictures first, so the dialog says what goes. */
  protected async askRemove(): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    this.note.set(null);
    try {
      const n = await firstValueFrom(this.api.removeGrammarImages([...this.pointIds()], true));
      if (n.pictures === 0) {
        this.note.set(`Các bài ngữ pháp ${this.level()} chưa có ảnh nào.`);
      } else {
        this.removal.set(n);
      }
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không đếm được ảnh, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(): Promise<void> {
    this.removal.set(null);
    this.busy.set(true);
    try {
      const n = await firstValueFrom(this.api.removeGrammarImages([...this.pointIds()]));
      this.note.set(`Đã gỡ ${n.pictures} ảnh của ${n.points} bài.`);
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không gỡ được ảnh, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }

  protected async stop(): Promise<void> {
    this.busy.set(true);
    try {
      this.batches.set(await firstValueFrom(this.api.stopGrammarImageBatch()));
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
