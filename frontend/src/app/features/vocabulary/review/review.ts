import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  signal,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { DueList, ReviewForecast, ReviewMode } from '../../../core/models/vocab';
import { StudyApiService } from '../../../core/services/study-api.service';
import { FocusMode } from '../../../core/services/focus-mode.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { Icon, IconName } from '../../../shared/components/icon/icon';
import { ReviewSession } from '../../../shared/components/review-session/review-session';
import { Loading } from '../../../shared/components/loading/loading';
import { suggested } from '../lesson-review/lesson-review-logic';
import { LessonItem, loadLessonItems } from '../lesson-list/lesson-items';
import { LessonList } from '../lesson-list/lesson-list';

const SESSION_SIZE = 100;

/** How many lessons the tab lists, those needing a review most; the rest are behind "Xem tất cả". */
export const LESSONS_SHOWN = 4;

/** About how long one card takes, for the estimate of the session. */
const SECONDS_PER_CARD = 25;

const MODES: { value: ReviewMode; icon: IconName; title: string; hint: string }[] = [
  { value: 'flip', icon: 'cards', title: 'Xem từ', hint: 'chọn nghĩa' },
  { value: 'listen', icon: 'headphones', title: 'Nghe', hint: 'rồi gõ từ' },
];

const WEEKDAYS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];
const WEEKDAY_NAMES = ['Chủ nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy'];

const dateTime = new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' });
const clock = new Intl.DateTimeFormat('vi-VN', { hour: '2-digit', minute: '2-digit' });

/** "08:30 hôm nay", "00:00 ngày mai", else the date and time. */
export function dueWhen(at: Date, now: Date): string {
  const tomorrow = new Date(now);
  tomorrow.setDate(now.getDate() + 1);
  if (at.toDateString() === now.toDateString()) {
    return `${clock.format(at)} hôm nay`;
  }
  if (at.toDateString() === tomorrow.toDateString()) {
    return `${clock.format(at)} ngày mai`;
  }
  return dateTime.format(at);
}

/** One bar of the 7-day chart. */
interface DayBar {
  label: string;
  /** Read by screen readers, e.g. "Thứ Sáu 16/10: 4 thẻ". */
  name: string;
  count: number;
  /** Height in % of the busiest day. */
  height: number;
  today: boolean;
}

/**
 * The review tab (design/on-tap-theo-bai.png, design/on-tap-bo-cuc.png): the lesson to review
 * today and the LESSONS_SHOWN lessons needing it most (all of them behind "Xem tất cả"), each
 * opening a lesson review; then the free review of every card due now (F5) in the mode the
 * learner picks, the cards by stage and the next 7 days. A session takes the whole screen.
 */
@Component({
  selector: 'lu-review',
  imports: [Loading, Icon, LessonList, ReviewSession, RouterLink],
  templateUrl: './review.html',
  styleUrl: './review.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Review {
  private readonly api = inject(VocabApiService);
  private readonly study = inject(StudyApiService);
  private readonly focusMode = inject(FocusMode);
  /** The lesson to go on with after the review, when the learner came from a finished lesson. */
  protected readonly nextLesson = inject(ActivatedRoute).snapshot.queryParamMap.get('next');

  protected readonly modes = MODES;
  /** The lessons learned, by need; null while loading or when it failed (the section is left out). */
  protected readonly lessons = signal<LessonItem[] | null>(null);
  protected readonly shownLessons = computed(() => this.lessons()?.slice(0, LESSONS_SHOWN) ?? []);
  protected readonly suggestion = computed(() => {
    const rows = this.lessons();
    return rows ? suggested(rows, Date.now()) : null;
  });
  /** Meanings of the latest saved words: the wrong options of the multiple choice. */
  protected readonly pool = signal<string[]>([]);
  protected readonly due = signal<DueList | null>(null);
  /** Null while loading or when it failed: the stage counts and the chart are left out. */
  protected readonly forecast = signal<ReviewForecast | null>(null);
  protected readonly loadError = signal(false);
  protected readonly mode = signal<ReviewMode>('flip');
  protected readonly started = signal(false);
  protected readonly done = signal(false);
  protected readonly nextDue = computed(() => {
    const next = this.due()?.nextDue;
    return next ? dueWhen(new Date(next), new Date()) : null;
  });
  protected readonly minutes = computed(() =>
    Math.max(1, Math.round(((this.due()?.cards.length ?? 0) * SECONDS_PER_CARD) / 60)),
  );
  protected readonly cardCount = computed(() => {
    const f = this.forecast();
    return f ? f.new + f.learning + f.known : null;
  });
  protected readonly week = computed<DayBar[]>(() => {
    const days = this.forecast()?.days ?? [];
    const most = Math.max(1, ...days.map((d) => d.count));
    return days.map((d, i) => {
      const [y, m, day] = d.day.split('-').map(Number);
      const weekday = new Date(Date.UTC(y, m - 1, day)).getUTCDay();
      const date = `${day}/${m}`;
      return {
        label: i === 0 ? 'Nay' : WEEKDAYS[weekday],
        name: `${i === 0 ? 'Hôm nay' : WEEKDAY_NAMES[weekday]} ${date}: ${d.count} thẻ`,
        count: d.count,
        height: (d.count / most) * 100,
        today: i === 0,
      };
    });
  });

  constructor() {
    // The session takes the whole screen: no tab bar under its buttons.
    effect(() => this.focusMode.on.set(this.started() && !this.done()));
    inject(DestroyRef).onDestroy(() => this.focusMode.on.set(false));
    this.load();
  }

  protected load(): void {
    this.due.set(null);
    this.started.set(false);
    this.done.set(false);
    this.loadError.set(false);
    this.api.due(SESSION_SIZE).subscribe({
      next: (list) => this.due.set(list),
      error: () => this.loadError.set(true),
    });
    this.api.forecast().subscribe({
      next: (f) => this.forecast.set(f),
      error: () => this.forecast.set(null),
    });
    this.api.list({}).subscribe({
      next: (page) => this.pool.set(page.cards.map((c) => c.meaningVi)),
      error: () => this.pool.set([]),
    });
    loadLessonItems(this.study, this.api).subscribe({
      next: (items) => this.lessons.set(items),
      error: () => this.lessons.set(null),
    });
  }

  protected onModeKeydown(event: KeyboardEvent): void {
    if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)) {
      event.preventDefault();
      const next = this.mode() === 'flip' ? 'listen' : 'flip';
      this.mode.set(next);
      const group = (event.currentTarget as HTMLElement).closest('[role="radiogroup"]');
      group?.querySelectorAll<HTMLElement>('[role="radio"]')[next === 'flip' ? 0 : 1]?.focus();
    }
  }
}
