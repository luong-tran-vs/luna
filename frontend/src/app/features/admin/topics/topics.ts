import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { NgTemplateOutlet } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Level, LEVELS } from '../../../core/models/lesson';
import { sortTopics, Topic } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';
import { Loading } from '../../../shared/components/loading/loading';

type FieldName = 'name' | 'description';

const REQUIRED: Partial<Record<FieldName, string>> = {
  name: 'Vui lòng nhập tên chủ đề',
};

/** What the list shows: the topics with lessons at a level, or every topic. */
type TopicFilter = Level | 'all';

interface FilterOption {
  id: TopicFilter;
  label: string;
  count: number;
}

/**
 * Topic catalogue (F14). Topics are shared by every level; each card counts the lessons of each
 * level. The list shows the topics with lessons at one level at a time (the first such level by
 * default), or every topic; add and edit in place, delete with confirmation.
 */
@Component({
  selector: 'lu-topics',
  imports: [Loading, ConfirmDialog, NgTemplateOutlet, ReactiveFormsModule, RouterLink],
  templateUrl: './topics.html',
  styleUrl: './topics.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Topics {
  private readonly api = inject(AdminApiService);

  protected readonly topics = signal<Topic[] | null>(null);
  /** The filter picked above the list; null (or a level left without topics) shows the first one. */
  protected readonly filter = signal<TopicFilter | null>(null);
  /** One button per level that has lessons, then "Tất cả". */
  protected readonly filters = computed<FilterOption[]>(() => {
    const list = this.topics() ?? [];
    const levels = LEVELS.map((level) => ({
      id: level as TopicFilter,
      label: level,
      count: list.filter((t) => t.levels.some((l) => l.level === level && l.lessonCount > 0)).length,
    })).filter((f) => f.count > 0);
    return [...levels, { id: 'all', label: 'Tất cả', count: list.length }];
  });
  protected readonly shownFilter = computed<TopicFilter>(() => {
    const options = this.filters();
    return options.find((f) => f.id === this.filter())?.id ?? options[0].id;
  });
  /** The topics of the picked filter, by name; a long catalogue stays short. */
  protected readonly shown = computed(() => {
    const list = sortTopics(this.topics() ?? []);
    const f = this.shownFilter();
    return f === 'all' ? list : list.filter((t) => t.levels.some((l) => l.level === f && l.lessonCount > 0));
  });
  protected readonly heading = computed(() => {
    const f = this.shownFilter();
    const n = this.shown().length;
    return f === 'all' ? `Tất cả · ${n} chủ đề` : `Có bài ${f} · ${n} chủ đề`;
  });
  protected readonly error = signal<string | null>(null);
  /** "new", the id of the topic being edited, or null. */
  protected readonly editing = signal<string | null>(null);
  protected readonly deleting = signal<Topic | null>(null);
  protected readonly pending = signal(false);
  protected readonly submitted = signal(false);
  protected readonly serverFields = signal<Partial<Record<FieldName, string>>>({});

  protected readonly form = inject(NonNullableFormBuilder).group({
    name: ['', [Validators.required, Validators.maxLength(60)]],
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
    this.open('new', { name: '', description: '' });
  }

  protected startEdit(t: Topic): void {
    this.open(t.id, { name: t.name, description: t.description });
  }

  /** Query parameters of "Sinh bài AI": the roadmap page opens on the topic, at the level being listed. */
  protected generateParams(t: Topic): Record<string, string | number> {
    const f = this.shownFilter();
    const level = f === 'all' ? (t.levels.at(0)?.level ?? 'A1') : f;
    return { topicId: t.id, level, generate: 1 };
  }

  private open(target: string, value: { name: string; description: string }): void {
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
    const input = { name: v.name.trim(), description: v.description.trim() };
    this.pending.set(true);
    try {
      await firstValueFrom(target === 'new' ? this.api.createTopic(input) : this.api.updateTopic(target, input));
      this.editing.set(null);
      // A new topic has no lesson yet: only "Tất cả" lists it.
      if (target === 'new') {
        this.filter.set('all');
      }
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
