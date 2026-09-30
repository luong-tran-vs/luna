import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { NgTemplateOutlet } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Level, LEVELS } from '../../../core/models/lesson';
import { groupByLevel, Topic, topicLabel } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';

type FieldName = 'name' | 'level' | 'description';

const REQUIRED: Partial<Record<FieldName, string>> = {
  name: 'Vui lòng nhập tên chủ đề',
  level: 'Vui lòng chọn trình độ',
};

/** Topic catalogue (F14): topics grouped by level, add and edit in place, delete with confirmation. */
@Component({
  selector: 'lu-topics',
  imports: [ConfirmDialog, NgTemplateOutlet, ReactiveFormsModule, RouterLink],
  templateUrl: './topics.html',
  styleUrl: './topics.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Topics {
  private readonly api = inject(AdminApiService);

  protected readonly levels = LEVELS;
  protected readonly label = topicLabel;
  protected readonly topics = signal<Topic[] | null>(null);
  protected readonly groups = computed(() => groupByLevel(this.topics() ?? []));
  protected readonly error = signal<string | null>(null);
  /** "new", the id of the topic being edited, or null. */
  protected readonly editing = signal<string | null>(null);
  protected readonly deleting = signal<Topic | null>(null);
  protected readonly pending = signal(false);
  protected readonly submitted = signal(false);
  protected readonly serverFields = signal<Partial<Record<FieldName, string>>>({});

  protected readonly form = inject(NonNullableFormBuilder).group({
    name: ['', [Validators.required, Validators.maxLength(60)]],
    level: ['' as Level | '', Validators.required],
    description: ['', Validators.maxLength(200)],
  });

  constructor() {
    this.load();
  }

  private load(): void {
    this.api.topics().subscribe({
      next: (list) => this.topics.set(list),
      error: () => this.error.set('Không tải được danh sách chủ đề.'),
    });
  }

  protected startAdd(): void {
    this.open('new', { name: '', level: '', description: '' });
  }

  protected startEdit(t: Topic): void {
    this.open(t.id, { name: t.name, level: t.level, description: t.description });
  }

  private open(target: string, value: { name: string; level: Level | ''; description: string }): void {
    this.form.reset(value);
    this.submitted.set(false);
    this.serverFields.set({});
    this.error.set(null);
    this.editing.set(target);
  }

  protected cancel(): void {
    this.editing.set(null);
  }

  protected fieldError(name: FieldName): string | null {
    const c = this.form.controls[name];
    if (c.touched || this.submitted()) {
      if (c.hasError('required')) {
        return REQUIRED[name] ?? null;
      }
      if (c.hasError('maxlength')) {
        return `Tối đa ${(c.getError('maxlength') as { requiredLength: number }).requiredLength} ký tự`;
      }
    }
    return this.serverFields()[name] ?? null;
  }

  protected describedBy(name: FieldName): string | null {
    return this.fieldError(name) ? `topic-${name}-error` : null;
  }

  protected async save(): Promise<void> {
    this.submitted.set(true);
    this.serverFields.set({});
    const target = this.editing();
    if (this.form.invalid || this.pending() || !target) {
      this.form.markAllAsTouched();
      return;
    }
    const v = this.form.getRawValue();
    const input = { name: v.name.trim(), level: v.level, description: v.description.trim() };
    this.pending.set(true);
    try {
      await firstValueFrom(target === 'new' ? this.api.createTopic(input) : this.api.updateTopic(target, input));
      this.editing.set(null);
      this.load();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      if (body?.fields) {
        this.serverFields.set(body.fields);
      } else {
        this.error.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
      }
    } finally {
      this.pending.set(false);
    }
  }

  protected async confirmDelete(): Promise<void> {
    const t = this.deleting();
    this.deleting.set(null);
    if (!t) {
      return;
    }
    this.error.set(null);
    try {
      await firstValueFrom(this.api.deleteTopic(t.id));
      this.load();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Không xoá được chủ đề, vui lòng thử lại.');
    }
  }
}
