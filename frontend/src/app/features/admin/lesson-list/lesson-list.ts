import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed, toObservable } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { catchError, EMPTY, firstValueFrom, forkJoin, switchMap } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { isRunning, JobKind, Level, LessonFilter, LessonSummary, LEVELS } from '../../../core/models/lesson';
import { lowRoadmaps, sortTopics, Topic } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { clampPage, pageCountOf, Pager, pageSlice, rangeText } from '../../../shared/components/pager/pager';
import { SearchOption, SearchSelect } from '../../../shared/components/search-select/search-select';
import { pollWhile } from '../../../shared/utils/poll-while';
import { AdminApiService } from '../admin-api.service';
import { StatusChip } from '../status-chip/status-chip';
import { Loading } from '../../../shared/components/loading/loading';

/** Lessons per page. */
export const PAGE_SIZE = 20;

interface ListData {
  lessons: LessonSummary[];
  topics: Topic[];
}

@Component({
  selector: 'lu-lesson-list',
  imports: [Loading, RouterLink, StatusChip, ConfirmDialog, SearchSelect, Pager],
  templateUrl: './lesson-list.html',
  styleUrl: './lesson-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonList {
  private readonly api = inject(AdminApiService);

  protected readonly levels = LEVELS;
  protected readonly filter = signal<LessonFilter>({ level: '', topicId: '' });
  private readonly reloads = signal(0);

  protected readonly data = signal<ListData | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly busy = signal(false);

  private readonly topics = computed(() => this.data()?.topics ?? []);
  /** Topic choices: every topic, as topics are shared by every level. */
  protected readonly topicOptions = computed<SearchOption[]>(() => [
    { value: '', label: 'Tất cả' },
    ...sortTopics(this.topics()).map((t) => ({ value: t.id, label: t.name })),
  ]);
  /** Number of roadmaps (topic and level) running low (details on the roadmap page). */
  protected readonly lowTopics = computed(() => lowRoadmaps(this.topics()).length);

  protected readonly page = signal(1);
  private readonly lessons = computed(() => this.data()?.lessons ?? []);
  protected readonly pageCount = computed(() => pageCountOf(this.lessons().length, PAGE_SIZE));
  /** The current page, kept inside the page count when the list shrinks (filter, delete). */
  protected readonly currentPage = computed(() => clampPage(this.page(), this.lessons().length, PAGE_SIZE));
  protected readonly pageLessons = computed(() => pageSlice(this.lessons(), this.page(), PAGE_SIZE));
  /** "21–40 / 57" */
  protected readonly rangeText = computed(() => rangeText(this.page(), this.lessons().length, PAGE_SIZE));

  // --- quick delete: lessons ticked on any page ---
  protected readonly selected = signal<ReadonlySet<string>>(new Set());
  /** Ticked lessons still in the list (deleted or filtered-out ones drop out). */
  private readonly selectedLessons = computed(() => this.lessons().filter((l) => this.selected().has(l.id)));
  protected readonly selectedCount = computed(() => this.selectedLessons().length);
  protected readonly allPageSelected = computed(
    () => this.pageLessons().length > 0 && this.pageLessons().every((l) => this.selected().has(l.id)),
  );
  protected readonly somePageSelected = computed(
    () => !this.allPageSelected() && this.pageLessons().some((l) => this.selected().has(l.id)),
  );
  protected readonly deleteOpen = signal(false);
  protected readonly deleteMessage = computed(() => {
    const inRoadmap = this.selectedLessons().filter((l) => l.inRoadmap).length;
    let text = `${this.selectedCount()} bài cùng chú thích sẽ bị xoá và không khôi phục được.`;
    if (inRoadmap > 0) {
      text += ` ${inRoadmap} bài đang trong lộ trình sẽ không bị xoá (cần gỡ khỏi lộ trình trước).`;
    }
    return text;
  });
  /** Result of the last quick delete; kept across the reload that follows it. */
  protected readonly notice = signal<string | null>(null);
  protected readonly deleteError = signal<string | null>(null);

  constructor() {
    toObservable(computed(() => ({ filter: this.filter(), reload: this.reloads() })))
      .pipe(
        switchMap(({ filter }) =>
          pollWhile(
            () => forkJoin({ lessons: this.api.list(filter), topics: this.api.topics() }),
            (d) => d.lessons.some(isRunning),
          ).pipe(
            catchError(() => {
              this.error.set('Không tải được danh sách bài. Vui lòng thử lại.');
              return EMPTY;
            }),
          ),
        ),
        takeUntilDestroyed(),
      )
      .subscribe((d) => {
        this.error.set(null);
        this.data.set(d);
      });
  }

  protected setLevel(value: string): void {
    this.page.set(1);
    this.filter.update((f) => ({ ...f, level: value as Level | '' }));
  }

  protected setTopic(value: string): void {
    this.page.set(1);
    this.filter.update((f) => ({ ...f, topicId: value }));
  }

  protected goTo(page: number): void {
    this.page.set(Math.min(Math.max(1, page), this.pageCount()));
  }

  protected toggle(id: string, checked: boolean): void {
    this.selected.update((s) => withIds(s, [id], checked));
  }

  /** Ticks or clears every lesson of the current page. */
  protected togglePage(checked: boolean): void {
    this.selected.update((s) => withIds(s, this.pageLessons().map((l) => l.id), checked));
  }

  protected clearSelection(): void {
    this.selected.set(new Set());
  }

  /** Deletes the ticked lessons one by one; those the server refuses (in a roadmap) stay ticked. */
  protected async deleteSelected(): Promise<void> {
    this.deleteOpen.set(false);
    const targets = this.selectedLessons();
    if (targets.length === 0 || this.busy()) {
      return;
    }
    this.notice.set(null);
    this.deleteError.set(null);
    await this.run(async () => {
      const failed: string[] = [];
      let reason: string | null = null;
      for (const lesson of targets) {
        try {
          await firstValueFrom(this.api.remove(lesson.id));
        } catch (err) {
          failed.push(lesson.id);
          reason ??= messageOf(err);
        }
      }
      this.selected.set(new Set(failed));
      const deleted = targets.length - failed.length;
      if (deleted > 0) {
        this.notice.set(`Đã xoá ${deleted} bài.`);
      }
      if (failed.length > 0) {
        this.deleteError.set(`Không xoá được ${failed.length} bài: ${reason ?? 'có lỗi xảy ra, vui lòng thử lại.'}`);
      }
    });
  }

  protected async retry(id: string, job: JobKind): Promise<void> {
    await this.run(() => firstValueFrom(this.api.retry(id, job)));
  }

  /** Adds a lesson at the end of the roadmap of its topic and level. */
  protected async addToRoadmap(lesson: LessonSummary): Promise<void> {
    await this.run(async () => {
      const rm = await firstValueFrom(this.api.topicRoadmap(lesson.topicId, lesson.level));
      await firstValueFrom(this.api.setTopicRoadmap(lesson.topicId, lesson.level, [...rm.lessons.map((l) => l.id), lesson.id]));
    });
  }

  private async run(action: () => Promise<unknown>): Promise<void> {
    this.busy.set(true);
    try {
      await action();
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Có lỗi xảy ra, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
      this.reloads.update((n) => n + 1);
    }
  }
}

/** The server's message, when the error has one. */
function messageOf(err: unknown): string | null {
  const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
  return body?.message ?? null;
}

/** A copy of `set` with `ids` added (checked) or removed. */
function withIds(set: ReadonlySet<string>, ids: string[], checked: boolean): Set<string> {
  const next = new Set(set);
  for (const id of ids) {
    if (checked) {
      next.add(id);
    } else {
      next.delete(id);
    }
  }
  return next;
}
