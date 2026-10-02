import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';

import { accuracyOf, Stats as StatsData } from '../../core/models/dashboard';
import { formatScore } from '../../core/models/writing';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { Icon } from '../../shared/components/icon/icon';
import { ProgressRing } from '../../shared/components/progress-ring/progress-ring';
import { Loading } from '../../shared/components/loading/loading';

/** Writing scores go from 1 to 5 (F8). */
const MAX_SCORE = 5;

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

/** "75%" for a 0–1 rate, "—" when nothing was measured. */
function percentText(rate: number | null | undefined): string {
  return rate === null || rate === undefined ? '—' : `${Math.round(rate * 100)}%`;
}

/** The stats page (F6, client sketch screen 10): accuracy, skills, words, dictation and lessons over every topic. */
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

  protected readonly data = signal<StatsData | null>(null);
  protected readonly loadError = signal(false);

  /** The dictation correct rate as a whole percentage, "—" when nothing was checked. */
  protected readonly rate = computed(() => percentText(this.data()?.dictation.rate));

  /** Mean writing score (F8), "3,8", or "—" until a writing is graded. */
  protected readonly writingAverage = computed(() => {
    const a = this.data()?.writing.averageScore;
    return a === null || a === undefined ? '—' : formatScore(a);
  });

  /** Comprehension answers right, as a whole percentage (F15). */
  protected readonly readingRate = computed(() => percentText(this.data()?.reading.rate));

  /** For the ring: 0–100, or null before any answer. */
  protected readonly accuracy = computed(() => {
    const s = this.data();
    const a = s ? accuracyOf(s) : null;
    return a === null ? null : a * 100;
  });

  protected readonly skills = computed<SkillBar[]>(() => {
    const s = this.data();
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

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api
      .stats()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (s) => this.data.set(s),
        error: () => this.loadError.set(true),
      });
  }
}
