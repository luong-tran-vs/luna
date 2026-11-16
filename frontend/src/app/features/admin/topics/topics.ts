import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  signal,
} from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { NgTemplateOutlet } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Level, LEVELS } from '../../../core/models/lesson';
import { sortTopics, sortTopicsAt, Topic } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { serverUrl } from '../../../core/api-url';
import { TopicImageList } from '../../../core/models/topic-images';
import { engineLabel, ImageEngine, MAX_IMAGE_BATCH } from '../../../core/models/word-bank';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { TopicImageBatchService } from '../topic-images/topic-image-batch.service';
import { TopicImagesApiService } from '../topic-images/topic-images-api.service';

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
 * default), in the order learners of that level see them, which ↑ ↓ change; or every topic, by
 * name. Add and edit in place, delete with confirmation.
 */
@Component({
  selector: 'lu-topics',
  imports: [
    Loading,
    ConfirmDialog,
    EngineDialog,
    NgTemplateOutlet,
    ReactiveFormsModule,
    RouterLink,
  ],
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
      count: list.filter((t) => t.levels.some((l) => l.level === level && l.lessonCount > 0))
        .length,
    })).filter((f) => f.count > 0);
    return [...levels, { id: 'all', label: 'Tất cả', count: list.length }];
  });
  protected readonly shownFilter = computed<TopicFilter>(() => {
    const options = this.filters();
    return options.find((f) => f.id === this.filter())?.id ?? options[0].id;
  });
  /**
   * The topics of the picked filter; a long catalogue stays short. At a level they are in the order
   * learners of the level see them, which ↑ ↓ change; "Tất cả" is by name.
   */
  protected readonly shown = computed(() => {
    const list = this.topics() ?? [];
    const f = this.shownFilter();
    return f === 'all'
      ? sortTopics(list)
      : sortTopicsAt(list, f).filter((t) =>
          t.levels.some((l) => l.level === f && l.lessonCount > 0),
        );
  });
  /** The level being ordered, or null on "Tất cả". */
  protected readonly orderLevel = computed(() => {
    const f = this.shownFilter();
    return f === 'all' ? null : f;
  });
  protected readonly ordering = signal(false);
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

  // --- Topic pictures (added 2026-11-16): one banner per topic, drawn one at a time or for every
  // topic lacking one, in a background batch. ---
  private readonly imagesApi = inject(TopicImagesApiService);
  private readonly batches = inject(TopicImageBatchService);
  protected readonly images = signal<TopicImageList | null>(null);
  /** The picture URL of each topic that has one. */
  protected readonly imageOf = computed(() => {
    const m = new Map<string, string>();
    for (const im of this.images()?.images ?? []) {
      m.set(im.topicId, serverUrl(im.imageUrl));
    }
    return m;
  });
  /** Every topic without a picture, by name. */
  protected readonly missingImages = computed(() =>
    sortTopics(this.topics() ?? []).filter((t) => !this.imageOf().has(t.id)),
  );
  protected readonly imageBatch = computed(() => {
    const b = this.batches.status();
    return b?.running ? b : null;
  });
  protected readonly drawingTopics = computed(() => new Set(this.batches.status()?.drawing ?? []));
  protected readonly imageEngine = computed(() => {
    const e = this.batches.status()?.engine;
    return e ? engineLabel(e) : '';
  });
  /** The topics the engine dialog is about to draw; null while it is closed. */
  protected readonly toDraw = signal<Topic[] | null>(null);
  protected readonly removingImage = signal<Topic | null>(null);
  protected readonly imageMessage = signal<string | null>(null);
  protected readonly drawHeading = computed(() => {
    const list = this.toDraw() ?? [];
    return list.length === 1
      ? `Tạo ảnh cho chủ đề “${list[0].name}”?`
      : `Tạo ảnh cho ${list.length} chủ đề?`;
  });
  protected readonly drawMessage = computed(() => {
    const list = this.toDraw() ?? [];
    if (list.length === 1) {
      return this.imageOf().has(list[0].id)
        ? 'Ảnh mới thay ảnh đang có. Máy chủ vẽ ngầm, bạn có thể rời trang.'
        : 'Máy chủ vẽ ngầm, xong thì ảnh hiện ở đây. Bạn có thể rời trang.';
    }
    const rest = this.missingImages().length - list.length;
    return (
      `Máy chủ vẽ lần lượt ${list.length} chủ đề chưa có ảnh, chạy ngầm, xong ảnh nào lưu ảnh đó. Bạn có thể rời trang.` +
      (rest > 0
        ? ` Mỗi lượt tối đa ${MAX_IMAGE_BATCH} ảnh: ${rest} chủ đề còn lại vẽ ở lượt sau.`
        : '')
    );
  });

  private loadImages(): void {
    this.imagesApi.list().subscribe({
      next: (d) => this.images.set(d),
      // Without the list the topics show no picture; the rest of the page still works.
      error: () => this.images.set({ images: [] }),
    });
  }

  protected askDrawMissing(): void {
    this.imageMessage.set(null);
    this.toDraw.set(this.missingImages().slice(0, MAX_IMAGE_BATCH));
  }

  protected askDraw(t: Topic): void {
    this.imageMessage.set(null);
    this.toDraw.set([t]);
  }

  /** Starts the batch with the engine the admin picked in the dialog. */
  protected drawImages(engine: ImageEngine): void {
    const list = this.toDraw() ?? [];
    this.toDraw.set(null);
    if (list.length === 0) {
      return;
    }
    this.imagesApi.startBatch({ topicIds: list.map((t) => t.id), engine }).subscribe({
      next: (b) => this.batches.set(b),
      error: (err) =>
        this.imageMessage.set(apiMessage(err, 'Không bắt đầu tạo ảnh được, vui lòng thử lại.')),
    });
  }

  protected stopImages(): void {
    this.imagesApi.stopBatch().subscribe({
      next: (b) => this.batches.set(b),
      error: () => this.imageMessage.set('Không dừng được, vui lòng thử lại.'),
    });
  }

  protected removeImage(): void {
    const t = this.removingImage();
    this.removingImage.set(null);
    if (!t) {
      return;
    }
    this.imagesApi.remove(t.id).subscribe({
      next: () => this.loadImages(),
      error: () => this.imageMessage.set(`Không xoá được ảnh của “${t.name}”, vui lòng thử lại.`),
    });
  }

  protected readonly form = inject(NonNullableFormBuilder).group({
    name: ['', [Validators.required, Validators.maxLength(60)]],
    description: ['', Validators.maxLength(200)],
  });

  constructor() {
    this.load();
    this.loadImages();
    inject(DestroyRef).onDestroy(this.batches.watch());
    // A picture saved by the batch shows at once: reload the pictures when the batch saves one more.
    let saved = -1;
    effect(() => {
      const n = this.batches.status()?.saved.length ?? 0;
      if (saved >= 0 && n !== saved) {
        this.loadImages();
      }
      saved = n;
    });
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
      await firstValueFrom(
        target === 'new' ? this.api.createTopic(input) : this.api.updateTopic(target, input),
      );
      this.editing.set(null);
      // A new topic has no lesson yet: only "Tất cả" lists it.
      if (target === 'new') {
        this.filter.set('all');
      }
      this.load();
    } catch (err) {
      const body =
        err instanceof ApiError
          ? (err.body as { message?: string; fields?: Record<string, string> } | null)
          : null;
      if (body?.fields) {
        this.serverFields.set(body.fields);
      } else {
        this.error.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
      }
    } finally {
      this.pending.set(false);
    }
  }

  /**
   * ↑ / ↓: moves a topic one place in the order of the level being shown and saves the whole list
   * as that level's order (so every topic shown gets a place).
   */
  protected async move(index: number, delta: -1 | 1): Promise<void> {
    const level = this.orderLevel();
    const ids = this.shown().map((t) => t.id);
    const to = index + delta;
    if (!level || to < 0 || to >= ids.length) {
      return;
    }
    [ids[index], ids[to]] = [ids[to], ids[index]];
    await this.saveOrder(level, ids);
  }

  private async saveOrder(level: Level, ids: string[]): Promise<boolean> {
    if (this.ordering()) {
      return false;
    }
    this.ordering.set(true);
    this.error.set(null);
    try {
      this.topics.set(await firstValueFrom(this.api.setTopicOrder(level, ids)));
      return true;
    } catch {
      this.error.set('Không lưu được thứ tự chủ đề, vui lòng thử lại.');
      return false;
    } finally {
      this.ordering.set(false);
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

/** The server's message of a failed request (a field error first), else fallback. */
function apiMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field = Object.values(body?.fields ?? {})[0];
    if (field) {
      return field;
    }
    if (typeof body?.message === 'string' && body.message) {
      return body.message;
    }
  }
  return fallback;
}
