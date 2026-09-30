import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { AbstractControl, NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Lesson, LessonInput } from '../../../core/models/lesson';
import { groupByLevel, Topic, topicLabel } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';

type FieldName = 'title' | 'topicId' | 'source' | 'license' | 'content';

const REQUIRED_MESSAGE: Record<FieldName, string> = {
  title: 'Vui lòng nhập tiêu đề',
  topicId: 'Vui lòng chọn chủ đề',
  source: 'Vui lòng nhập nguồn',
  license: 'Vui lòng nhập giấy phép',
  content: 'Vui lòng dán nội dung bài',
};

export const MAX_CONTENT = 10000;

@Component({
  selector: 'lu-lesson-form',
  imports: [ReactiveFormsModule, RouterLink, ConfirmDialog],
  templateUrl: './lesson-form.html',
  styleUrl: './lesson-form.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonForm {
  private readonly api = inject(AdminApiService);
  private readonly router = inject(Router);
  protected readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id');

  protected readonly topics = signal<Topic[] | null>(null);
  protected readonly topicGroups = computed(() => groupByLevel(this.topics() ?? []));
  protected readonly topicLabel = topicLabel;
  protected readonly maxContent = MAX_CONTENT;
  protected readonly form = inject(NonNullableFormBuilder).group({
    title: ['', [Validators.required, Validators.maxLength(200)]],
    topicId: ['', Validators.required],
    source: ['', [Validators.required, Validators.maxLength(200)]],
    license: ['', [Validators.required, Validators.maxLength(100)]],
    content: ['', [Validators.required, Validators.maxLength(MAX_CONTENT)]],
  });

  private readonly original = signal<Lesson | null>(null);
  protected readonly loadError = signal<string | null>(null);
  protected readonly contentLength = signal(0);
  protected readonly pending = signal(false);
  protected readonly submitted = signal(false);
  protected readonly serverError = signal<string | null>(null);
  protected readonly serverFields = signal<Partial<Record<FieldName, string>>>({});
  protected readonly confirmOpen = signal(false);

  protected readonly isEdit = computed(() => this.id !== null);
  private readonly hasManualEdits = computed(
    () => this.original()?.annotations.some((a) => a.editedByAdmin) ?? false,
  );

  constructor() {
    this.form.controls.content.valueChanges.subscribe((v) => this.contentLength.set(v.length));
    this.api.topics().subscribe({
      next: (list) => this.topics.set(list),
      error: () => this.loadError.set('Không tải được danh sách chủ đề.'),
    });
    if (this.id) {
      void this.load(this.id);
    }
  }

  private async load(id: string): Promise<void> {
    try {
      const l = await firstValueFrom(this.api.get(id));
      this.original.set(l);
      this.form.setValue({
        title: l.title, topicId: l.topicId, source: l.source, license: l.license, content: l.content,
      });
    } catch {
      this.loadError.set('Không tải được bài học.');
    }
  }

  protected error(name: FieldName): string | null {
    const c: AbstractControl = this.form.controls[name];
    if (c.touched || this.submitted()) {
      if (c.hasError('required')) {
        return REQUIRED_MESSAGE[name];
      }
      if (c.hasError('maxlength')) {
        const max = (c.getError('maxlength') as { requiredLength: number }).requiredLength;
        return name === 'content' ? 'Nội dung tối đa 10.000 ký tự' : `Tối đa ${max} ký tự`;
      }
    }
    return this.serverFields()[name] ?? null;
  }

  protected describedBy(name: FieldName, hint?: string): string | null {
    const ids = [hint, this.error(name) ? `lesson-${name}-error` : undefined].filter(Boolean);
    return ids.length ? ids.join(' ') : null;
  }

  protected submit(): void {
    this.submitted.set(true);
    this.serverError.set(null);
    this.serverFields.set({});
    if (this.form.invalid || this.pending()) {
      this.form.markAllAsTouched();
      return;
    }
    const contentChanged = this.form.controls.content.value.trim() !== this.original()?.content;
    if (this.isEdit() && contentChanged && this.hasManualEdits()) {
      this.confirmOpen.set(true);
      return;
    }
    void this.save();
  }

  protected confirmSave(): void {
    this.confirmOpen.set(false);
    void this.save();
  }

  private async save(): Promise<void> {
    const v = this.form.getRawValue();
    const input: LessonInput = {
      title: v.title.trim(),
      topicId: v.topicId,
      source: v.source.trim(),
      license: v.license.trim(),
      content: v.content.trim(),
    };

    this.pending.set(true);
    try {
      const saved = await firstValueFrom(this.id ? this.api.update(this.id, input) : this.api.create(input));
      await this.router.navigateByUrl(`/admin/lessons/${saved.id}`);
    } catch (err) {
      this.showError(err);
    } finally {
      this.pending.set(false);
    }
  }

  private showError(err: unknown): void {
    const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
    if (body?.fields) {
      this.serverFields.set(body.fields as Partial<Record<FieldName, string>>);
      this.serverError.set('Vui lòng sửa các ô được đánh dấu.');
    } else if (err instanceof ApiError && err.kind === 'network') {
      this.serverError.set('Không kết nối được máy chủ. Vui lòng thử lại.');
    } else {
      this.serverError.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
    }
  }
}
