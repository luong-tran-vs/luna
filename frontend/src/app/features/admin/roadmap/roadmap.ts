import { CdkDrag, CdkDragDrop, CdkDragHandle, CdkDropList, moveItemInArray } from '@angular/cdk/drag-drop';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { LessonSummary } from '../../../core/models/lesson';
import { groupByLevel, Topic, topicLabel, TopicRoadmap } from '../../../core/models/topic';
import { AdminApiService } from '../admin-api.service';
import { StatusChip } from '../status-chip/status-chip';

/** One roadmap per topic (F14): choose a topic, then add, remove and reorder its lessons. */
@Component({
  selector: 'lu-roadmap',
  imports: [RouterLink, CdkDropList, CdkDrag, CdkDragHandle, StatusChip],
  templateUrl: './roadmap.html',
  styleUrl: './roadmap.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Roadmap {
  private readonly api = inject(AdminApiService);
  private readonly router = inject(Router);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);

  protected readonly topicLabel = topicLabel;
  protected readonly topics = signal<Topic[] | null>(null);
  protected readonly topicGroups = computed(() => groupByLevel(this.topics() ?? []));
  protected readonly lowTopics = computed(() => (this.topics() ?? []).filter((t) => t.warning));
  protected readonly selectedId = signal(inject(ActivatedRoute).snapshot.queryParamMap.get('topicId') ?? '');

  protected readonly data = signal<TopicRoadmap | null>(null);
  /** All lessons of the selected topic, to offer those not in the roadmap. */
  private readonly topicLessons = signal<LessonSummary[]>([]);
  protected readonly lessons = computed(() => this.data()?.lessons ?? []);
  protected readonly available = computed(() => {
    const inRoadmap = new Set(this.lessons().map((l) => l.id));
    return this.topicLessons().filter((l) => !inRoadmap.has(l.id));
  });
  protected readonly error = signal<string | null>(null);
  protected readonly saving = signal(false);

  protected readonly warning = computed(() => {
    const d = this.data();
    if (!d?.warning) {
      return null;
    }
    return d.remaining === 0
      ? 'Lộ trình chưa có bài. Thêm bài của chủ đề này bên dưới.'
      : `Lộ trình chỉ còn ${d.remaining} bài chưa học. Hãy thêm bài mới.`;
  });

  constructor() {
    firstValueFrom(this.api.topics())
      .then((list) => {
        this.topics.set(list);
        if (this.selectedId()) {
          void this.load(this.selectedId());
        }
      })
      .catch(() => this.error.set('Không tải được danh sách chủ đề.'));
  }

  protected warningText(t: Topic): string {
    return t.remaining === 0
      ? `${topicLabel(t)}: lộ trình chưa có bài`
      : `${topicLabel(t)}: còn ${t.remaining} bài chưa học`;
  }

  protected select(id: string): void {
    this.selectedId.set(id);
    void this.router.navigate([], { queryParams: { topicId: id || null }, replaceUrl: true });
    this.data.set(null);
    this.topicLessons.set([]);
    if (id) {
      void this.load(id);
    }
  }

  private async load(id: string): Promise<void> {
    this.error.set(null);
    try {
      const [roadmap, lessons] = await Promise.all([
        firstValueFrom(this.api.topicRoadmap(id)),
        firstValueFrom(this.api.list({ topicId: id })),
      ]);
      if (this.selectedId() === id) {
        this.data.set(roadmap);
        this.topicLessons.set(lessons);
      }
    } catch {
      this.error.set('Không tải được lộ trình.');
    }
  }

  protected drop(event: CdkDragDrop<LessonSummary[]>): void {
    if (event.previousIndex === event.currentIndex) {
      return;
    }
    const next = [...this.lessons()];
    moveItemInArray(next, event.previousIndex, event.currentIndex);
    void this.save(next);
  }

  protected move(index: number, delta: -1 | 1): void {
    const next = [...this.lessons()];
    const id = next[index].id;
    moveItemInArray(next, index, index + delta);
    void this.save(next).then(() => this.refocus(id, delta));
  }

  protected remove(index: number): void {
    void this.save(this.lessons().filter((_, i) => i !== index));
  }

  protected add(lesson: LessonSummary): void {
    void this.save([...this.lessons(), lesson]);
  }

  /** Shows the new order at once, then saves it; on failure restores the previous order. */
  private async save(next: LessonSummary[]): Promise<void> {
    const previous = this.data();
    const id = this.selectedId();
    if (!previous || !id) {
      return;
    }
    this.data.set({ ...previous, lessons: next });
    this.error.set(null);
    this.saving.set(true);
    try {
      const saved = await firstValueFrom(this.api.setTopicRoadmap(id, next.map((l) => l.id)));
      this.data.set(saved);
      this.topics.update((list) => list?.map((t) => (t.id === saved.topic.id ? saved.topic : t)) ?? null);
    } catch {
      this.data.set(previous);
      this.error.set('Không lưu được lộ trình. Thứ tự đã được khôi phục, vui lòng thử lại.');
    } finally {
      this.saving.set(false);
    }
  }

  /**
   * Keeps keyboard focus on the moved lesson after the list re-renders: the same direction
   * button, or the opposite one when the lesson reached the top or bottom.
   */
  private refocus(id: string, delta: -1 | 1): void {
    afterNextRender(
      () => {
        const row = Array.from(this.host.nativeElement.querySelectorAll<HTMLElement>('li[data-id]')).find(
          (li) => li.dataset['id'] === id,
        );
        const same = row?.querySelector<HTMLButtonElement>(`button[data-move="${delta < 0 ? 'up' : 'down'}"]`);
        const other = row?.querySelector<HTMLButtonElement>(`button[data-move="${delta < 0 ? 'down' : 'up'}"]`);
        (same && !same.disabled ? same : other)?.focus();
      },
      { injector: this.injector },
    );
  }
}
