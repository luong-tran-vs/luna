import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { engineLabel, ImageEngine } from '../../../core/models/word-bank';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { AdminApiService } from '../admin-api.service';
import { GrammarImageBatchService } from './grammar-image-batch.service';

/** The batch key of the hint pictures of a level's lessons. */
export function levelHintsKey(level: string): string {
  return `hints:level:${level}`;
}

/** The batch key of the hint pictures of one lesson. */
export function pointHintsKey(pointId: string): string {
  return `hints:point:${pointId}`;
}

/**
 * "Tạo ảnh câu luyện tập" for every grammar lesson of a level (added 2026-11-16), on the grammar list: the
 * server draws, one after another in the background, the picture of each practice choice exercise
 * that has none: its sentence with the right answer, which the learner sees under "Gợi ý". It
 * shares the example picture batch (one runs at a time) and its banner.
 */
@Component({
  selector: 'lu-grammar-level-hints',
  imports: [EngineDialog, RouterLink],
  template: `
    <section class="level-batch" aria-labelledby="level-hints-title">
      <div class="line">
        <div class="text">
          <h3 class="title" id="level-hints-title">Ảnh câu luyện tập</h3>
          <p class="field-hint">
            Ảnh cảnh của câu đúng, hiện trên câu chọn đáp án, điền từ và sửa lỗi ở Luyện tập.
            @if (count(); as c) {
              Đã có {{ c.drawn }}/{{ c.total }} ảnh.
            }
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
            [disabled]="busy() || missing() === 0 || !!other()"
            (click)="open.set(true)"
          >
            Tạo ảnh<span class="visually-hidden"> câu luyện tập {{ level() }}</span
            >{{ missing() ? ' (' + missing() + ' câu)' : '' }}
          </button>
        }
      </div>

      @if (mine(); as b) {
        <div class="status" [class.bad]="!b.running && !!b.stopMessage" role="status">
          @if (b.running) {
            <p>
              <span class="dot" aria-hidden="true"></span>
              Máy chủ đang tạo ảnh câu luyện tập ngầm bằng {{ engineOf(b.engine) }}: xong
              <strong>{{ b.done }}/{{ b.total }}</strong> ảnh của {{ b.points }} bài
              @if (b.failures.length) {
                · {{ b.failures.length }} lỗi
              }
            </p>
            <progress
              [max]="b.total"
              [value]="b.done"
              aria-label="Tiến độ tạo ảnh câu luyện tập"
            ></progress>
            <p class="field-hint">
              Có thể đóng trang hay làm việc khác, ảnh nào vẽ xong là lưu ngay.
            </p>
          } @else {
            <p>
              {{
                b.stopMessage
                  ? 'Tạo ảnh câu luyện tập dừng sớm: ' + b.stopMessage
                  : b.stopped
                    ? 'Đã dừng tạo ảnh câu luyện tập.'
                    : 'Đã tạo ảnh câu luyện tập xong.'
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
                    <a [routerLink]="['/admin/grammar', f.pointId]" lang="en">{{ f.en }}</a>:
                    {{ f.message }}
                  </li>
                }
              </ul>
            }
          }
        </div>
      } @else if (other(); as o) {
        <p class="field-hint">
          Máy chủ đang tạo ảnh ngầm cho {{ o.label }} (xong {{ o.done }}/{{ o.total }}). Chờ lượt đó
          xong hoặc bấm Dừng.
        </p>
      }
      @if (error(); as message) {
        <p class="form-alert" role="alert">{{ message }}</p>
      }
    </section>

    <lu-engine-dialog
      [open]="open()"
      [heading]="'Tạo ảnh câu luyện tập cho các bài ngữ pháp ' + level()"
      [message]="
        'Vẽ ' +
        missing() +
        ' ảnh còn thiếu cho câu chọn đáp án, điền từ và sửa lỗi ở Luyện tập, lần lượt từng ảnh. Mỗi lượt tối đa ' +
        maxBatch +
        ' ảnh.'
      "
      confirmLabel="Tạo ảnh"
      [codexAvailable]="codex()"
      (confirmed)="start($event)"
      (cancelled)="open.set(false)"
    />
  `,
  styleUrl: './grammar-level-batch.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLevelHints {
  private readonly api = inject(AdminApiService);
  protected readonly batches = inject(GrammarImageBatchService);

  readonly level = input.required<string>();
  /** The points of the level that have a lesson. */
  readonly pointIds = input.required<readonly string[]>();

  /** The server draws at most this many pictures in one batch. */
  protected readonly maxBatch = 200;
  protected readonly open = signal(false);
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);
  /** The level's practice choice exercises and how many have their picture; null until counted. */
  protected readonly count = signal<{ total: number; drawn: number } | null>(null);
  protected readonly missing = computed(() => {
    const c = this.count();
    return c ? c.total - c.drawn : 0;
  });
  protected readonly codex = computed(() => this.batches.status()?.codexImages ?? false);

  /** The batch of this level: running, or finished and not hidden yet. */
  protected readonly mine = computed(() => {
    const b = this.batches.status();
    if (!b || b.key !== levelHintsKey(this.level())) {
      return null;
    }
    return b.running ? b : this.batches.result();
  });
  /** Another picture batch runs (one runs at a time). */
  protected readonly other = computed(() => {
    const b = this.batches.status();
    return b?.running && b.key !== levelHintsKey(this.level()) ? b : null;
  });

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
    effect(() => {
      const ids = this.pointIds();
      untracked(() => this.countHints(ids));
    });
    // Counted again once a batch ends, whichever it was.
    let wasRunning = false;
    effect(() => {
      const running = this.batches.running();
      if (wasRunning && !running) {
        untracked(() => this.countHints(this.pointIds()));
      }
      wasRunning = running;
    });
  }

  private countHints(ids: readonly string[]): void {
    if (ids.length === 0) {
      this.count.set({ total: 0, drawn: 0 });
      return;
    }
    this.api.countGrammarHintPictures([...ids]).subscribe({
      next: (c) => {
        if (ids === this.pointIds()) {
          this.count.set(c);
        }
      },
      error: () => this.count.set(null),
    });
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
            hints: true,
            engine,
            key: levelHintsKey(this.level()),
            label: `ảnh câu luyện tập các bài ngữ pháp ${this.level()}`,
          }),
        ),
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu tạo ảnh được, vui lòng thử lại.');
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
export function messageOf(err: unknown): string | null {
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
