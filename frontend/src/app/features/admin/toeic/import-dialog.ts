import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { ImportReport } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { errorBody, fieldLines, parseJson, trapTab } from '../json-import/json-import';
import { JsonSource } from '../json-import/json-source';

const STATUS_LABELS: Record<string, string> = {
  ok: 'Mới',
  duplicate: 'Trùng – bỏ qua',
  replace: 'Thay thế',
  error: 'Lỗi',
};

/**
 * "Nhập JSON" of the TOEIC groups (FR-040): a file of content/toeic/ is pasted or picked, checked on
 * the server (dry run: a report per group, the field errors, the listening minutes), then imported.
 * Import is only on after a clean dry run of the text as it is now. New groups come in as drafts;
 * a known key is skipped unless "Thay thế cụm trùng" is ticked (then attempts are graded again).
 */
@Component({
  selector: 'lu-toeic-import-dialog',
  imports: [JsonSource],
  template: `
    <div class="backdrop">
      <div
        #dialog
        class="dialog wide"
        role="dialog"
        aria-modal="true"
        aria-labelledby="toeic-import-title"
        (keydown)="onKeydown($event)"
      >
        <header class="head">
          <div>
            <h2 class="title" id="toeic-import-title">Nhập cụm câu TOEIC</h2>
            <p class="sub">
              File JSON <code>&#123; "groups": [ … ] &#125;</code> trong content/toeic/. Cụm mới lưu
              ở dạng nháp.
            </p>
          </div>
          <button type="button" class="close" aria-label="Đóng" (click)="closed.emit(false)">
            ×
          </button>
        </header>

        <lu-json-source idPrefix="toeic-import" [(text)]="text" [invalid]="!!parseError()" />
        <label class="check">
          <input type="checkbox" [checked]="replace()" (change)="toggleReplace()" />
          Thay thế cụm trùng mã (sửa nội dung, giữ trạng thái, chấm lại các bài đã nộp)
        </label>

        @if (parseError()) {
          <p class="form-alert" role="alert">{{ parseError() }}</p>
        }
        @if (failure()) {
          <p class="form-alert" role="alert">{{ failure() }}</p>
        }
        @if (bodyErrors().length > 0) {
          <div class="form-alert" role="alert">
            <ul>
              @for (f of bodyErrors(); track f.key) {
                <li>
                  <code>{{ f.key }}</code>: {{ f.text }}
                </li>
              }
            </ul>
          </div>
        }

        @if (report(); as r) {
          <section class="report" aria-label="Báo cáo">
            <p class="summary" role="status">
              {{ r.dryRun ? 'Chạy thử' : 'Đã nhập' }}: {{ r.created.length }} mới ·
              {{ r.replaced.length }} thay thế · {{ r.duplicates.length }} trùng ·
              {{ errorCount() }} lỗi
              @if (r.listeningMinutes !== null) {
                · phần Nghe ≈ {{ r.listeningMinutes }} phút
              }
              @if (r.rescored > 0) {
                · {{ r.dryRun ? 'sẽ chấm lại' : 'đã chấm lại' }} {{ r.rescored }} bài
              }
            </p>
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th scope="col">Mã cụm</th>
                    <th scope="col">Part</th>
                    <th scope="col">Câu</th>
                    <th scope="col">Kết quả</th>
                  </tr>
                </thead>
                <tbody>
                  @for (g of r.reports; track $index) {
                    <tr [attr.data-status]="g.status">
                      <td>
                        <code>{{ g.key || '(trống)' }}</code>
                      </td>
                      <td>{{ g.part }}</td>
                      <td>{{ g.questions }}</td>
                      <td>
                        <span class="mk" aria-hidden="true">{{
                          g.status === 'error' ? '✕' : '✓'
                        }}</span>
                        {{ label(g.status) }}
                        @if (g.fields) {
                          <ul class="fields">
                            @for (f of lines(g.fields); track f.key) {
                              <li>
                                <code>{{ f.key }}</code>: {{ f.text }}
                              </li>
                            }
                          </ul>
                        }
                      </td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          </section>
        }

        <div class="actions">
          <button type="button" class="btn" (click)="closed.emit(imported())">
            {{ imported() ? 'Đóng' : 'Huỷ' }}
          </button>
          <button type="button" class="btn" [disabled]="busy()" (click)="dryRun()">Chạy thử</button>
          <button type="button" class="btn-primary" [disabled]="!canImport()" (click)="run()">
            Nhập
          </button>
        </div>
      </div>
    </div>
  `,
  styleUrls: [
    '../../../shared/components/confirm-dialog/confirm-dialog.css',
    './import-dialog.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'closed.emit(imported())' },
})
export class ToeicImportDialog {
  /** true when something was imported (the list should reload). */
  readonly closed = output<boolean>();

  private readonly api = inject(ToeicApiService);
  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  protected readonly text = signal('');
  protected readonly replace = signal(false);
  protected readonly parseError = signal('');
  protected readonly failure = signal('');
  protected readonly bodyErrors = signal<{ key: string; text: string }[]>([]);
  protected readonly report = signal<ImportReport | null>(null);
  protected readonly busy = signal(false);
  protected readonly imported = signal(false);
  /** The text and option of the last clean dry run. */
  private readonly checked = signal<{ text: string; replace: boolean } | null>(null);

  protected readonly errorCount = computed(
    () => this.report()?.reports.filter((r) => r.status === 'error').length ?? 0,
  );
  protected readonly canImport = computed(() => {
    const c = this.checked();
    return !this.busy() && c !== null && c.text === this.text() && c.replace === this.replace();
  });

  protected label(s: string): string {
    return STATUS_LABELS[s] ?? s;
  }

  protected lines(f: Record<string, string>): { key: string; text: string }[] {
    return fieldLines(f);
  }

  protected toggleReplace(): void {
    this.replace.update((v) => !v);
  }

  protected dryRun(): void {
    this.send(true);
  }

  protected run(): void {
    this.send(false);
  }

  private send(dry: boolean): void {
    this.parseError.set('');
    this.failure.set('');
    this.bodyErrors.set([]);
    const parsed = parseJson(this.text());
    if ('error' in parsed) {
      this.parseError.set(parsed.error);
      return;
    }
    const sentText = this.text();
    const replace = this.replace();
    this.busy.set(true);
    this.api.importGroups(parsed.value, replace, dry).subscribe({
      next: (r) => {
        this.busy.set(false);
        this.report.set(r);
        const clean = r.reports.every((x) => x.status !== 'error');
        this.checked.set(dry && clean ? { text: sentText, replace } : null);
        if (!dry) {
          this.imported.set(true);
        }
      },
      error: (err: unknown) => {
        this.busy.set(false);
        this.checked.set(null);
        const e = errorBody(err);
        this.failure.set(e.message ?? 'Không gửi được. Kiểm tra mạng rồi thử lại.');
        if (e.fields) {
          this.bodyErrors.set(fieldLines(e.fields));
        }
      },
    });
  }

  protected onKeydown(event: KeyboardEvent): void {
    trapTab(event, this.dialog()?.nativeElement);
  }
}
