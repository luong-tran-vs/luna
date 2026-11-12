import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
  untracked,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { CardWordsSummary, ImageEngine, MAX_IMAGE_BATCH } from '../../../../core/models/word-bank';
import { EngineDialog } from '../../../../shared/components/engine-dialog/engine-dialog';
import { AdminApiService } from '../../admin-api.service';
import { bankMessage } from '../bank-message';
import { ImageBatchService } from '../image-batch.service';

/** How many of the words lacking a picture the section names. */
const NAMED = 12;

/**
 * "Ảnh cho thẻ ôn" on the word bank page (added 2026-11-12): the review cards show the word bank
 * picture of their word, so the admin sees how many of the words learners saved have one and
 * draws the missing ones in one batch, the most saved first. A word not in the bank yet is added
 * with a learner's meaning. The batch is the word bank's: the page shows its progress.
 */
@Component({
  selector: 'lu-card-word-pictures',
  imports: [EngineDialog],
  template: `
    <section class="card card-words" aria-labelledby="card-words-heading">
      <h2 class="section-title" id="card-words-heading">Ảnh cho thẻ ôn của học viên</h2>
      @if (summary(); as s) {
        @if (s.total === 0) {
          <p class="muted">Học viên chưa lưu từ nào vào thẻ ôn.</p>
        } @else {
          <p class="count">
            Có ảnh <strong>{{ s.withImage }}/{{ s.total }}</strong> từ học viên đã lưu
          </p>
          @if (s.missingCount > 0) {
            <p class="muted">
              Lưu nhiều nhất mà chưa có ảnh:
              @for (w of named(); track w.lemma; let last = $last) {
                <span lang="en">{{ w.lemma }}</span>
                ({{ w.learners }}){{ w.inBank ? '' : '*' }}{{ last ? '' : ',' }}
              }
              @if (s.missingCount > named().length) {
                và {{ s.missingCount - named().length }} từ khác
              }
            </p>
            @if (notInBank() > 0) {
              <p class="muted hint">
                * Chưa có trong Kho: sẽ được thêm vào Kho với nghĩa lấy từ thẻ của học viên.
              </p>
            }
            <button
              type="button"
              class="btn-primary"
              [disabled]="busy() || batches.running()"
              [attr.title]="
                batches.running() ? 'Đang có một lượt sinh ảnh chạy, chờ xong hoặc bấm Dừng.' : null
              "
              (click)="asking.set(true)"
            >
              Sinh ảnh còn thiếu ({{ s.missingCount }})
            </button>
          } @else {
            <p class="muted">Mọi từ trong thẻ ôn đã có ảnh.</p>
          }
        }
      } @else if (loadError()) {
        <p class="form-alert" role="alert">
          Không tải được số từ trong thẻ ôn.
          <button type="button" class="btn btn-sm" (click)="load()">Thử lại</button>
        </p>
      } @else {
        <p class="muted">Đang tải…</p>
      }
      @if (error(); as e) {
        <p class="form-alert" role="alert">{{ e }}</p>
      }
    </section>

    <lu-engine-dialog
      [open]="asking()"
      heading="Sinh ảnh cho từ trong thẻ ôn"
      [message]="message()"
      [note]="note()"
      [confirmLabel]="'Sinh ' + drawCount() + ' ảnh'"
      [codexAvailable]="summary()?.codexImages ?? false"
      (confirmed)="start($event)"
      (cancelled)="asking.set(false)"
    />
  `,
  styles: `
    .card-words {
      display: grid;
      gap: var(--space-2);
      justify-items: start;
    }
    .count {
      font: var(--text-base) var(--font-ui);
    }
    .hint {
      font: var(--text-sm) var(--font-ui);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CardWordPictures {
  private readonly api = inject(AdminApiService);
  protected readonly batches = inject(ImageBatchService);

  protected readonly summary = signal<CardWordsSummary | null>(null);
  protected readonly loadError = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly asking = signal(false);
  protected readonly busy = signal(false);

  protected readonly named = computed(() => this.summary()?.missing.slice(0, NAMED) ?? []);
  protected readonly notInBank = computed(() => this.named().filter((w) => !w.inBank).length);
  protected readonly drawCount = computed(() =>
    Math.min(this.summary()?.missingCount ?? 0, MAX_IMAGE_BATCH),
  );
  protected readonly message = computed(
    () =>
      `Máy chủ vẽ ${this.drawCount()} từ được lưu nhiều nhất mà chưa có ảnh, chạy ngầm, xong ảnh nào lưu ảnh đó. Từ chưa có trong Kho được thêm vào Kho với nghĩa từ thẻ của học viên.`,
  );
  protected readonly note = computed(() => {
    const rest = (this.summary()?.missingCount ?? 0) - this.drawCount();
    return rest > 0
      ? `Mỗi lượt tối đa ${MAX_IMAGE_BATCH} ảnh: ${rest} từ còn lại vẽ ở lượt sau.`
      : '';
  });

  constructor() {
    this.load();
    // Counted again when a batch ends: it may have drawn some of these words.
    let wasRunning = false;
    effect(() => {
      const running = this.batches.running();
      if (wasRunning && !running) {
        untracked(() => this.load());
      }
      wasRunning = running;
    });
  }

  protected load(): void {
    this.api.cardWords().subscribe({
      next: (s) => {
        this.summary.set(s);
        this.loadError.set(false);
      },
      error: () => this.loadError.set(true),
    });
  }

  protected async start(engine: ImageEngine): Promise<void> {
    this.asking.set(false);
    if (this.busy()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    try {
      this.batches.set(await firstValueFrom(this.api.startCardWordsBatch(engine)));
    } catch (err) {
      this.error.set(bankMessage(err) ?? 'Không bắt đầu được, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }
}
