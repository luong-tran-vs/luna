import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  OnInit,
  signal,
} from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { SpotCount, SpotImportReply, SpotItem } from '../../../core/models/games';
import { GamesApiService } from '../../../core/services/games-api.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { grammarFailure } from '../grammar-errors';
import { SpotImportDialog } from './spot-import-dialog';

const GENERATE_FAILED = 'AI chưa soạn được câu, vui lòng thử lại.';
const ACTION_FAILED = 'Không thực hiện được, vui lòng thử lại.';
/** A point with fewer sentences is flagged (FR-026a). */
export const SPOT_ENOUGH = 10;

/** The fields of the edit form the server can complain about. */
const EDIT_FIELDS = ['sentence', 'wrong', 'right', 'distractors', 'ruleVi'] as const;
type EditField = (typeof EDIT_FIELDS)[number];

/**
 * F26 (FR-026a, FR-026b): the "Bắn lỗi" sentences of one grammar point on its admin page: how many
 * the game gets by source, "Nhờ AI soạn thêm" (one AI request, up to 10 drafts), and the extra
 * sentences with Duyệt / Sửa / Xoá. Drafts reach the game only once approved.
 */
@Component({
  selector: 'lu-grammar-spots',
  imports: [ConfirmDialog, ReactiveFormsModule, SpotImportDialog],
  templateUrl: './grammar-spots.html',
  styleUrl: './grammar-spots.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarSpots implements OnInit {
  readonly pointId = input.required<string>();
  /** Whether the point's lesson is published: only then can the AI write sentences. */
  readonly published = input(false);

  private readonly api = inject(GamesApiService);

  protected readonly enough = SPOT_ENOUGH;
  protected readonly counts = signal<SpotCount | null>(null);
  protected readonly items = signal<SpotItem[] | null>(null);
  protected readonly loadError = signal(false);
  /** 'generate', or the id of the sentence whose request runs. */
  protected readonly busy = signal<string | null>(null);
  protected readonly note = signal('');
  protected readonly error = signal<string | null>(null);
  protected readonly deleting = signal<SpotItem | null>(null);
  /** The import dialog is open. */
  protected readonly importing = signal(false);

  protected readonly editing = signal<string | null>(null);
  protected readonly editError = signal<string | null>(null);
  protected readonly editFields = signal<Partial<Record<EditField, string>>>({});
  protected readonly form = new FormGroup({
    sentence: new FormControl('', { nonNullable: true }),
    wrong: new FormControl('', { nonNullable: true }),
    right: new FormControl('', { nonNullable: true }),
    d0: new FormControl('', { nonNullable: true }),
    d1: new FormControl('', { nonNullable: true }),
    ruleVi: new FormControl('', { nonNullable: true }),
  });

  ngOnInit(): void {
    this.loadCounts();
    this.loadItems();
  }

  private loadCounts(): void {
    this.api.spotCounts().subscribe({
      next: (r) => this.counts.set(r.points.find((p) => p.pointId === this.pointId()) ?? null),
      error: () => this.counts.set(null),
    });
  }

  protected loadItems(): void {
    this.loadError.set(false);
    this.api.spotItems(this.pointId()).subscribe({
      next: (r) => this.items.set(r.items),
      error: () => this.loadError.set(true),
    });
  }

  protected async generate(): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.busy.set('generate');
    this.error.set(null);
    this.note.set('');
    try {
      const r = await firstValueFrom(this.api.generateSpots(this.pointId()));
      this.items.update((list) => [...r.created, ...(list ?? [])]);
      this.note.set(`Đã soạn ${r.created.length} câu, bỏ ${r.dropped} câu không hợp lệ.`);
      this.loadCounts();
    } catch (err) {
      this.error.set(grammarFailure(err, GENERATE_FAILED, true).message);
    } finally {
      this.busy.set(null);
    }
  }

  /** The drafts of the list, approved at once by "Duyệt hết" (added 2026-11-16). */
  protected readonly draftCount = computed(
    () => (this.items() ?? []).filter((i) => i.status === 'draft').length,
  );

  protected async approveAll(): Promise<void> {
    if (this.busy() || this.draftCount() === 0) {
      return;
    }
    this.busy.set('approve-all');
    this.error.set(null);
    this.note.set('');
    try {
      const r = await firstValueFrom(this.api.approveSpotDrafts([this.pointId()]));
      this.items.update((list) =>
        (list ?? []).map((i) => (i.status === 'draft' ? { ...i, status: 'approved' } : i)),
      );
      this.note.set(`Đã duyệt ${r.approved} câu.`);
      this.loadCounts();
    } catch (err) {
      this.error.set(grammarFailure(err, ACTION_FAILED).message);
    } finally {
      this.busy.set(null);
    }
  }

  protected imported(r: SpotImportReply): void {
    this.importing.set(false);
    this.items.update((list) => [...r.created, ...(list ?? [])]);
    this.error.set(null);
    this.note.set(`Đã nhập ${r.created.length} câu thành nháp, bỏ ${r.dropped} câu.`);
    this.loadCounts();
  }

  protected async approve(item: SpotItem): Promise<void> {
    await this.run(item, async () => {
      const r = await firstValueFrom(this.api.approveSpot(item.id));
      this.replace(r.item);
      this.note.set('Đã duyệt câu.');
    });
  }

  protected startEdit(item: SpotItem): void {
    if (this.busy()) {
      return;
    }
    this.form.setValue({
      sentence: item.sentence,
      wrong: item.wrong,
      right: item.right,
      d0: item.distractors[0] ?? '',
      d1: item.distractors[1] ?? '',
      ruleVi: item.ruleVi,
    });
    this.editError.set(null);
    this.editFields.set({});
    this.editing.set(item.id);
  }

  protected cancelEdit(): void {
    this.editing.set(null);
  }

  protected async save(item: SpotItem): Promise<void> {
    const v = this.form.getRawValue();
    this.editError.set(null);
    this.editFields.set({});
    await this.run(
      item,
      async () => {
        const r = await firstValueFrom(
          this.api.updateSpot(item.id, {
            sentence: v.sentence.trim(),
            wrong: v.wrong.trim(),
            right: v.right.trim(),
            distractors: [v.d0.trim(), v.d1.trim()],
            ruleVi: v.ruleVi.trim(),
          }),
        );
        this.replace(r.item);
        this.editing.set(null);
        this.note.set('Đã lưu câu.');
      },
      (err) => {
        const fields = fieldsOf(err);
        this.editFields.set(fields);
        this.editError.set(
          Object.keys(fields).length
            ? 'Câu chưa hợp lệ, xem lỗi ở từng ô.'
            : grammarFailure(err, ACTION_FAILED).message,
        );
      },
    );
  }

  protected askDelete(item: SpotItem): void {
    if (!this.busy()) {
      this.deleting.set(item);
    }
  }

  protected async confirmDelete(): Promise<void> {
    const item = this.deleting();
    this.deleting.set(null);
    if (!item) {
      return;
    }
    await this.run(item, async () => {
      await firstValueFrom(this.api.deleteSpot(item.id));
      this.items.update((list) => (list ?? []).filter((i) => i.id !== item.id));
      if (this.editing() === item.id) {
        this.editing.set(null);
      }
      this.note.set('Đã xoá câu.');
    });
  }

  protected deleteMessage(): string {
    const item = this.deleting();
    return item ? `Xoá câu “${item.sentence}”? Không lấy lại được.` : '';
  }

  private async run(
    item: SpotItem,
    call: () => Promise<void>,
    fail: (err: unknown) => void = (err) =>
      this.error.set(grammarFailure(err, ACTION_FAILED).message),
  ): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.busy.set(item.id);
    this.error.set(null);
    this.note.set('');
    try {
      await call();
      this.loadCounts();
    } catch (err) {
      fail(err);
    } finally {
      this.busy.set(null);
    }
  }

  private replace(item: SpotItem): void {
    this.items.update((list) => (list ?? []).map((i) => (i.id === item.id ? item : i)));
  }
}

/** The field errors of a 400 (`fields`), by field of the form. */
function fieldsOf(err: unknown): Partial<Record<EditField, string>> {
  if (!(err instanceof ApiError) || err.status !== 400) {
    return {};
  }
  const raw = (err.body as { fields?: unknown } | null)?.fields;
  if (!raw || typeof raw !== 'object') {
    return {};
  }
  const out: Partial<Record<EditField, string>> = {};
  for (const [key, text] of Object.entries(raw as Record<string, unknown>)) {
    const field = EDIT_FIELDS.find(
      (f) => key === f || key.startsWith(f + '.') || key.startsWith(f + '['),
    );
    if (field) {
      out[field] = String(text);
    }
  }
  return out;
}
