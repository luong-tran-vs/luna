import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';

import { Stats as StatsData } from '../../core/models/dashboard';
import { formatScore } from '../../core/models/writing';
import { DashboardApiService } from '../../core/services/dashboard-api.service';

/** The stats page (F6): words, dictation and lessons over every topic. */
@Component({
  selector: 'lu-stats',
  imports: [RouterLink],
  templateUrl: './stats.html',
  styleUrl: './stats.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Stats {
  private readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly data = signal<StatsData | null>(null);
  protected readonly loadError = signal(false);

  /** The correct rate as a whole percentage, "—" when nothing was checked. */
  protected readonly rate = computed(() => {
    const r = this.data()?.dictation.rate;
    return r === null || r === undefined ? '—' : `${Math.round(r * 100)}%`;
  });

  /** Mean writing score (F8), "3,8", or "—" until a writing is graded. */
  protected readonly writingAverage = computed(() => {
    const a = this.data()?.writing.averageScore;
    return a === null || a === undefined ? '—' : formatScore(a);
  });

  /** Comprehension answers right, as a whole percentage (F15). */
  protected readonly readingRate = computed(() => {
    const r = this.data()?.reading.rate;
    return r === null || r === undefined ? '—' : `${Math.round(r * 100)}%`;
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
