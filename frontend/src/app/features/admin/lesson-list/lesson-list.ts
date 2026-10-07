import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed, toObservable } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { catchError, EMPTY, firstValueFrom, forkJoin, switchMap } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { isRunning, JobKind, Level, LessonFilter, LessonSummary, LEVELS } from '../../../core/models/lesson';
import { lowRoadmaps, sortTopics, Topic } from '../../../core/models/topic';
import { pollWhile } from '../../../shared/utils/poll-while';
import { AdminApiService } from '../admin-api.service';
import { StatusChip } from '../status-chip/status-chip';
import { Loading } from '../../../shared/components/loading/loading';

interface ListData {
  lessons: LessonSummary[];
  topics: Topic[];
}

@Component({
  selector: 'lu-lesson-list',
  imports: [Loading, RouterLink, StatusChip],
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
  protected readonly topicOptions = computed(() => sortTopics(this.topics()));
  /** Number of roadmaps (topic and level) running low (details on the roadmap page). */
  protected readonly lowTopics = computed(() => lowRoadmaps(this.topics()).length);

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
    this.filter.update((f) => ({ ...f, level: value as Level | '' }));
  }

  protected setTopic(value: string): void {
    this.filter.update((f) => ({ ...f, topicId: value }));
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
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
      this.reloads.update((n) => n + 1);
    }
  }
}
