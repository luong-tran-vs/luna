import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { forkJoin, Subscription } from 'rxjs';

import { Dashboard, Stats as StatsData, StatsPeriod } from '../../core/models/dashboard';
import { formatScore } from '../../core/models/writing';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { Icon, IconName } from '../../shared/components/icon/icon';
import { ProgressRing } from '../../shared/components/progress-ring/progress-ring';
import { Loading } from '../../shared/components/loading/loading';
import { badges } from './badges';

/** Writing scores go from 1 to 5 (F8). */
const MAX_SCORE = 5;

const PERIODS: readonly { id: StatsPeriod; label: string; scope: string }[] = [
  { id: 'week', label: 'Tuần', scope: 'tuần này' },
  { id: 'month', label: 'Tháng', scope: 'tháng này' },
  { id: 'all', label: 'Tổng', scope: 'từ trước tới nay' },
];

interface SkillBar {
  label: string;
  tone: 'read' | 'listen' | 'write';
  /** 0–100, null when there is no data yet. */
  percent: number | null;
  /** Shown next to the bar: "75%", "3,8/5" or "—". */
  value: string;
  /** What the bar measures, for screen readers. */
  hint: string;
}

/** One line of the period's figures: "Từ vựng · 120 từ". */
interface CountRow {
  label: string;
  icon: IconName;
  tone: string;
  /** "120 từ", or "—" when the app does not record it. */
  value: string;
}

/** "75%" for a 0–1 rate, "—" when nothing was measured. */
function percentText(rate: number | null | undefined): string {
  return rate === null || rate === undefined ? '—' : `${Math.round(rate * 100)}%`;
}

/**
 * The stats page (F6; design/ungdunghoctienganh.png, screen 8 "Lộ trình & Tiến độ"): this week's,
 * this month's or every day's figures by skill next to the roadmap ring, the accuracy by skill,
 * and badges for milestones.
 */
@Component({
  selector: 'lu-stats',
  imports: [Loading, Icon, ProgressRing, RouterLink],
  templateUrl: './stats.html',
  styleUrl: './stats.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Stats {
  private readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly periods = PERIODS;
  protected readonly period = signal<StatsPeriod>('week');
  /** Every-day figures (badges) and the dashboard (streak, roadmap). */
  protected readonly allTime = signal<StatsData | null>(null);
  protected readonly dashboard = signal<Dashboard | null>(null);
  /** Figures of the chosen period; null while they load. */
  private readonly periodFigures = signal<StatsData | null>(null);
  protected readonly loadError = signal(false);
  protected readonly periodError = signal(false);
  private periodLoad?: Subscription;

  protected readonly figures = computed(() => (this.period() === 'all' ? this.allTime() : this.periodFigures()));
  protected readonly scope = computed(() => PERIODS.find((p) => p.id === this.period())!.scope);

  /** The roadmap done, 0–100; null without a goal. */
  protected readonly goalPercent = computed(() => {
    const g = this.dashboard()?.goal;
    return g && g.totalLessons > 0 ? (g.completedLessons / g.totalLessons) * 100 : null;
  });
  protected readonly goalLine = computed(() => {
    const g = this.dashboard()?.goal;
    return g ? `${g.completedLessons}/${g.totalLessons} bài · ${g.level} ${g.topicName}` : 'Chưa chọn lộ trình';
  });

  protected readonly counts = computed<CountRow[]>(() => {
    const s = this.figures();
    if (!s) {
      return [];
    }
    return [
      { label: 'Từ vựng', icon: 'notebook', tone: 'accent', value: `${s.cards} từ` },
      { label: 'Ngữ pháp', icon: 'grammar', tone: 'read', value: s.grammar ? `${s.grammar.lessons} bài` : '—' },
      { label: 'Luyện nghe', icon: 'headphones', tone: 'listen', value: `${s.dictation.lessons ?? s.lessons.listen} bài` },
      // The Speaking practice keeps its results on the page: nothing to count yet.
      { label: 'Luyện nói', icon: 'mic', tone: 'listen', value: '—' },
      { label: 'Luyện đọc', icon: 'book', tone: 'read', value: `${s.lessons.read} bài` },
      { label: 'Luyện viết', icon: 'pencil', tone: 'write', value: `${s.writing.submitted} bài` },
    ];
  });

  protected readonly skills = computed<SkillBar[]>(() => {
    const s = this.figures();
    if (!s) {
      return [];
    }
    const score = s.writing.averageScore;
    return [
      {
        label: 'Đọc',
        tone: 'read',
        percent: s.reading.rate === null ? null : s.reading.rate * 100,
        value: percentText(s.reading.rate),
        hint: 'tỷ lệ trả lời đúng câu hỏi hiểu bài',
      },
      {
        label: 'Nghe',
        tone: 'listen',
        percent: s.dictation.rate === null ? null : s.dictation.rate * 100,
        value: percentText(s.dictation.rate),
        hint: 'tỷ lệ chép chính tả đúng',
      },
      {
        label: 'Viết',
        tone: 'write',
        percent: score === null ? null : (score / MAX_SCORE) * 100,
        value: score === null ? '—' : `${formatScore(score)}/${MAX_SCORE}`,
        hint: 'điểm trung bình bài viết',
      },
    ];
  });

  protected readonly badges = computed(() => {
    const s = this.allTime();
    return s ? badges(s, this.dashboard()?.streak ?? 0) : [];
  });
  protected readonly earnedCount = computed(() => this.badges().filter((b) => b.earned).length);

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    forkJoin({ all: this.api.stats(), dashboard: this.api.dashboard() })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: ({ all, dashboard }) => {
          this.allTime.set(all);
          this.dashboard.set(dashboard);
          this.loadPeriod();
        },
        error: () => this.loadError.set(true),
      });
  }

  protected choose(period: StatsPeriod): void {
    if (period !== this.period()) {
      this.period.set(period);
      this.loadPeriod();
    }
  }

  /** Tuần, Tháng, Tổng as tabs: arrow keys move between them. */
  protected onPeriodKeydown(event: KeyboardEvent): void {
    const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (!step) {
      return;
    }
    event.preventDefault();
    const i = PERIODS.findIndex((p) => p.id === this.period());
    const next = PERIODS[(i + step + PERIODS.length) % PERIODS.length];
    this.choose(next.id);
    (event.currentTarget as HTMLElement).parentElement
      ?.querySelector<HTMLElement>(`[data-period="${next.id}"]`)
      ?.focus();
  }

  /** Every-day figures are loaded already; a week or a month is asked for. */
  private loadPeriod(): void {
    this.periodLoad?.unsubscribe();
    this.periodError.set(false);
    this.periodFigures.set(null);
    const period = this.period();
    if (period === 'all') {
      return;
    }
    this.periodLoad = this.api
      .stats(period)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (s) => this.periodFigures.set(s),
        error: () => this.periodError.set(true),
      });
  }

  protected retryPeriod(): void {
    this.loadPeriod();
  }
}
