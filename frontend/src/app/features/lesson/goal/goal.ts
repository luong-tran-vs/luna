import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Dashboard } from '../../../core/models/dashboard';
import { Level, LEVELS } from '../../../core/models/lesson';
import { Goals, GoalView, PublicTopic } from '../../../core/models/study';
import { DashboardApiService } from '../../../core/services/dashboard-api.service';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';

const LEVEL_NAMES: Record<Level, string> = {
  A1: 'Mới bắt đầu',
  A2: 'Sơ cấp',
  B1: 'Trung cấp',
  B2: 'Trung cấp cao',
  C1: 'Nâng cao',
  C2: 'Thành thạo',
};

/** Icon colors cycled over the topic rows. */
const TONES = ['tone-read', 'tone-listen', 'tone-accent', 'tone-write'];

/** Choosing a goal (L, client sketch screen 3): a level, then a topic of that level; congratulations at the end. */
@Component({
  selector: 'lu-goal',
  imports: [Icon, RouterLink],
  templateUrl: './goal.html',
  styleUrl: './goal.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Goal {
  private readonly api = inject(StudyApiService);
  private readonly router = inject(Router);

  protected readonly levels = LEVELS;
  protected readonly levelNames = LEVEL_NAMES;
  protected readonly tones = TONES;
  protected readonly goals = signal<Goals | null>(null);
  protected readonly level = signal<Level | null>(null);
  protected readonly topics = signal<PublicTopic[] | null>(null);
  protected readonly congrats = signal(inject(ActivatedRoute).snapshot.queryParamMap.get('completed') === '1');
  protected readonly pending = signal(false);
  protected readonly startsTomorrow = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly active = computed(() => this.goals()?.active ?? null);
  protected readonly nextLevel = computed(() => {
    const current = this.active()?.level;
    const i = current ? LEVELS.indexOf(current) : -1;
    return i >= 0 && i + 1 < LEVELS.length ? LEVELS[i + 1] : null;
  });

  /** Today's state of the current course, to offer the right action (null while loading or on error). */
  protected readonly today = signal<Dashboard | null>(null);
  protected readonly todayLabel = computed(() => (this.today()?.action?.kind === 'start' ? 'Bắt đầu học' : 'Tiếp tục học'));

  constructor() {
    inject(DashboardApiService)
      .dashboard()
      // Without it the card falls back to a plain link to today's page.
      .subscribe({ next: (d) => this.today.set(d), error: () => undefined });
    this.api.goals().subscribe({
      next: (g) => {
        this.goals.set(g);
        if (!this.congrats() && g.active) {
          this.selectLevel(g.active.level);
        }
      },
      error: () => this.error.set('Không tải được mục tiêu.'),
    });
  }

  /** Progress on a topic already studied (active or paused goal). */
  protected progressOf(topicId: string): GoalView | undefined {
    const g = this.goals();
    return [g?.active, ...(g?.others ?? [])].find((x) => x?.topicId === topicId) ?? undefined;
  }

  protected selectLevel(level: Level): void {
    this.congrats.set(false);
    this.level.set(level);
    this.topics.set(null);
    this.api.topics(level).subscribe({
      next: (list) => this.topics.set(list),
      error: () => this.error.set('Không tải được danh sách chủ đề.'),
    });
  }

  protected onLevelKeydown(event: KeyboardEvent, index: number): void {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const next = (index + step + LEVELS.length) % LEVELS.length;
    this.selectLevel(LEVELS[next]);
    const group = (event.currentTarget as HTMLElement).closest('[role="radiogroup"]');
    group?.querySelectorAll<HTMLElement>('[role="radio"]')[next]?.focus();
  }

  protected async choose(topic: PublicTopic): Promise<void> {
    this.pending.set(true);
    this.error.set(null);
    try {
      const res = await firstValueFrom(this.api.setGoal(topic.id));
      if (res.startsTomorrow) {
        this.startsTomorrow.set(true);
      } else {
        await this.router.navigateByUrl('/today');
      }
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Không chọn được chủ đề, vui lòng thử lại.');
    } finally {
      this.pending.set(false);
    }
  }
}
