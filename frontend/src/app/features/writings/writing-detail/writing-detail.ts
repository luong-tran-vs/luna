import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { countWords } from '../../../core/models/generate';
import { CRITERIA_LABELS, formatScore, MAX_WRITING_WORDS, MIN_WRITING_WORDS, Writing } from '../../../core/models/writing';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { wordDiff } from '../../../shared/utils/word-diff';
import { Loading } from '../../../shared/components/loading/loading';

/**
 * One writing (F8): the prompt and text, and once graded the four criteria, the comments, the
 * corrected text and a word diff. A writing is graded at most twice: the learner may edit and
 * resubmit it once, or run a failed grading again.
 */
@Component({
  selector: 'lu-writing-detail',
  imports: [Loading, RouterLink],
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

  /** Gradings the writing may still use (resubmit or regrade); 0 for older answers without the count. */
  protected readonly gradingsLeft = computed(() => {
    const g = this.writing()?.gradings;
    return g ? Math.max(0, g.max - g.used) : 0;
  });

  /** The edited text while resubmitting; null when the editor is closed. */
  protected readonly editing = signal<string | null>(null);
  protected readonly editWords = computed(() => countWords(this.editing() ?? ''));
  protected readonly minWords = MIN_WRITING_WORDS;
  protected readonly maxWords = MAX_WRITING_WORDS;
  protected readonly resubmitting = signal(false);

  protected startEditing(): void {
    this.actionError.set(null);
    this.editing.set(this.writing()?.text ?? '');
  }

  protected async resubmit(): Promise<void> {
    const text = this.editing();
    if (text === null || this.resubmitting()) {
      return;
    }
    const n = countWords(text);
    if (n < MIN_WRITING_WORDS || n > MAX_WRITING_WORDS) {
      this.actionError.set(`Bài viết cần từ ${MIN_WRITING_WORDS} đến ${MAX_WRITING_WORDS} từ.`);
      return;
    }
    this.resubmitting.set(true);
    this.actionError.set(null);
    try {
      this.writing.set(await firstValueFrom(this.api.resubmit(this.id, text)));
      this.editing.set(null);
      this.notifier.submitted();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      this.actionError.set(body?.fields?.['text'] ?? body?.message ?? 'Chưa nộp lại được, vui lòng thử lại.');
    } finally {
      this.resubmitting.set(false);
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
