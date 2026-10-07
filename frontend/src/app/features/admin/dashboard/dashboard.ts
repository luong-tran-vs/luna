import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom, forkJoin } from 'rxjs';

import { GrammarLessonSummary, GrammarReportGroup } from '../../../core/models/grammar-admin';
import { LessonSummary } from '../../../core/models/lesson';
import { lowRoadmaps, Topic } from '../../../core/models/topic';
import { Icon, IconName } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { ProgressBar } from '../../../shared/components/progress-bar/progress-bar';
import { AdminApiService } from '../admin-api.service';

interface DashboardData {
  lessons: LessonSummary[];
  topics: Topic[];
  grammar: GrammarLessonSummary[];
  reports: GrammarReportGroup[];
}

/** A pending job on the dashboard: "{count} {label}", linking to the page where it is done. */
export interface TodoItem {
  count: number;
  label: string;
  link: string;
}

interface QuickAction {
  label: string;
  hint: string;
  link: string;
  icon: IconName;
  tone: 'listen' | 'accent' | 'primary';
}

const QUICK_ACTIONS: readonly QuickAction[] = [
  { label: 'Thêm bài', hint: 'Dán bài đọc có sẵn', link: '/admin/lessons/new', icon: 'plus', tone: 'listen' },
  { label: 'Sinh bài bằng AI', hint: 'Sinh bài vào lộ trình chủ đề', link: '/admin/roadmap', icon: 'sparkles', tone: 'accent' },
  { label: 'Ngữ pháp', hint: 'Soạn và duyệt bài ngữ pháp', link: '/admin/grammar', icon: 'grammar', tone: 'primary' },
];

const RECENT_COUNT = 3;

/** Pending jobs computed from the lists; jobs with nothing to do are left out. */
export function todoItems(d: DashboardData): TodoItem[] {
  const count = <T>(list: readonly T[], test: (x: T) => boolean) => list.filter(test).length;
  const items: TodoItem[] = [
    { count: count(d.lessons, (l) => (l.flags ?? 0) > 0), label: 'bài có chỗ AI đánh dấu cần xem', link: '/admin/lessons' },
    { count: count(d.lessons, (l) => l.annotationStatus === 'failed'), label: 'bài chú thích bị lỗi', link: '/admin/lessons' },
    { count: count(d.lessons, (l) => l.annotationStatus === 'done' && !l.checked), label: 'bài chưa kiểm tra bằng AI', link: '/admin/lessons' },
    { count: count(d.lessons, (l) => !l.inRoadmap), label: 'bài chưa vào lộ trình', link: '/admin/lessons' },
    { count: lowRoadmaps(d.topics).length, label: 'lộ trình sắp hết bài chưa học', link: '/admin/roadmap' },
    { count: d.reports.reduce((n, r) => n + r.count, 0), label: 'báo lỗi bài tập ngữ pháp từ người học', link: '/admin/grammar' },
    { count: count(d.grammar, (g) => g.status === 'draft'), label: 'bài ngữ pháp còn là bản nháp', link: '/admin/grammar' },
  ];
  return items.filter((i) => i.count > 0);
}

/** "Chào buổi sáng!" from 4h until noon, "buổi chiều" until 18h, then "buổi tối". */
export function greeting(hour: number): string {
  if (hour >= 4 && hour < 12) {
    return 'Chào buổi sáng!';
  }
  return hour >= 12 && hour < 18 ? 'Chào buổi chiều!' : 'Chào buổi tối!';
}

/** "Thứ Tư, 07/10/2026" */
export function dayLabel(date: Date): string {
  const weekday = new Intl.DateTimeFormat('vi-VN', { weekday: 'long' }).format(date);
  const day = new Intl.DateTimeFormat('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(date);
  return `${weekday.charAt(0).toUpperCase()}${weekday.slice(1)}, ${day}`;
}

/**
 * Admin home: a greeting with how many jobs are pending, how much of the content is checked,
 * shortcuts to the main tasks, today's pending jobs, content counts and the newest lessons.
 */
@Component({
  selector: 'lu-admin-dashboard',
  imports: [Icon, Loading, ProgressBar, RouterLink],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard {
  private readonly api = inject(AdminApiService);
  private readonly now = new Date();

  protected readonly actions = QUICK_ACTIONS;
  protected readonly greeting = greeting(this.now.getHours());
  protected readonly today = dayLabel(this.now);

  protected readonly data = signal<DashboardData | null>(null);
  protected readonly loadError = signal(false);

  protected readonly todos = computed(() => {
    const d = this.data();
    return d ? todoItems(d) : [];
  });
  protected readonly todoTotal = computed(() => this.todos().reduce((n, t) => n + t.count, 0));

  protected readonly lessonCount = computed(() => this.data()?.lessons.length ?? 0);
  protected readonly verifiedCount = computed(() => this.data()?.lessons.filter((l) => l.verified).length ?? 0);
  protected readonly roadmapCount = computed(() => this.data()?.lessons.filter((l) => l.inRoadmap).length ?? 0);
  protected readonly checkedPercent = computed(() => {
    const total = this.lessonCount();
    return total > 0 ? Math.round((this.verifiedCount() / total) * 100) : 0;
  });

  protected readonly stats = computed(() => {
    const d = this.data();
    return [
      { label: 'Bài học', value: d?.lessons.length ?? 0, icon: 'book' as IconName, tone: 'tone-listen' },
      { label: 'Chủ đề', value: d?.topics.length ?? 0, icon: 'layers' as IconName, tone: 'tone-accent' },
      {
        label: 'Bài ngữ pháp đã xuất bản',
        value: d?.grammar.filter((g) => g.status === 'published').length ?? 0,
        icon: 'grammar' as IconName,
        tone: 'tone-read',
      },
    ];
  });

  /** The newest lessons first. */
  protected readonly recent = computed(() =>
    [...(this.data()?.lessons ?? [])].sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, RECENT_COUNT),
  );

  constructor() {
    void this.load();
  }

  protected async load(): Promise<void> {
    this.loadError.set(false);
    try {
      this.data.set(
        await firstValueFrom(
          forkJoin({
            lessons: this.api.list(),
            topics: this.api.topics(),
            grammar: this.api.grammarLessons(),
            reports: this.api.grammarReports(),
          }),
        ),
      );
    } catch {
      this.loadError.set(true);
    }
  }
}
