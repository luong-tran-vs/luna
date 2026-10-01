import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';

import { Dashboard } from '../../core/models/dashboard';
import { Step } from '../../core/models/study';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { ProgressBar } from '../../shared/components/progress-bar/progress-bar';
import { StepIndicator } from '../../shared/components/step-indicator/step-indicator';

const STEP_LABELS: Record<Step, string> = { review: 'Ôn', read: 'Đọc', listen: 'Nghe', write: 'Viết' };

/**
 * The home page (F6): goal, skills, today's lesson and the button to its current step. Numbers
 * are loaded every time the page opens, so they reflect the step just finished.
 */
@Component({
  selector: 'lu-home',
  imports: [ProgressBar, RouterLink, StepIndicator],
  templateUrl: './home.html',
  styleUrl: './home.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Home {
  private readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly data = signal<Dashboard | null>(null);
  protected readonly loadError = signal(false);

  protected readonly actionLabel = computed(() => {
    const a = this.data()?.action;
    return a ? `${a.kind === 'start' ? 'Bắt đầu' : 'Tiếp tục'}: ${STEP_LABELS[a.step]}` : '';
  });

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api
      .dashboard()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (d) => this.data.set(d),
        error: () => this.loadError.set(true),
      });
  }
}
