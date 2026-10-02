import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { DueList, ReviewMode } from '../../../core/models/vocab';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { ReviewSession } from '../../../shared/components/review-session/review-session';

const SESSION_SIZE = 100;

const MODES: { value: ReviewMode; label: string }[] = [
  { value: 'flip', label: 'Xem từ đoán nghĩa' },
  { value: 'listen', label: 'Nghe rồi gõ' },
];

const dateTime = new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' });

/** Free review (F5, client sketch screen 9): the cards due now, in the mode the learner picks. */
@Component({
  selector: 'lu-review',
  imports: [Icon, ReviewSession, RouterLink],
  templateUrl: './review.html',
  styleUrl: './review.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Review {
  private readonly api = inject(VocabApiService);

  protected readonly modes = MODES;
  protected readonly due = signal<DueList | null>(null);
  protected readonly loadError = signal(false);
  protected readonly mode = signal<ReviewMode>('flip');
  protected readonly started = signal(false);
  protected readonly done = signal(false);
  protected readonly nextDue = computed(() => {
    const next = this.due()?.nextDue;
    return next ? dateTime.format(new Date(next)) : null;
  });

  constructor() {
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
