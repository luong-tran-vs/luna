import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Step, Today as TodayData } from '../../../core/models/study';
import { DueCard } from '../../../core/models/vocab';
import { StudyApiService } from '../../../core/services/study-api.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { ReviewSession } from '../../../shared/components/review-session/review-session';
import { StepIndicator } from '../../../shared/components/step-indicator/step-indicator';
import { Listening } from '../listening/listening';
import { Reading } from '../reading/reading';
import { Writing } from '../writing/writing';

/** Delay before saving the reading or listening position (khối 2: debounce 1 s). */
const POSITION_DELAY = 1000;

/** Today's lesson (L): goal, streak, steps Ôn → Đọc → Nghe → Viết, and the current step itself. */
@Component({
  selector: 'lu-today',
  imports: [Listening, Reading, ReviewSession, RouterLink, StepIndicator, Writing],
  templateUrl: './today.html',
  styleUrl: './today.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Today {
  private readonly api = inject(StudyApiService);
  private readonly vocab = inject(VocabApiService);
  private readonly router = inject(Router);

  protected readonly data = signal<TodayData | null>(null);
  protected readonly cards = signal<DueCard[] | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly loadError = signal<string | null>(null);
  protected readonly completing = signal(false);

  private positionTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    this.load();
    inject(DestroyRef).onDestroy(() => clearTimeout(this.positionTimer));
  }

  private load(): void {
    this.api.today().subscribe({
      next: (d) => this.show(d),
      error: () => this.loadError.set('Không tải được bài hôm nay.'),
    });
  }

  private show(d: TodayData): void {
    this.data.set(d);
    this.cards.set(null);
    if (d.kind === 'studying' && d.currentStep === 'review') {
      this.vocab.due(d.reviewCount).subscribe({
        next: (list) => this.cards.set(list.cards),
        error: () => this.error.set('Không tải được thẻ cần ôn.'),
      });
    }
  }

  protected async complete(step: Step): Promise<void> {
    clearTimeout(this.positionTimer);
    this.completing.set(true);
    this.error.set(null);
    try {
      const d = await firstValueFrom(this.api.completeStep(step));
      if (d.kind === 'doneToday' && d.goalCompleted) {
        await this.router.navigateByUrl('/goal?completed=1');
        return;
      }
      this.show(d);
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Chưa lưu được bước này, vui lòng thử lại.');
    } finally {
      this.completing.set(false);
    }
  }

  /** Saves where the learner is, once they stop moving for a second. */
  protected savePosition(step: Step, sentenceIndex: number): void {
    clearTimeout(this.positionTimer);
    this.positionTimer = setTimeout(() => {
      this.api.savePosition(step, sentenceIndex).subscribe({ error: () => undefined });
    }, POSITION_DELAY);
  }
}
