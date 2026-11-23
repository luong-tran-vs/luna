import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  OnInit,
  output,
  signal,
} from '@angular/core';

import { MissingPictures, PhotoBatch } from '../../../../core/models/toeic';
import { ImageEngine } from '../../../../core/models/word-bank';
import { ToeicApiService } from '../../../../core/services/toeic-api.service';
import { EngineDialog } from '../../../../shared/components/engine-dialog/engine-dialog';
import { errorBody } from '../../json-import/json-import';
import { LevelDialog } from './level-dialog';

/** How often the batch's progress is asked for while it runs. */
const POLL_MS = 2000;

/** The parts whose groups have pictures, and what a batch draws in each. */
export const PICTURE_PARTS: readonly { part: 1 | 6 | 7; name: string; what: string }[] = [
  { part: 1, name: 'Part 1 · Mô tả tranh', what: 'ảnh chụp của mỗi câu (bắt buộc để đăng đề)' },
  {
    part: 6,
    name: 'Part 6 · Điền đoạn văn',
    what: 'một ảnh minh hoạ cho mỗi văn bản (email, memo, chat… vẽ cảnh của tình huống)',
  },
  {
    part: 7,
    name: 'Part 7 · Đọc hiểu',
    what: 'một ảnh minh hoạ cho mỗi văn bản (email, memo, chat… vẽ cảnh của tình huống)',
  },
];

/**
 * "Ảnh minh hoạ" in Kho cụm câu: for each part with pictures, how many are missing and "Vẽ ảnh
 * còn thiếu…", which asks for the levels (a level with every picture is told so), then the engine
 * (Gemini or Codex, chosen every time), and draws them one after another in the background, with
 * the progress and "Dừng". Every Part 6/7 text gets
 * a picture: an email, a memo or a chat one of its situation.
 */
@Component({
  selector: 'lu-toeic-bank-pictures',
  imports: [EngineDialog, LevelDialog],
  template: `
    <section class="card" aria-labelledby="bank-pictures-title">
      <div class="head">
        <h2 class="section-title" id="bank-pictures-title">Ảnh minh hoạ</h2>
        @if (batch()?.running) {
          <button type="button" class="btn small" [disabled]="batch()?.stopping" (click)="stop()">
            {{ batch()?.stopping ? 'Đang dừng…' : 'Dừng vẽ' }}
          </button>
        }
      </div>
      <ul class="rows">
        @for (p of parts; track p.part) {
          <li>
            <div class="what">
              <b>{{ p.name }}</b>
              <span class="muted">{{ p.what }}</span>
            </div>
            <span class="count" [class.zero]="missingOf(p.part) === 0">
              {{
                missing()
                  ? missingOf(p.part) === 0
                    ? '✓ Đủ ảnh'
                    : 'Còn ' + missingOf(p.part) + ' ảnh'
                  : '…'
              }}
            </span>
            <button
              type="button"
              class="btn small"
              [disabled]="!missing() || !!batch()?.running"
              (click)="openLevels(p.part)"
            >
              Vẽ ảnh còn thiếu…
            </button>
          </li>
        }
      </ul>
      @if (progress(); as text) {
        <p class="progress" role="status">{{ text }}</p>
      }
      @if (error()) {
        <p class="form-alert" role="alert">{{ error() }}</p>
      }
    </section>
    <lu-toeic-level-dialog
      [open]="picking() !== null"
      [heading]="'Vẽ ảnh còn thiếu · Part ' + picking()"
      [missing]="levelsOf(picking())"
      [selected]="chosen()"
      (confirmed)="chooseLevels($event)"
      (cancelled)="picking.set(null)"
    />
    <lu-engine-dialog
      [open]="asking() !== null"
      [heading]="'Vẽ ảnh còn thiếu · Part ' + asking()"
      [message]="dialogMessage()"
      [codexAvailable]="codex()"
      confirmLabel="Bắt đầu vẽ"
      backLabel="Quay lại"
      (confirmed)="start($event)"
      (back)="backToLevels()"
      (cancelled)="asking.set(null)"
    />
  `,
  styleUrls: ['../toeic-admin.css'],
  styles: `
    .head {
      display: flex;
      gap: var(--space-3);
      align-items: center;
      justify-content: space-between;
    }
    .rows {
      display: grid;
      gap: var(--space-2);
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .rows li {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-2) var(--space-3);
      align-items: center;
      padding: var(--space-2) 0;
      border-top: 1px solid var(--color-line);
    }
    .rows li:first-child {
      border-top: 0;
    }
    .what {
      display: grid;
      flex: 1 1 260px;
      gap: 2px;
      font: var(--text-sm) var(--font-ui);
    }
    .count {
      min-width: 90px;
      font: var(--text-sm) var(--font-ui);
      font-weight: 700;
    }
    .count.zero {
      color: var(--color-ok);
    }
    .progress {
      margin: 0;
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BankPictures implements OnInit {
  /** Emitted when pictures were drawn, so the list shows them. */
  readonly changed = output<void>();

  private readonly api = inject(ToeicApiService);
  protected readonly parts = PICTURE_PARTS;
  protected readonly missing = signal<MissingPictures['byLevel'] | null>(null);
  protected readonly batch = signal<PhotoBatch | null>(null);
  protected readonly codex = signal(false);
  /** The part whose levels are being chosen (first step), then whose engine is asked (second). */
  protected readonly picking = signal<1 | 6 | 7 | null>(null);
  protected readonly asking = signal<1 | 6 | 7 | null>(null);
  /** The levels ticked for the batch being asked (full ones too, kept on "Quay lại"). */
  protected readonly chosen = signal<number[]>([]);
  /** The ticked levels that still miss pictures in the part being asked: what the batch draws. */
  private readonly drawLevels = computed(() => {
    const byLevel = this.levelsOf(this.asking());
    return this.chosen().filter((l) => (byLevel[String(l)] ?? 0) > 0);
  });
  protected readonly error = signal('');
  private timer: ReturnType<typeof setTimeout> | null = null;

  protected readonly progress = computed(() => {
    const b = this.batch();
    if (!b || (!b.running && b.total === 0)) {
      return '';
    }
    const failed = Object.keys(b.failed).length;
    const where = b.keys?.length
      ? 'các cụm vừa soạn'
      : b.testKey
        ? `đề ${b.testKey}`
        : `Part ${b.part}` + (b.levels?.length ? ` (mức ${b.levels.join(', ')})` : '');
    let text = b.running
      ? `Đang vẽ ${where}: ảnh ${b.saved.length + failed + 1}/${b.total} (${b.engine})…`
      : `Đã vẽ ${b.saved.length}/${b.total} ảnh của ${where}.`;
    if (failed) {
      text += ` ${failed} ảnh lỗi.`;
    }
    if (b.stopReason) {
      text += ` Dừng vì: ${b.stopReason}`;
    }
    return text;
  });

  protected readonly dialogMessage = computed(() => {
    const p = this.asking();
    if (!p) {
      return '';
    }
    const byLevel = this.levelsOf(p);
    const n = this.drawLevels().reduce((sum, l) => sum + (byLevel[String(l)] ?? 0), 0);
    return `Vẽ lần lượt ${n} ảnh còn thiếu của Part ${p}, mức ${this.drawLevels().join(', ')} (kể cả cụm nháp), chạy ngầm trên máy chủ; xem tiến độ ở đây.`;
  });

  constructor() {
    inject(DestroyRef).onDestroy(() => this.clearTimer());
  }

  ngOnInit(): void {
    this.loadMissing();
    this.poll();
  }

  /** Reads the counts and the batch again (after "Nhờ AI soạn" may have started one). */
  refresh(): void {
    this.loadMissing();
    this.poll();
  }

  /** The pictures missing in a part, every level. */
  protected missingOf(part: number): number {
    return Object.values(this.levelsOf(part)).reduce((sum, n) => sum + n, 0);
  }

  /** The pictures missing in a part by level ("450" → n). */
  protected levelsOf(part: number | null): Record<string, number> {
    return part ? (this.missing()?.[String(part)] ?? {}) : {};
  }

  /** "Vẽ ảnh còn thiếu…": the levels are asked first, none ticked. */
  protected openLevels(part: 1 | 6 | 7): void {
    this.chosen.set([]);
    this.picking.set(part);
  }

  /** "Quay lại" from the engine step: the levels again, as they were ticked. */
  protected backToLevels(): void {
    const part = this.asking();
    this.asking.set(null);
    this.picking.set(part);
  }

  /** The levels are chosen: the engine is asked next. */
  protected chooseLevels(levels: number[]): void {
    const part = this.picking();
    this.picking.set(null);
    this.chosen.set(levels);
    this.asking.set(part);
  }

  protected start(engine: ImageEngine): void {
    const part = this.asking();
    const levels = this.drawLevels();
    this.asking.set(null);
    if (!part || levels.length === 0) {
      return;
    }
    this.error.set('');
    this.api.startPhotoBatch({ part, levels }, engine).subscribe({
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

  private loadMissing(): void {
    this.api.missingPictures().subscribe({
      next: (r) => {
        this.missing.set(r.byLevel);
        this.codex.set(r.codex);
      },
      error: () => this.error.set('Không đếm được ảnh còn thiếu.'),
    });
  }

  private poll(): void {
    this.api.photoBatch().subscribe({
      next: (r) => {
        const was = this.batch();
        this.batch.set(r.batch);
        this.codex.set(r.codex);
        if (r.batch.running) {
          if (was && r.batch.saved.length !== was.saved.length) {
            this.loadMissing();
          }
          this.schedule();
        } else if (was?.running) {
          this.loadMissing();
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
