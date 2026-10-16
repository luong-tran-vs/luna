import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { StudyApiService } from '../../../core/services/study-api.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { LessonFilter, LessonItem, loadLessonItems, matches } from '../lesson-list/lesson-items';
import { LessonList } from '../lesson-list/lesson-list';

const FILTERS: { value: LessonFilter; label: string }[] = [
  { value: 'all', label: 'Tất cả' },
  { value: 'due', label: 'Cần ôn' },
  { value: 'new', label: 'Chưa ôn' },
];

/**
 * "Bài đã học": every lesson learned, by how much it needs a review, with a quick filter
 * (design/on-tap-bo-cuc.png). The review tab lists only the first few.
 */
@Component({
  selector: 'lu-all-lessons',
  imports: [Icon, LessonList, Loading, RouterLink],
  templateUrl: './all-lessons.html',
  styleUrl: './all-lessons.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AllLessons {
  private readonly study = inject(StudyApiService);
  private readonly api = inject(VocabApiService);

  /** Null while loading. */
  protected readonly lessons = signal<LessonItem[] | null>(null);
  protected readonly loadError = signal(false);
  protected readonly filter = signal<LessonFilter>('all');

  protected readonly filters = computed(() => {
    const rows = this.lessons() ?? [];
    return FILTERS.map((f) => ({ ...f, count: rows.filter((r) => matches(r, f.value)).length }));
  });
  protected readonly shown = computed(() =>
    (this.lessons() ?? []).filter((r) => matches(r, this.filter())),
  );

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.lessons.set(null);
    loadLessonItems(this.study, this.api).subscribe({
      next: (items) => this.lessons.set(items),
      error: () => this.loadError.set(true),
    });
  }
}
