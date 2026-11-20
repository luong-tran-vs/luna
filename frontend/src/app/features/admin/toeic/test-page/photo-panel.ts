import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';

import { PhotoBatch } from '../../../../core/models/toeic';
import { ImageEngine } from '../../../../core/models/word-bank';
import { ToeicApiService } from '../../../../core/services/toeic-api.service';
import { EngineDialog } from '../../../../shared/components/engine-dialog/engine-dialog';
import { errorBody } from '../../json-import/json-import';
import { ImageSlots } from '../image-slots';

/** How often the batch's progress is asked for while it runs. */
const POLL_MS = 2000;

/**
 * The Part 1 photos of a test (FR-036, FR-039): each one drawn, redrawn or uploaded on its own, and
 * "Vẽ N ảnh còn thiếu" that draws the missing ones in the background, with its progress and "Dừng".
 */
@Component({
  selector: 'lu-toeic-photo-panel',
  imports: [ImageSlots, EngineDialog],
  template: `
    <div class="head">
      <p class="muted">
        Mỗi câu Part 1 cần một ảnh chụp; đề không đăng được khi còn thiếu ảnh. Mỗi lần vẽ đều phải
        chọn Gemini hoặc Codex.
      </p>
      @if (batch()?.running) {
        <button type="button" class="btn small" [disabled]="batch()?.stopping" (click)="stop()">
          {{ batch()?.stopping ? 'Đang dừng…' : 'Dừng' }}
        </button>
      } @else if (missing() > 0) {
        <button type="button" class="btn-primary small" (click)="asking.set(true)">
          Vẽ {{ missing() }} ảnh còn thiếu…
        </button>
      }
    </div>
    @if (batch(); as b) {
      @if (b.running || b.saved.length || b.stopReason) {
        <p class="progress" role="status">
          @if (b.running) {
            Đang vẽ ảnh {{ b.saved.length + failedCount(b) + 1 }}/{{ b.total }} ({{ b.engine }})…
          } @else {
            Đã vẽ {{ b.saved.length }}/{{ b.total }} ảnh.
          }
          @if (failedCount(b)) {
            {{ failedCount(b) }} ảnh lỗi.
          }
          @if (b.stopReason) {
            Dừng vì: {{ b.stopReason }}
          }
        </p>
      }
    }
    @if (error()) {
      <p class="form-alert" role="alert">{{ error() }}</p>
    }
    <ul class="groups">
      @for (k of groupKeys(); track k + '-' + round()) {
        <li>
          <p class="key">
            <code>{{ k }}</code>
          </p>
          <lu-toeic-image-slots [groupKey]="k" (changed)="changed.emit()" />
        </li>
      }
    </ul>
    <lu-engine-dialog
      [open]="asking()"
      heading="Vẽ ảnh Part 1 còn thiếu"
      [message]="'Vẽ lần lượt ' + missing() + ' ảnh còn thiếu của đề này, chạy ngầm trên máy chủ.'"
      [codexAvailable]="codex()"
      confirmLabel="Bắt đầu vẽ"
      (confirmed)="start($event)"
      (cancelled)="asking.set(false)"
    />
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-3);
    }
    .head {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-3);
      align-items: center;
      justify-content: space-between;
    }
    .head .muted {
      flex: 1 1 280px;
      margin: 0;
    }
    .progress {
      margin: 0;
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
    }
    .groups {
      display: grid;
      gap: var(--space-3);
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .groups li {
      display: grid;
      gap: var(--space-1);
      padding-top: var(--space-3);
      border-top: 1px solid var(--color-line);
    }
    .key {
      margin: 0;
    }
    .small {
      min-height: 36px;
      padding: 0 var(--space-3);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PhotoPanel implements OnInit {
  readonly testKey = input.required<string>();
  readonly groupKeys = input.required<string[]>();
  readonly missing = input(0);
  /** Emitted when a photo was saved, so the test's check is reloaded. */
  readonly changed = output<void>();

  private readonly api = inject(ToeicApiService);
  protected readonly batch = signal<PhotoBatch | null>(null);
  protected readonly codex = signal(false);
  protected readonly asking = signal(false);
  protected readonly error = signal('');
  /** Goes up when a batch ends, so the slots reload their pictures. */
  protected readonly round = signal(0);
  private timer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.clearTimer());
  }

  ngOnInit(): void {
    this.poll();
  }

  protected failedCount(b: PhotoBatch): number {
    return Object.keys(b.failed).length;
  }

  protected start(engine: ImageEngine): void {
    this.asking.set(false);
    this.error.set('');
    this.api.startPhotoBatch({ testKey: this.testKey() }, engine).subscribe({
      next: (b) => {
        this.batch.set(b);
        this.schedule();
      },
      error: (err: unknown) => this.error.set(errorBody(err).message ?? 'Chưa bắt đầu vẽ được.'),
    });
  }

  protected stop(): void {
    this.api.stopPhotoBatch().subscribe({ next: (b) => this.batch.set(b), error: () => undefined });
  }

  private poll(): void {
    this.api.photoBatch().subscribe({
      next: (r) => {
        const was = this.batch();
        // Another test's batch is not this page's.
        const mine =
          r.batch.testKey === this.testKey() && (r.batch.running || r.batch.total > 0)
            ? r.batch
            : null;
        this.batch.set(mine);
        this.codex.set(r.codex);
        if (mine && was && mine.saved.length !== was.saved.length) {
          this.changed.emit();
        }
        if (mine?.running) {
          this.schedule();
        } else if (was?.running) {
          this.round.update((n) => n + 1);
          this.changed.emit();
        }
      },
      error: () => undefined,
    });
  }

  private schedule(): void {
    this.clearTimer();
    this.timer = setTimeout(() => this.poll(), POLL_MS);
  }

  private clearTimer(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }
}
