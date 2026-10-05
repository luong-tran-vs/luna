import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { AbstractControl, NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom, Subscription } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { GrammarPoint, grammarOptionLabel } from '../../../core/models/grammar';
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

type PointsState = 'idle' | 'loading' | 'error';

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
    grammarPointId: [''],
  });
  protected readonly points = signal<GrammarPoint[]>([]);
  protected readonly pointsState = signal<PointsState>('idle');
  protected readonly pointLabel = grammarOptionLabel;
  private pointsSub: Subscription | null = null;

  private readonly original = signal<Lesson | null>(null);
  protected readonly loadError = signal<string | null>(null);
  protected readonly contentLength = signal(0);
  protected readonly pending = signal(false);
  protected readonly submitted = signal(false);
  protected readonly serverError = signal<string | null>(null);
  protected readonly serverFields = signal<Partial<Record<FieldName | 'grammarPointId', string>>>({});
  protected readonly confirmOpen = signal(false);

  protected readonly isEdit = computed(() => this.id !== null);
  private readonly hasManualEdits = computed(
    () => this.original()?.annotations.some((a) => a.editedByAdmin) ?? false,
  );

  constructor() {
    this.form.controls.content.valueChanges.subscribe((v) => this.contentLength.set(v.length));
    this.form.controls.grammarPointId.disable();
    this.form.controls.topicId.valueChanges.subscribe((topicId) => this.loadPoints(topicId, ''));
    this.api.topics().subscribe({
      next: (list) => {
        this.topics.set(list);
        const topicId = this.form.controls.topicId.value;
        if (topicId && this.pointsState() === 'loading' && !this.pointsSub) {
          this.loadPoints(topicId, this.form.controls.grammarPointId.value);
        }
      },
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
      this.form.patchValue(
        { title: l.title, topicId: l.topicId, source: l.source, license: l.license, content: l.content },
        { emitEvent: false },
      );
      this.contentLength.set(l.content.length);
      this.loadPoints(l.topicId, l.grammarPointId ?? '');
    } catch {
      this.loadError.set('Không tải được bài học.');
    }
  }

  /** Level of the chosen topic; while the topic list loads, the edited lesson's own level. */
  private levelOf(topicId: string): string {
    const original = this.original();
    return this.topics()?.find((t) => t.id === topicId)?.level ?? (original?.topicId === topicId ? original.level : '');
  }

  /** Loads the grammar points of the topic's level; `keep` is the point to keep selected when it is still offered. */
  private loadPoints(topicId: string, keep: string): void {
    this.pointsSub?.unsubscribe();
    this.pointsSub = null;
    const control = this.form.controls.grammarPointId;
    this.points.set([]);
    control.setValue(keep);
    if (!topicId) {
      control.disable();
      this.pointsState.set('idle');
      return;
    }
    const level = this.levelOf(topicId);
    if (!level) {
      // Topic list not loaded yet: ask again once it is.
      control.disable();
      this.pointsState.set('loading');
      return;
    }
    this.pointsState.set('loading');
    control.disable();
    this.pointsSub = this.api.grammarPoints(level, topicId).subscribe({
      next: (list) => {
        this.points.set(list);
        control.setValue(list.some((p) => p.id === keep) ? keep : '');
        control.enable();
        this.pointsState.set('idle');
      },
      error: () => this.pointsState.set('error'),
    });
  }

  protected grammarError(): string | null {
    return this.serverFields()['grammarPointId'] ?? null;
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
    // A failed list load must not clear the point the lesson already has: leave it out then.
    if (this.pointsState() !== 'error') {
      input.grammarPointId = v.grammarPointId;
    }

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
      this.serverFields.set(body.fields as Partial<Record<FieldName | 'grammarPointId', string>>);
      this.serverError.set('Vui lòng sửa các ô được đánh dấu.');
    } else if (err instanceof ApiError && err.kind === 'network') {
      this.serverError.set('Không kết nối được máy chủ. Vui lòng thử lại.');
    } else {
      this.serverError.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
    }
  }
}
