import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { SpotImportItem, SpotImportReply } from '../../../core/models/games';
import { GamesApiService } from '../../../core/services/games-api.service';
import { errorBody, fieldLines, trapTab } from '../json-import/json-import';
import { JsonSource } from '../json-import/json-source';

/** Vietnamese names of the fields a sentence can fail on. */
const FIELD_NAMES: Record<string, string> = {
  sentence: 'Câu',
  wrong: 'Chỗ sai',
  right: 'Chữ đúng',
  distractors: 'Chữ nhiễu',
  ruleVi: 'Quy tắc',
};

/** The sentences of the pasted JSON: an array, or an object with them under "items". */
export function parseSpotItems(text: string): { items: SpotImportItem[] } | { error: string } {
  if (!text.trim()) {
    return { error: 'Chưa có nội dung: dán JSON hoặc chọn file .json.' };
  }
  let value: unknown;
  try {
    value = JSON.parse(text);
  } catch (err) {
    return { error: `Không đọc được JSON: ${err instanceof Error ? err.message : String(err)}` };
  }
  const list = Array.isArray(value)
    ? value
    : value && typeof value === 'object' && Array.isArray((value as { items?: unknown }).items)
      ? (value as { items: unknown[] }).items
      : null;
  if (!list) {
    return { error: 'JSON phải là một mảng câu [ … ] hoặc { "items": [ … ] }.' };
  }
  if (list.some((it) => !it || typeof it !== 'object' || Array.isArray(it))) {
    return { error: 'Mỗi câu phải là một đối tượng { … }.' };
  }
  const str = (v: unknown): string => (typeof v === 'string' ? v : '');
  return {
    items: list.map((raw) => {
      const it = raw as Record<string, unknown>;
      return {
        sentence: str(it['sentence']),
        wrong: str(it['wrong']),
        right: str(it['right']),
        distractors: Array.isArray(it['distractors']) ? it['distractors'].map(str) : [],
        ruleVi: str(it['ruleVi']),
      };
    }),
  };
}

/**
 * Imports "Bắn lỗi" sentences written outside the AI (an admin, or Claude) into a grammar point:
 * the JSON is in the shape the AI returns, checked on the server without saving, then the valid
 * sentences that are not already there are saved as drafts.
 */
@Component({
  selector: 'lu-spot-import-dialog',
  imports: [JsonSource],
  templateUrl: './spot-import-dialog.html',
  styleUrls: [
    '../../../shared/components/confirm-dialog/confirm-dialog.css',
    '../grammar-list/import-dialog/import-dialog.css',
    './spot-import-dialog.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class SpotImportDialog {
  private readonly api = inject(GamesApiService);

  readonly open = input(false);
  readonly pointId = input('');

  readonly saved = output<SpotImportReply>();
  readonly closed = output<void>();

  protected readonly text = signal('');
  protected readonly checking = signal(false);
  protected readonly saving = signal(false);
  protected readonly parseError = signal('');
  protected readonly failure = signal('');
  protected readonly checked = signal<{ text: string; reply: SpotImportReply } | null>(null);

  protected readonly reports = computed(() =>
    (this.checked()?.reply.reports ?? []).map((r) => ({
      ...r,
      reason: r.duplicate
        ? 'Trùng câu đã có'
        : fieldLines(r.fields ?? {})
            .map((f) => `${FIELD_NAMES[f.key] ?? f.key}: ${f.text}`)
            .join('; '),
    })),
  );
  protected readonly canSave = computed(() => {
    const c = this.checked();
    return !!c && c.text === this.text() && c.reply.kept > 0 && !this.saving() && !this.checking();
  });
  protected readonly status = computed(() => {
    const c = this.checked();
    if (this.checking()) {
      return 'Đang kiểm tra…';
    }
    if (this.saving()) {
      return 'Đang lưu…';
    }
    if (c && c.text !== this.text()) {
      return 'Nội dung đã đổi sau lần kiểm tra, hãy kiểm tra lại.';
    }
    if (c) {
      return `${c.reply.kept} câu hợp lệ, ${c.reply.dropped} câu bị bỏ. Đã kiểm tra, chưa lưu.`;
    }
    return '';
  });

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        this.pointId();
        untracked(() => this.reset());
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        untracked(() => this.dialog()?.nativeElement.querySelector<HTMLElement>('h2')?.focus());
      }
    });
  }

  private reset(): void {
    this.text.set('');
    this.parseError.set('');
    this.failure.set('');
    this.checked.set(null);
  }

  protected onEscape(): void {
    if (this.open()) {
      this.closed.emit();
    }
  }

  protected onKeydown(event: KeyboardEvent): void {
    trapTab(event, this.dialog()?.nativeElement);
  }

  protected async check(): Promise<void> {
    const items = this.parse();
    if (!items) {
      return;
    }
    const text = this.text();
    this.checking.set(true);
    try {
      const reply = await firstValueFrom(this.api.importSpots(this.pointId(), items, true));
      this.checked.set({ text, reply });
    } catch (err) {
      this.showError(err);
    } finally {
      this.checking.set(false);
    }
  }

  protected async save(): Promise<void> {
    const items = this.parse();
    if (!items || !this.canSave()) {
      return;
    }
    this.saving.set(true);
    try {
      this.saved.emit(await firstValueFrom(this.api.importSpots(this.pointId(), items, false)));
    } catch (err) {
      this.showError(err);
    } finally {
      this.saving.set(false);
    }
  }

  private parse(): SpotImportItem[] | null {
    this.failure.set('');
    const parsed = parseSpotItems(this.text());
    if ('error' in parsed) {
      this.parseError.set(parsed.error);
      this.checked.set(null);
      return null;
    }
    this.parseError.set('');
    return parsed.items;
  }

  private showError(err: unknown): void {
    const e = errorBody(err);
    this.checked.set(null);
    if (e.fields) {
      this.failure.set(Object.values(e.fields).join('; '));
      return;
    }
    this.failure.set(e.message ?? 'Không gửi được câu, vui lòng thử lại.');
  }

  protected readonly sample = `[
  {
    "sentence": "She go to school every day.",
    "wrong": "go",
    "right": "goes",
    "distractors": ["going", "gone"],
    "ruleVi": "Chủ ngữ số ít (he/she/it) + động từ thêm -s/-es."
  }
]`;
}
