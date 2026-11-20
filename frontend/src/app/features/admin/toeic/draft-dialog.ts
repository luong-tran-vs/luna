import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { DraftReport, PhotoBatch, ToeicPart } from '../../../core/models/toeic';
import { ImageEngine } from '../../../core/models/word-bank';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { PART_NAMES } from '../../toeic/shared/score.logic';
import { errorBody, fieldLines } from '../json-import/json-import';
import { DRAFT_PLAN, TOEIC_LEVELS } from './toeic-admin.logic';

const PARTS: ToeicPart[] = [1, 2, 3, 4, 5, 6, 7];

/**
 * "Nhờ AI soạn": the AI writes new groups of one part in one request; the server checks each one like
 * an import and keeps the valid ones as drafts, to preview and publish in Kho cụm câu. For Part 1,
 * 6 and 7 the pictures of the new groups can then be drawn at once ("Vẽ ảnh cho các cụm vừa soạn").
 * Emits true when some drafts were made.
 */
@Component({
  selector: 'lu-toeic-draft-dialog',
  imports: [EngineDialog],
  template: `
    <div class="backdrop">
      <div
        #dialog
        class="dialog wide"
        role="dialog"
        aria-modal="true"
        aria-labelledby="draft-title"
        tabindex="-1"
      >
        <h2 class="title" id="draft-title">Nhờ AI soạn cụm câu</h2>
        <p class="muted">
          AI viết cụm mới theo đúng định dạng nhập; cụm nào qua được kiểm tra thì lưu thành nháp.
          Hãy xem trước kỹ (đáp án, giải thích) rồi mới đăng.
        </p>
        @if (!report()) {
          <div class="fields">
            <label>
              Part
              <select (change)="setPart($event)" [disabled]="busy()">
                @for (p of parts; track p) {
                  <option [value]="p" [selected]="p === part()">
                    Part {{ p }} · {{ names[p] }}
                  </option>
                }
              </select>
            </label>
            <label>
              Mức (điểm)
              <select (change)="setLevel($event)" [disabled]="busy()">
                @for (l of levels; track l) {
                  <option [value]="l" [selected]="l === level()">{{ l }}</option>
                }
              </select>
            </label>
            <label class="grow">
              Gợi ý cho AI (không bắt buộc)
              <input
                type="text"
                maxlength="200"
                [disabled]="busy()"
                [value]="topic()"
                (input)="setTopic($event)"
                placeholder="vd: giới từ chỉ thời gian, khách sạn, có biểu đồ…"
              />
            </label>
            <p class="plan">
              Mỗi lần AI soạn: <b>{{ plan[part()] }}</b>
            </p>
          </div>
          @if (busy()) {
            <p class="muted" role="status">
              AI đang soạn Part {{ part() }} ({{ plan[part()] }})… có thể mất 1–3 phút.
            </p>
          }
        } @else {
          @let r = report()!;
          <p role="status">
            <b>Đã lưu {{ r.created.length }} cụm nháp</b>
            @if (duplicates() > 0) {
              · bỏ {{ duplicates() }} cụm trùng với cụm đã có
            }
            @if (r.dropped.length - duplicates() > 0) {
              · bỏ {{ r.dropped.length - duplicates() }} cụm không qua kiểm tra
            }
          </p>
          @if (r.created.length) {
            <p class="muted">
              @for (k of r.created; track k; let last = $last) {
                <code>{{ k }}</code>{{ last ? '' : ', ' }}
              }
            </p>
          }
          @if (canDraw()) {
            <div class="pictures">
              @if (batch(); as b) {
                <p class="progress" role="status">{{ batchText() }}</p>
                @if (b.running) {
                  <p class="muted">
                    Có thể đóng hộp thoại: ảnh vẫn vẽ tiếp, xem tiến độ ở khu Ảnh minh hoạ.
                  </p>
                  <button
                    type="button"
                    class="btn small"
                    [disabled]="b.stopping"
                    (click)="stopDrawing()"
                  >
                    {{ b.stopping ? 'Đang dừng…' : 'Dừng vẽ' }}
                  </button>
                } @else if (b.saved.length < b.total) {
                  <button type="button" class="btn small" (click)="asking.set(true)">
                    Vẽ tiếp ảnh còn thiếu…
                  </button>
                }
              } @else {
                <p class="muted">{{ pictureHint() }}</p>
                <button type="button" class="btn small" (click)="asking.set(true)">
                  Vẽ ảnh cho {{ r.created.length }} cụm vừa soạn…
                </button>
              }
            </div>
          }
          @if (r.dropped.length) {
            <ul class="dropped">
              @for (d of r.dropped; track d.key) {
                <li>
                  Cụm {{ $index + 1 }}:
                  @for (f of lines(d.fields); track f.key) {
                    @if (f.key === 'duplicate') {
                      <span>{{ f.text }}</span>
                    } @else {
                      <span
                        ><code>{{ f.key }}</code> {{ f.text }}</span
                      >
                    }
                  }
                </li>
              }
            </ul>
          }
        }
        @if (error()) {
          <p class="form-alert" role="alert">{{ error() }}</p>
        }
        <div class="actions">
          @if (report()) {
            <button type="button" class="btn" (click)="again()">Soạn tiếp</button>
            <button type="button" class="btn-primary" (click)="closed.emit(made())">
              Xem các cụm nháp
            </button>
          } @else {
            <button type="button" class="btn" [disabled]="busy()" (click)="closed.emit(made())">
              Huỷ
            </button>
            <button type="button" class="btn-primary" [disabled]="busy()" (click)="draft()">
              {{ busy() ? 'Đang soạn…' : 'Soạn bằng AI' }}
            </button>
          }
        </div>
      </div>
    </div>
    <lu-engine-dialog
      [open]="asking()"
      heading="Vẽ ảnh cho các cụm vừa soạn"
      [message]="
        'Vẽ lần lượt ảnh còn thiếu của ' +
        (report()?.created?.length ?? 0) +
        ' cụm vừa soạn, chạy ngầm trên máy chủ.'
      "
      [codexAvailable]="codex()"
      confirmLabel="Bắt đầu vẽ"
      (confirmed)="draw($event)"
      (cancelled)="asking.set(false)"
    />
  `,
  styleUrls: ['../../../shared/components/confirm-dialog/confirm-dialog.css', './toeic-admin.css'],
  styles: `
    .fields {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-3);
    }
    .fields label {
      display: grid;
      gap: var(--space-1);
      font: var(--text-sm) var(--font-ui);
    }
    .fields .grow {
      flex: 1 1 100%;
    }
    input[type='text'] {
      min-height: 40px;
      padding: 0 var(--space-2);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }
    .plan {
      margin: 0;
      font: var(--text-sm) var(--font-ui);
    }
    .pictures {
      display: grid;
      justify-items: start;
      gap: var(--space-2);
      padding: var(--space-3);
      background: var(--color-bg);
      border-radius: var(--radius-md);
    }
    .pictures p {
      margin: 0;
    }
    .progress {
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
    }
    .dropped li {
      display: grid;
      gap: 2px;
      font: var(--text-sm) var(--font-ui);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'escape()' },
})
export class DraftDialog {
  private readonly api = inject(ToeicApiService);

  readonly closed = output<boolean>();

  protected readonly parts = PARTS;
  protected readonly levels = TOEIC_LEVELS;
  protected readonly names = PART_NAMES;
  protected readonly part = signal<ToeicPart>(5);
  protected readonly level = signal(600);
  protected readonly topic = signal('');
  protected readonly busy = signal(false);
  protected readonly error = signal('');
  protected readonly report = signal<DraftReport | null>(null);
  /** How many groups were dropped because the part already has them (or the answer had them twice). */
  protected readonly duplicates = computed(
    () => this.report()?.dropped.filter((d) => d.fields?.['duplicate']).length ?? 0,
  );
  protected readonly made = signal(false);
  protected readonly plan = DRAFT_PLAN;

  // --- pictures of the new groups (Part 1, 6, 7) ---
  protected readonly asking = signal(false);
  protected readonly codex = signal(false);
  protected readonly batch = signal<PhotoBatch | null>(null);
  private timer: ReturnType<typeof setTimeout> | null = null;
  /** The part the report is of (the select may change after drafting). */
  private readonly draftedPart = signal<ToeicPart>(5);

  protected readonly canDraw = computed(
    () => (this.report()?.created.length ?? 0) > 0 && [1, 6, 7].includes(this.draftedPart()),
  );
  protected readonly pictureHint = computed(() =>
    this.draftedPart() === 1
      ? 'Mỗi câu Part 1 cần một ảnh chụp: vẽ luôn cho các cụm vừa soạn.'
      : 'Vẽ ảnh minh hoạ cho từng văn bản của các cụm vừa soạn (email, memo, chat… được vẽ cảnh của tình huống).',
  );
  protected readonly batchText = computed(() => {
    const b = this.batch();
    if (!b) {
      return '';
    }
    if (b.total === 0) {
      return 'Không có ảnh nào cần vẽ: các văn bản vừa soạn không thuộc loại có hình (thêm tay ở Xem trước nếu cần).';
    }
    const failed = Object.keys(b.failed).length;
    let text = b.running
      ? `Đang vẽ ảnh ${b.saved.length + failed + 1}/${b.total} (${b.engine})…`
      : `Đã vẽ ${b.saved.length}/${b.total} ảnh.`;
    if (failed) {
      text += ` ${failed} ảnh lỗi.`;
    }
    if (b.stopReason) {
      text += ` Dừng vì: ${b.stopReason}`;
    }
    return text;
  });

  protected draw(engine: ImageEngine): void {
    this.asking.set(false);
    const keys = this.report()?.created ?? [];
    this.error.set('');
    this.api.startPhotoBatch({ keys }, engine).subscribe({
      next: (b) => {
        this.batch.set(b);
        this.schedule();
      },
      error: (err: unknown) => this.error.set(errorBody(err).message ?? 'Chưa bắt đầu vẽ được.'),
    });
  }

  protected stopDrawing(): void {
    this.api.stopPhotoBatch().subscribe({ next: (b) => this.batch.set(b), error: () => undefined });
  }

  private poll(): void {
    this.api.photoBatch().subscribe({
      next: (r) => {
        this.codex.set(r.codex);
        const mine = this.batch();
        // Only this dialog's batch: the one started for these groups.
        if (mine && r.batch.keys?.join() === mine.keys?.join()) {
          this.batch.set(r.batch);
          if (r.batch.running) {
            this.schedule();
          }
        }
      },
      error: () => undefined,
    });
  }

  private schedule(): void {
    this.clearTimer();
    this.timer = setTimeout(() => this.poll(), 2000);
  }

  private clearTimer(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    afterNextRender(() => this.dialog()?.nativeElement.querySelector('select')?.focus());
    inject(DestroyRef).onDestroy(() => this.clearTimer());
  }

  protected setPart(event: Event): void {
    this.part.set(Number((event.target as HTMLSelectElement).value) as ToeicPart);
  }

  protected setLevel(event: Event): void {
    this.level.set(Number((event.target as HTMLSelectElement).value));
  }

  protected setTopic(event: Event): void {
    this.topic.set((event.target as HTMLInputElement).value);
  }

  protected lines(fields?: Record<string, string>): { key: string; text: string }[] {
    return fields ? fieldLines(fields) : [];
  }

  protected escape(): void {
    if (this.asking()) {
      return; // the engine dialog handles its own Escape
    }
    if (!this.busy()) {
      this.closed.emit(this.made());
    }
  }

  protected again(): void {
    this.report.set(null);
    this.batch.set(null);
    this.clearTimer();
    this.error.set('');
  }

  protected draft(): void {
    this.busy.set(true);
    this.error.set('');
    this.api
      .draftGroups({ part: this.part(), level: this.level(), topic: this.topic().trim() })
      .subscribe({
        next: (r) => {
          this.busy.set(false);
          this.report.set(r);
          this.draftedPart.set(this.part());
          this.batch.set(null);
          this.poll();
          if (r.created.length) {
            this.made.set(true);
          }
        },
        error: (err: unknown) => {
          this.busy.set(false);
          this.error.set(errorBody(err).message ?? 'AI soạn thất bại, vui lòng thử lại.');
        },
      });
  }
}
