import { effect, inject, Injectable, signal, untracked } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { GradeStatus, UnseenCount } from '../models/writing';
import { AuthService } from './auth.service';
import { WritingApiService } from './writing-api.service';

/** How often to ask for results while a writing is being graded (SC-004: within a minute). */
export const POLL_INTERVAL = 30_000;

export interface WritingToast {
  writingId: string;
  status: GradeStatus;
  message: string;
}

/**
 * In-app notice of new writing results (F8): the unseen count for the header badge and a toast
 * when a grading finishes. It polls only while a writing is being graded.
 */
@Injectable({ providedIn: 'root' })
export class WritingNotifier {
  private readonly api = inject(WritingApiService);
  private readonly auth = inject(AuthService);

  private readonly count = signal<UnseenCount | null>(null);
  readonly unseen = signal(0);
  readonly latest = signal<UnseenCount['latest']>(null);
  readonly toast = signal<WritingToast | null>(null);

  private timer?: ReturnType<typeof setTimeout>;

  constructor() {
    effect(() => {
      if (this.auth.isLoggedIn()) {
        untracked(() => void this.refresh());
      } else {
        untracked(() => this.reset());
      }
    });
  }

  /** Loads the counts; shows a toast when there are more new results than before. */
  async refresh(): Promise<void> {
    clearTimeout(this.timer);
    let next: UnseenCount;
    try {
      next = await firstValueFrom(this.api.unseenCount());
    } catch {
      return;
    }
    if (!this.auth.isLoggedIn()) {
      return;
    }
    const before = this.count();
    if (before && next.unseen > before.unseen && next.latest) {
      this.toast.set({
        writingId: next.latest.id,
        status: next.latest.status,
        message: next.latest.status === 'failed' ? 'Chấm bài viết bị lỗi' : 'Bài viết đã có kết quả',
      });
    }
    this.count.set(next);
    this.unseen.set(next.unseen);
    this.latest.set(next.latest);
    if (next.pending > 0) {
      this.timer = setTimeout(() => void this.refresh(), POLL_INTERVAL);
    }
  }

  /** A writing was just submitted or sent for regrading: watch for its result. */
  submitted(): void {
    void this.refresh();
  }

  /** The learner opened a result. */
  async markSeen(id: string): Promise<void> {
    try {
      await firstValueFrom(this.api.seen(id));
    } catch {
      return;
    }
    if (this.toast()?.writingId === id) {
      this.toast.set(null);
    }
    await this.refresh();
  }

  dismissToast(): void {
    this.toast.set(null);
  }

  private reset(): void {
    clearTimeout(this.timer);
    this.count.set(null);
    this.unseen.set(0);
    this.latest.set(null);
    this.toast.set(null);
  }
}
