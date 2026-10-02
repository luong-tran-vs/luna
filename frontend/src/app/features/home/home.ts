import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';

import { accuracyOf, Dashboard, Stats } from '../../core/models/dashboard';
import { Step, STEPS } from '../../core/models/study';
import { AuthService } from '../../core/services/auth.service';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Icon } from '../../shared/components/icon/icon';
import { ProgressBar } from '../../shared/components/progress-bar/progress-bar';
import { ProgressRing } from '../../shared/components/progress-ring/progress-ring';

const STEP_LABELS: Record<Step, string> = { review: 'Ôn', read: 'Đọc', listen: 'Nghe', write: 'Viết' };

/**
 * The home page (F6, client sketch screen 2): greeting, today's progress, three figures, the button
 * to the current step and the topic being studied. Numbers are loaded every time the page opens.
 */
@Component({
  selector: 'lu-home',
  imports: [Icon, ProgressBar, ProgressRing, RouterLink],
  templateUrl: './home.html',
  styleUrl: './home.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Home {
  private readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly auth = inject(AuthService);
  protected readonly notifier = inject(WritingNotifier);

  protected readonly data = signal<Dashboard | null>(null);
  /** Figures for the tiles; the page works without them ("—"). */
  protected readonly stats = signal<Stats | null>(null);
  protected readonly loadError = signal(false);

  /** The part of the email before "@", as there is no display name. */
  protected readonly name = computed(() => this.auth.currentUser()?.email.split('@')[0] ?? '');

  protected readonly doneSteps = computed(() => {
    const steps = this.data()?.steps;
    return steps ? STEPS.filter((s) => steps[s] === 'done').length : 0;
  });
  protected readonly stepPercent = computed(() => (this.doneSteps() / STEPS.length) * 100);

  protected readonly subtitle = computed(() => {
    const d = this.data();
    switch (d?.kind) {
      case 'noGoal':
        return 'Chọn một chủ đề để bắt đầu học nhé!';
      case 'doneToday':
        return 'Bạn đã xong bài hôm nay, giỏi lắm!';
      case 'noNewLesson':
        return 'Ôn lại từ đã học trong lúc chờ bài mới nhé!';
      default: {
        const left = STEPS.length - this.doneSteps();
        return `Hôm nay còn ${left} bước nữa thôi!`;
      }
    }
  });

  protected readonly actionLabel = computed(() => {
    const a = this.data()?.action;
    return a ? `${a.kind === 'start' ? 'Bắt đầu' : 'Tiếp tục'}: ${STEP_LABELS[a.step]}` : '';
  });

  /** Comprehension and dictation together, as a whole percentage. */
  protected readonly accuracy = computed(() => {
    const s = this.stats();
    const a = s ? accuracyOf(s) : null;
    return a === null ? '—' : `${Math.round(a * 100)}%`;
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
    this.api
      .stats()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (s) => this.stats.set(s), error: () => undefined });
  }
}
