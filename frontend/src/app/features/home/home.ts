import { DOCUMENT } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { firstValueFrom, forkJoin } from 'rxjs';

import { accuracyOf, Dashboard, Stats } from '../../core/models/dashboard';
import { GoalView, Goals, PublicTopic } from '../../core/models/study';
import { formatScore } from '../../core/models/writing';
import { AuthService } from '../../core/services/auth.service';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { StudyApiService } from '../../core/services/study-api.service';
import { VocabApiService } from '../../core/services/vocab-api.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Icon } from '../../shared/components/icon/icon';
import { ProgressBar } from '../../shared/components/progress-bar/progress-bar';
import { FLOW_STEP_NAMES, LESSON_FLOW, readSavedStep, resumeIndex } from '../../shared/utils/lesson-flow';

/** The week in one line: lessons, words, writings (with their average); zero parts are left out. */
export function weekSummary(w: Stats): string {
  const parts: string[] = [];
  if (w.lessons.completed > 0) {
    parts.push(`${w.lessons.completed} bài`);
  }
  if (w.cards > 0) {
    parts.push(`${w.cards} từ`);
  }
  if (w.writing.submitted > 0) {
    const avg = w.writing.averageScore;
    parts.push(`${w.writing.submitted} bài viết${avg === null ? '' : ` (${formatScore(avg)}/5)`}`);
  }
  return parts.length ? parts.join(' · ') : 'Chưa học gì tuần này';
}

/**
 * The topic to study after the goal's: the first one after it in the level's suggested order that
 * has lessons and is not finished, else the first such one before it; null when none is left.
 */
export function nextTopic(goal: GoalView, topics: PublicTopic[], goals: Goals): PublicTopic | null {
  const finished = new Set(
    [goals.active, ...goals.others]
      .filter((g): g is GoalView => !!g && g.level === goal.level && g.totalLessons > 0 && g.completedLessons >= g.totalLessons)
      .map((g) => g.topicId),
  );
  const open = (t: PublicTopic) => t.id !== goal.topicId && t.lessonCount > 0 && !finished.has(t.id);
  const at = topics.findIndex((t) => t.id === goal.topicId);
  return topics.slice(at + 1).find(open) ?? topics.slice(0, Math.max(at, 0)).find(open) ?? null;
}

/**
 * The home page (F6, client sketch screen 2): greeting, progress of the lesson being studied, three
 * figures and the button into that lesson. Numbers are loaded every time the page opens, and again
 * when the app comes back to the screen: an installed app is mostly resumed on this page, and the
 * lesson shown may have been completed elsewhere or deleted in the meantime.
 */
@Component({
  selector: 'lu-home',
  imports: [Icon, ProgressBar, RouterLink],
  templateUrl: './home.html',
  styleUrl: './home.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Home {
  private readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly document = inject(DOCUMENT);
  private readonly auth = inject(AuthService);
  private readonly vocab = inject(VocabApiService);
  private readonly study = inject(StudyApiService);
  protected readonly notifier = inject(WritingNotifier);

  protected readonly data = signal<Dashboard | null>(null);
  /** Figures for the tiles; the page works without them ("—"). */
  protected readonly stats = signal<Stats | null>(null);
  /** This week's figures for the "Tuần này" card; the card is hidden without them. */
  protected readonly week = signal<Stats | null>(null);
  protected readonly weekLine = computed(() => {
    const w = this.week();
    return w ? weekSummary(w) : '';
  });
  protected readonly loadError = signal(false);
  /** Cards due now; the page works without it (0). */
  protected readonly dueToday = signal(0);
  protected readonly dueMinutes = computed(() => Math.max(1, Math.ceil((this.dueToday() * 20) / 60)));
  /** Once the goal's topic is finished: the topic suggested next; null when none is left. */
  protected readonly next = signal<PublicTopic | null>(null);
  protected readonly switching = signal(false);
  protected readonly switchError = signal(false);

  /** "Bài 3/12": the place of the lesson being studied in its topic. */
  protected readonly lessonPlace = computed(() => {
    const g = this.data()?.goal;
    return g && g.totalLessons > 0 ? `Bài ${Math.min(g.completedLessons + 1, g.totalLessons)}/${g.totalLessons}` : '';
  });

  /** The part of the email before "@", as there is no display name. */
  protected readonly stepCount = LESSON_FLOW.length;

  protected readonly name = computed(() => this.auth.currentUser()?.email.split('@')[0] ?? '');

  /**
   * Where the learner is in the lesson's five steps, as its page shows it: the step saved on this
   * device or the one after the last the server knows is done; -1 before anything is done.
   */
  private readonly flowIndex = computed(() => {
    const d = this.data();
    return d?.lesson ? resumeIndex(LESSON_FLOW, readSavedStep(d.lesson.id), d.steps) : -1;
  });

  protected readonly doneSteps = computed(() => Math.max(this.flowIndex(), 0));

  /** "Bước 3/5: Nghe", as on the lesson's page. */
  protected readonly stepLabel = computed(() => {
    const i = Math.max(this.flowIndex(), 0);
    return `Bước ${i + 1}/${LESSON_FLOW.length}: ${FLOW_STEP_NAMES[LESSON_FLOW[i]]}`;
  });

  protected readonly subtitle = computed(() => {
    const d = this.data();
    switch (d?.kind) {
      case 'noGoal':
        return 'Chọn một chủ đề để bắt đầu học nhé!';
      case 'noNewLesson':
        return d.goalCompleted ? 'Học tiếp chủ đề mới hoặc ôn lại từ đã học nhé!' : 'Ôn lại từ đã học trong lúc chờ bài mới nhé!';
      default: {
        const i = this.flowIndex();
        if (i < 0) {
          return 'Sẵn sàng cho bài mới chưa? Bắt đầu thôi!';
        }
        // The steps from here on, and those left behind for later (the server has them not done).
        const steps = d?.steps;
        const left = LESSON_FLOW.filter(
          (s, at) => at >= i || (s === 'read' || s === 'listen' || s === 'write' ? steps?.[s] !== 'done' : false),
        );
        const names = left.map((s) => FLOW_STEP_NAMES[s]).join(', ');
        return `Bài này còn ${left.length} bước nữa thôi (${names})!`;
      }
    }
  });

  protected readonly actionLabel = computed(() => {
    const a = this.data()?.action;
    if (!a) {
      return '';
    }
    const i = this.flowIndex();
    return i < 0 ? 'Bắt đầu học' : `Tiếp tục: ${FLOW_STEP_NAMES[LESSON_FLOW[i]]}`;
  });

  /** Comprehension and dictation together, as a whole percentage. */
  protected readonly accuracy = computed(() => {
    const s = this.stats();
    const a = s ? accuracyOf(s) : null;
    return a === null ? '—' : `${Math.round(a * 100)}%`;
  });

  constructor() {
    this.load();
    const reload = () => {
      if (this.document.visibilityState === 'visible') {
        this.load();
      }
    };
    this.document.addEventListener('visibilitychange', reload);
    this.destroyRef.onDestroy(() => this.document.removeEventListener('visibilitychange', reload));
  }

  protected load(): void {
    this.loadError.set(false);
    this.api
      .dashboard()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (d) => {
          this.data.set(d);
          this.loadNext(d.kind === 'noNewLesson' && d.goalCompleted ? d.goal : null);
        },
        error: () => this.loadError.set(true),
      });
    this.api
      .stats()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (s) => this.stats.set(s), error: () => undefined });
    this.api
      .stats('week')
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (s) => this.week.set(s), error: () => this.week.set(null) });
    this.vocab
      .due(1)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (list) => this.dueToday.set(list.total), error: () => undefined });
  }

  /** Looks for the topic to suggest once the goal's is finished; without it the card offers the list. */
  private loadNext(goal: GoalView | null): void {
    this.next.set(null);
    if (!goal) {
      return;
    }
    forkJoin([this.study.topics(goal.level), this.study.goals()])
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: ([topics, goals]) => this.next.set(nextTopic(goal, topics, goals)), error: () => undefined });
  }

  /** Makes the suggested topic the goal; the page then shows its first lesson. */
  protected async studyNext(): Promise<void> {
    const t = this.next();
    if (!t || this.switching()) {
      return;
    }
    this.switching.set(true);
    this.switchError.set(false);
    try {
      await firstValueFrom(this.study.setGoal(t.id, t.level));
      this.load();
    } catch {
      this.switchError.set(true);
    } finally {
      this.switching.set(false);
    }
  }
}
