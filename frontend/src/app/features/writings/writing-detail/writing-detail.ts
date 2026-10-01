import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { CRITERIA_LABELS, formatScore, Writing } from '../../../core/models/writing';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { wordDiff } from '../../../shared/utils/word-diff';

/**
 * One writing (F8): the prompt and text, and once graded the four criteria, the comments, the
 * corrected text and a word diff. A failed grading can be run again.
 */
@Component({
  selector: 'lu-writing-detail',
  imports: [RouterLink],
  templateUrl: './writing-detail.html',
  styleUrl: './writing-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WritingDetail {
  private readonly api = inject(WritingApiService);
  private readonly notifier = inject(WritingNotifier);
  private readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';

  protected readonly labels = CRITERIA_LABELS;
  protected readonly formatScore = formatScore;

  protected readonly writing = signal<Writing | null>(null);
  protected readonly notFound = signal(false);
  protected readonly loadError = signal(false);
  protected readonly regrading = signal(false);
  protected readonly actionError = signal<string | null>(null);

  protected readonly diff = computed(() => {
    const w = this.writing();
    return w?.grade?.status === 'done' ? wordDiff(w.text, w.grade.correctedText) : [];
  });

  constructor() {
    void this.load();
    // While this writing is graded, reload as soon as the notifier sees its result.
    effect(() => {
      const latest = this.notifier.latest();
      const pending = untracked(() => this.writing()?.grade?.status === 'pending');
      if (latest?.id === this.id && pending) {
        untracked(() => void this.load());
      }
    });
  }

  protected async load(): Promise<void> {
    this.loadError.set(false);
    try {
      const w = await firstValueFrom(this.api.get(this.id));
      this.writing.set(w);
      const status = w.grade?.status;
      if (status === 'done' || status === 'failed') {
        void this.notifier.markSeen(w.id);
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        this.notFound.set(true);
      } else {
        this.loadError.set(true);
      }
    }
  }

  protected async regrade(): Promise<void> {
    if (this.regrading()) {
      return;
    }
    this.regrading.set(true);
    this.actionError.set(null);
    try {
      this.writing.set(await firstValueFrom(this.api.regrade(this.id)));
      this.notifier.submitted();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.actionError.set(body?.message ?? 'Chưa chấm lại được, vui lòng thử lại.');
    } finally {
      this.regrading.set(false);
    }
  }
}
