import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  input,
  output,
  signal,
  untracked,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { serverUrl } from '../../../core/api-url';
import { GrammarHintPicture } from '../../../core/models/grammar-admin';
import { engineLabel, ImageEngine } from '../../../core/models/word-bank';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { AdminApiService } from '../admin-api.service';
import { GrammarImageBatchService } from './grammar-image-batch.service';
import { messageOf, pointHintsKey } from './grammar-level-hints';

/**
 * "Ảnh câu" of one grammar lesson (added 2026-11-16), on its admin page beside the practice: each
 * practice choice exercise may have a picture of its sentence with the right answer, which the
 * learner sees under "Gợi ý". The admin draws the missing ones in one batch, or one again, or
 * removes one. Every point has them, drawable or not.
 */
@Component({
  selector: 'lu-grammar-hint-pictures',
  imports: [EngineDialog],
  template: `
    <section class="card pictures" aria-labelledby="pv-hint-images" [attr.aria-busy]="drawingNow()">
      <div class="head">
        <h3 class="part-title" id="pv-hint-images">Ảnh câu luyện tập</h3>
        @if (items(); as list) {
          <span class="muted">{{ list.length - missing() }}/{{ list.length }} ảnh đã có</span>
        }
      </div>
      <p class="field-hint">
        Mỗi câu chọn đáp án, điền từ và sửa lỗi ở Luyện tập có thể có một ảnh vẽ cảnh của câu đúng,
        hiện ngay trên câu để người học dễ hiểu tình huống. Sửa câu hay đổi đáp án thì cần tạo lại
        ảnh.
      </p>

      @if (loadError()) {
        <div class="form-alert" role="alert">
          <span>Không tải được danh sách ảnh câu.</span>
          <button type="button" class="btn" (click)="load()">Thử lại</button>
        </div>
      } @else if (items(); as list) {
        @if (list.length === 0) {
          <p class="muted">Phần Luyện tập chưa có câu chọn đáp án, điền từ hay sửa lỗi nào.</p>
        } @else {
          <div class="actions">
            @if (drawingNow()) {
              <p class="bg-note" role="status">
                <span class="dot" aria-hidden="true"></span>
                <span>
                  Máy chủ đang tạo ảnh câu ngầm: xong
                  <strong>{{ batch()?.done }}/{{ batch()?.total }}</strong> ảnh. Có thể rời trang.
                </span>
              </p>
              <button type="button" class="btn" [disabled]="!!batch()?.stopping" (click)="stop()">
                {{ batch()?.stopping ? 'Đang dừng…' : 'Dừng' }}
              </button>
            } @else if (missing() > 0) {
              <button type="button" class="btn-primary" [disabled]="locked()" (click)="ask({})">
                {{ busy() === 'all' ? 'Đang gửi…' : 'Tạo ' + missing() + ' ảnh câu còn thiếu' }}
              </button>
            }
          </div>
          @if (otherBatch(); as o) {
            <p class="field-hint">
              Máy chủ đang tạo ảnh ngầm cho {{ o.label }} (xong {{ o.done }}/{{ o.total }}). Chờ
              lượt đó xong hoặc bấm Dừng.
            </p>
          }
          @if (note(); as n) {
            <p class="muted" role="status">{{ n }}</p>
          }
          @if (error(); as message) {
            <p class="form-alert" role="alert">{{ message }}</p>
          }

          <ul class="rows">
            @for (h of list; track h.exerciseId) {
              <li class="row" [class.drawing]="drawingKeys().has(h.key)">
                <div class="thumb">
                  @if (h.imageUrl) {
                    <img
                      [src]="thumb(h)"
                      crossorigin="use-credentials"
                      [alt]="'Ảnh câu của câu: ' + h.sentence"
                    />
                  } @else if (drawingKeys().has(h.key)) {
                    <span class="thumb-empty"
                      ><span class="spinner" aria-hidden="true"></span>Đang vẽ</span
                    >
                  } @else {
                    <span class="thumb-empty">Chưa có ảnh</span>
                  }
                </div>
                <div class="text">
                  <span class="ex-tag">Bài tập {{ h.exerciseId }} · {{ kindLabel(h.kind) }}</span>
                  <span lang="en">{{ h.sentence }}</span>
                </div>
                @if (h.imageUrl) {
                  <div class="row-actions">
                    <button
                      type="button"
                      class="btn"
                      [disabled]="locked()"
                      (click)="ask({ keys: [h.key], replace: true }, h.exerciseId)"
                    >
                      {{ busy() === h.exerciseId ? 'Đang gửi…' : 'Vẽ lại'
                      }}<span class="visually-hidden"> ảnh của câu {{ h.sentence }}</span>
                    </button>
                    <button
                      type="button"
                      class="btn"
                      [disabled]="locked()"
                      [attr.aria-label]="'Xoá ảnh câu của câu ' + h.sentence"
                      (click)="remove(h)"
                    >
                      Xoá
                    </button>
                  </div>
                }
              </li>
            }
          </ul>
        }
      }
    </section>

    <lu-engine-dialog
      [open]="pending() !== null"
      heading="Tạo ảnh câu luyện tập"
      [message]="
        pending()?.keys
          ? 'Vẽ lại ảnh câu của câu này.'
          : 'Vẽ ' + missing() + ' ảnh câu còn thiếu của bài, lần lượt từng ảnh.'
      "
      confirmLabel="Tạo ảnh"
      [codexAvailable]="codex()"
      (confirmed)="start($event)"
      (cancelled)="pending.set(null)"
    />
  `,
  styleUrl: './grammar-example-images.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarHintPictures {
  private readonly api = inject(AdminApiService);
  private readonly batches = inject(GrammarImageBatchService);

  readonly pointId = input.required<string>();
  readonly title = input('');
  /** A picture was drawn or removed: the page reads the lesson again. */
  readonly reload = output<void>();

  protected readonly items = signal<GrammarHintPicture[] | null>(null);
  protected readonly loadError = signal(false);
  protected readonly busy = signal<string | null>(null);
  protected readonly note = signal<string | null>(null);
  protected readonly error = signal<string | null>(null);
  /** The batch waiting for the engine dialog: all missing, or one picture again. */
  protected readonly pending = signal<{ keys?: string[]; replace?: boolean; busy: string } | null>(
    null,
  );
  /** Changes when pictures are drawn, so the browser asks for them again under the same URL. */
  private readonly version = signal(Date.now());

  protected readonly missing = computed(
    () => (this.items() ?? []).filter((h) => !h.imageUrl).length,
  );
  protected readonly codex = computed(() => !!this.batches.status()?.codexImages);
  private readonly batchKey = computed(() => pointHintsKey(this.pointId()));
  protected readonly drawingNow = computed(() => {
    const b = this.batches.status();
    return !!b?.running && b.key === this.batchKey();
  });
  protected readonly drawingKeys = computed(
    () =>
      new Set(this.drawingNow() ? (this.batches.status()?.drawing ?? []).map((d) => d.key) : []),
  );
  protected readonly batch = computed(() => (this.drawingNow() ? this.batches.status() : null));
  protected readonly otherBatch = computed(() => {
    const b = this.batches.status();
    return b?.running && b.key !== this.batchKey() ? b : null;
  });
  protected readonly locked = computed(
    () => this.busy() !== null || this.drawingNow() || !!this.otherBatch(),
  );

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
    effect(() => {
      this.pointId();
      untracked(() => this.load());
    });
    let wasDrawing = false;
    effect(() => {
      const drawing = this.drawingNow();
      if (wasDrawing && !drawing) {
        untracked(() => this.finished());
      }
      wasDrawing = drawing;
    });
  }

  protected load(): void {
    this.loadError.set(false);
    this.api.grammarHintPictures(this.pointId()).subscribe({
      next: (r) => this.items.set(r.items),
      error: () => this.loadError.set(true),
    });
  }

  protected kindLabel(kind: string): string {
    return kind === 'fill' ? 'điền từ' : kind === 'fix' ? 'sửa lỗi' : 'chọn đáp án';
  }

  protected thumb(h: GrammarHintPicture): string {
    return `${serverUrl(h.imageUrl ?? '')}?v=${this.version()}`;
  }

  protected ask(extra: { keys?: string[]; replace?: boolean }, busy = 'all'): void {
    if (!this.locked()) {
      this.pending.set({ ...extra, busy });
    }
  }

  protected async start(engine: ImageEngine): Promise<void> {
    const p = this.pending();
    this.pending.set(null);
    if (!p || this.locked()) {
      return;
    }
    this.busy.set(p.busy);
    this.error.set(null);
    this.note.set(null);
    try {
      this.batches.set(
        await firstValueFrom(
          this.api.startGrammarImageBatch({
            pointIds: [this.pointId()],
            hints: true,
            keys: p.keys,
            replace: p.replace,
            engine,
            key: this.batchKey(),
            label: `ảnh câu bài “${this.title() || this.pointId()}”`,
          }),
        ),
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu tạo ảnh được, vui lòng thử lại.');
    } finally {
      this.busy.set(null);
    }
  }

  protected async remove(h: GrammarHintPicture): Promise<void> {
    if (this.locked()) {
      return;
    }
    this.busy.set(h.exerciseId);
    this.error.set(null);
    this.note.set(null);
    try {
      await firstValueFrom(this.api.deleteGrammarImage(this.pointId(), h.key));
      this.items.update((list) =>
        (list ?? []).map((i) => (i.key === h.key ? { ...i, imageUrl: undefined } : i)),
      );
      this.note.set('Đã xoá ảnh câu.');
      this.reload.emit();
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không xoá được ảnh, vui lòng thử lại.');
    } finally {
      this.busy.set(null);
    }
  }

  private finished(): void {
    const b = this.batches.status();
    this.version.set(Date.now());
    this.load();
    this.reload.emit();
    if (!b) {
      return;
    }
    const engine = engineLabel(b.engine === 'codex' ? 'codex' : 'gemini');
    const failed = b.failures.length ? `, ${b.failures.length} câu lỗi` : '';
    if (b.saved.length > 0) {
      this.note.set(`Đã tạo ${b.saved.length}/${b.total} ảnh câu bằng ${engine}${failed}.`);
    }
    if (b.saved.length === 0 || b.stopMessage) {
      this.error.set(
        b.stopMessage ??
          b.failures[0]?.message ??
          (b.stopped ? 'Đã dừng tạo ảnh.' : 'Sinh ảnh thất bại, vui lòng thử lại.'),
      );
    }
  }

  protected async stop(): Promise<void> {
    try {
      this.batches.set(await firstValueFrom(this.api.stopGrammarImageBatch()));
    } catch {
      // The next status read shows where it is.
    }
  }
}
