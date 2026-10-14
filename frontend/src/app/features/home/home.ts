import { DOCUMENT } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';

import { accuracyOf, Dashboard, Stats } from '../../core/models/dashboard';
import { Step, STEPS } from '../../core/models/study';
import { AuthService } from '../../core/services/auth.service';
import { DashboardApiService } from '../../core/services/dashboard-api.service';
import { VocabApiService } from '../../core/services/vocab-api.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Icon } from '../../shared/components/icon/icon';
import { ProgressBar } from '../../shared/components/progress-bar/progress-bar';

const STEP_LABELS: Record<Step, string> = { read: 'Đọc', listen: 'Nghe', write: 'Viết' };

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
  protected readonly notifier = inject(WritingNotifier);

  protected readonly data = signal<Dashboard | null>(null);
  /** Figures for the tiles; the page works without them ("—"). */
  protected readonly stats = signal<Stats | null>(null);
  protected readonly loadError = signal(false);
  /** Cards due now; the page works without it (0). */
  protected readonly dueToday = signal(0);
  protected readonly dueMinutes = computed(() => Math.max(1, Math.ceil((this.dueToday() * 20) / 60)));

  /** The part of the email before "@", as there is no display name. */
  protected readonly stepCount = STEPS.length;

  protected readonly name = computed(() => this.auth.currentUser()?.email.split('@')[0] ?? '');

  protected readonly doneSteps = computed(() => {
    const steps = this.data()?.steps;
    return steps ? STEPS.filter((s) => steps[s] === 'done').length : 0;
  });

  protected readonly subtitle = computed(() => {
    const d = this.data();
    switch (d?.kind) {
      case 'noGoal':
        return 'Chọn một chủ đề để bắt đầu học nhé!';
      case 'noNewLesson':
        return 'Ôn lại từ đã học trong lúc chờ bài mới nhé!';
      default: {
        const steps = d?.steps;
        const left = STEPS.filter((s) => steps?.[s] !== 'done');
        if (left.length === STEPS.length) {
          return 'Sẵn sàng cho bài mới chưa? Bắt đầu thôi!';
        }
        const names = left.map((s) => STEP_LABELS[s]).join(', ');
        return `Bài này còn ${left.length} phần nữa thôi (${names})!`;
      }
    }
  });

  protected readonly actionLabel = computed(() => {
    const a = this.data()?.action;
    if (!a) {
      return '';
    }
    // The lesson opens on its practice first, so a new lesson just starts.
    return a.kind === 'start' ? 'Bắt đầu học' : `Tiếp tục: ${STEP_LABELS[a.step]}`;
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
        next: (d) => this.data.set(d),
        error: () => this.loadError.set(true),
      });
    this.api
      .stats()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (s) => this.stats.set(s), error: () => undefined });
    this.vocab
      .due(1)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (list) => this.dueToday.set(list.total), error: () => undefined });
  }
}
