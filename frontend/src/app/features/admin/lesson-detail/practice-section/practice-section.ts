import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { JobStatus, Lesson, LessonFlag, TranslationInput } from '../../../../core/models/lesson';
import { AdminApiService } from '../../admin-api.service';
import { ConfirmDialog } from '../../../../shared/components/confirm-dialog/confirm-dialog';
import { FlagNote } from '../../flag-note/flag-note';
import { StatusChip } from '../../status-chip/status-chip';

/**
 * The lesson's practice (F17) as admins see it: status, failure reason, the generated content
 * (read only) and a button to generate it again with one AI request.
 */
@Component({
  selector: 'lu-practice-section',
  imports: [StatusChip, FlagNote, ConfirmDialog, ReactiveFormsModule, NgTemplateOutlet],
  templateUrl: './practice-section.html',
  styleUrl: './practice-section.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeSection {
  private readonly api = inject(AdminApiService);

  readonly lesson = input.required<Lesson>();
  /** The lesson returned by the API once regeneration is queued. */
  readonly regenerated = output<Lesson>();
  /** F22: flags on the translations (index = translation position); empty when not checked. */
  readonly flags = input<LessonFlag[]>([]);
  readonly flagBusy = input(false);
  readonly confirm = output<LessonFlag>();
  /** The lesson returned after a translation was edited or removed. */
  readonly updated = output<Lesson>();

  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  /** Status for the chip; null when there is no practice yet. */
  protected readonly chipStatus = computed<JobStatus | null>(() => {
    const s = this.lesson().practiceStatus;
    return s === 'none' ? null : s;
  });

  /** Why regenerating is not possible now, or null. */
  protected readonly blockedReason = computed(() => {
    const l = this.lesson();
    if (l.practiceStatus === 'running') {
      return 'Đang sinh…';
    }
    return l.annotationStatus === 'done' ? null : 'Cần chú thích xong trước';
  });

  // --- editing translations ---

  protected readonly editIndex = signal<number | null>(null);
  protected readonly deleteIndex = signal<number | null>(null);
  protected readonly saving = signal(false);
  protected readonly editError = signal<string | null>(null);
  protected readonly fields = signal<Record<string, string>>({});
  protected readonly vi = new FormControl('', { nonNullable: true });
  protected readonly en = new FormControl('', { nonNullable: true });
  protected readonly distractors = new FormControl('', { nonNullable: true });

  /** Why translations cannot be edited now, or null. */
  protected readonly editBlocked = computed(() =>
    this.lesson().practiceStatus === 'running' ? 'Đang sinh phần luyện tập, chưa sửa được câu dịch.' : null,
  );

  protected fieldError(name: 'vi' | 'en' | 'distractors'): string | null {
    const i = this.editIndex();
    return i === null ? null : (this.fields()[`translations.${i}.${name}`] ?? null);
  }

  protected startEdit(i: number): void {
    const t = this.lesson().practice?.translations[i];
    if (!t || this.editBlocked()) {
      return;
    }
    this.vi.setValue(t.vi);
    this.en.setValue(t.en);
    this.distractors.setValue(t.distractors.join(', '));
    this.fields.set({});
    this.editError.set(null);
    this.editIndex.set(i);
    setTimeout(() => document.getElementById('tr-vi-' + i)?.focus());
  }

  protected cancelEdit(): void {
    this.editIndex.set(null);
    this.fields.set({});
    this.editError.set(null);
  }

  private current(): TranslationInput[] {
    return (this.lesson().practice?.translations ?? []).map((t) => ({ vi: t.vi, en: t.en, distractors: [...t.distractors] }));
  }

  protected async saveEdit(): Promise<void> {
    const i = this.editIndex();
    if (i === null) {
      return;
    }
    const list = this.current();
    list[i] = {
      vi: this.vi.value,
      en: this.en.value,
      distractors: this.distractors.value
        .split(',')
        .map((d) => d.trim())
        .filter((d) => d !== ''),
    };
    if (await this.send(list)) {
      this.cancelEdit();
    }
  }

  protected async confirmDelete(): Promise<void> {
    const i = this.deleteIndex();
    this.deleteIndex.set(null);
    if (i === null) {
      return;
    }
    const list = this.current();
    list.splice(i, 1);
    if (await this.send(list)) {
      this.cancelEdit();
    }
  }

  private async send(list: TranslationInput[]): Promise<boolean> {
    if (this.saving() || this.editBlocked()) {
      return false;
    }
    this.saving.set(true);
    this.editError.set(null);
    this.fields.set({});
    try {
      this.updated.emit(await firstValueFrom(this.api.updateTranslations(this.lesson().id, list)));
      return true;
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      if (body?.fields) {
        this.fields.set(body.fields);
        this.editError.set('Vui lòng sửa các ô được đánh dấu.');
      } else {
        this.editError.set(body?.message ?? 'Không lưu được, vui lòng thử lại.');
      }
      return false;
    } finally {
      this.saving.set(false);
    }
  }

  protected flagFor(i: number): LessonFlag | null {
    return this.flags().find((f) => f.index === i) ?? null;
  }

  protected async regenerate(): Promise<void> {
    if (this.busy() || this.blockedReason()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    try {
      this.regenerated.emit(await firstValueFrom(this.api.regeneratePractice(this.lesson().id)));
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Không tạo lại được, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }
}
