import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { GoalView, MyLessons as MyLessonsData } from '../../../core/models/study';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';

/**
 * The learner's lessons (L, client sketch screen 4): one numbered path — lessons studied (reopen
 * freely), the one being studied, then the locked upcoming ones — under the topic being studied.
 * Only lessons that exist are shown: no empty "today" box (updated 2026-10-02).
 */
@Component({
  selector: 'lu-my-lessons',
  imports: [Loading, DatePipe, Icon, RouterLink],
  templateUrl: './my-lessons.html',
  styleUrl: './my-lessons.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MyLessons {
  protected readonly data = signal<MyLessonsData | null>(null);
  protected readonly goal = signal<GoalView | null>(null);
  protected readonly error = signal(false);

  /** Oldest first, so the numbers follow the order the lessons were studied. */
  protected readonly completed = computed(() => [...(this.data()?.completed ?? [])].reverse());
  /** Number of the lesson being studied, right after the completed ones. */
  protected readonly currentNumber = computed(() => this.completed().length + 1);
  protected readonly upcomingStart = computed(() => this.currentNumber() + (this.data()?.current ? 1 : 0));
  protected readonly title = computed(() => {
    const g = this.goal();
    return g ? `${g.level} · ${g.topicName}` : 'Bài học';
  });

  constructor() {
    const api = inject(StudyApiService);
    api.myLessons().subscribe({ next: (d) => this.data.set(d), error: () => this.error.set(true) });
    // The header names the topic; without it the list still works.
    api.goals().subscribe({ next: (g) => this.goal.set(g.active), error: () => undefined });
  }
}
