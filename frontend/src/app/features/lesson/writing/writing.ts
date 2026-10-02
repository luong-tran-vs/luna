import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, input, OnInit, output, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { countWords } from '../../../core/models/generate';
import {
  formatScore,
  LessonWriting,
  MAX_WRITING_WORDS,
  MIN_WRITING_WORDS,
  SUGGESTED_WORDS,
  Writing as WritingData,
} from '../../../core/models/writing';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { Icon } from '../../../shared/components/icon/icon';
import { loadErrorMessage } from '../load-error';

/** Delay after the last keystroke before the draft is saved (khối 2: 1 s). */
const DRAFT_DELAY = 1000;

type SaveState = 'idle' | 'saving' | 'saved' | 'error';

/**
 * The Write step (F8): the lesson's prompt, a text box that saves a draft while typing, and
 * Nộp. Submitting finishes the step at once; the AI grades in the background. Writing is
 * optional: on the lesson page, Bỏ qua finishes the lesson without a writing. In review mode
 * (or once submitted) it only shows the writing and its result.
 */
@Component({
  selector: 'lu-writing',
  imports: [Icon, RouterLink],
  templateUrl: './writing.html',
  styleUrl: './writing.css',
  host: { '[class.standalone]': 'mode() === null' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Writing implements OnInit {
  private readonly api = inject(WritingApiService);
  private readonly notifier = inject(WritingNotifier);
  private readonly route = inject(ActivatedRoute);

  /** Defaults to the route's :id. */
  readonly lessonId = input('');
  /** "review" only shows the writing; defaults to ?review=1. */
  readonly mode = input<'study' | 'review' | null>(null);
  /** Emitted once the writing is submitted (the page then completes the step). */
  readonly completed = output<void>();
  /** Emitted when the learner skips writing (lesson page only; it then finishes the lesson). */
  readonly skipped = output<void>();

  protected readonly minWords = MIN_WRITING_WORDS;
  protected readonly maxWords = MAX_WRITING_WORDS;
  protected readonly formatScore = formatScore;

  protected readonly view = signal<LessonWriting | null>(null);
  protected readonly loadError = signal<string | null>(null);
  protected readonly text = signal('');
  protected readonly saveState = signal<SaveState>('idle');
  protected readonly submitting = signal(false);
  protected readonly submitError = signal<string | null>(null);
  /** Submitted in this visit (the page completes the step from the event). */
  protected readonly justSubmitted = signal(false);

  private timer?: ReturnType<typeof setTimeout>;

  protected readonly review = computed(
    () => (this.mode() ?? (this.route.snapshot?.queryParamMap?.get('review') === '1' ? 'review' : 'study')) === 'review',
  );
  protected readonly writing = computed(() => this.view()?.writing ?? null);
  protected readonly submitted = computed(() => this.writing()?.status === 'submitted');
  protected readonly editable = computed(() => !this.review() && !this.submitted() && !!this.view()?.canWrite);
  protected readonly words = computed(() => countWords(this.text()));
  protected readonly suggestion = computed(() => {
    const level = this.view()?.level;
    return level ? SUGGESTED_WORDS[level] : null;
  });
  protected readonly lengthHint = computed(() => {
    const n = this.words();
    if (n < MIN_WRITING_WORDS) {
      return `Bài viết cần ít nhất ${MIN_WRITING_WORDS} từ.`;
    }
    if (n > MAX_WRITING_WORDS) {
      return `Bài viết tối đa ${MAX_WRITING_WORDS} từ.`;
    }
    return null;
  });
  protected readonly canSubmit = computed(() => this.editable() && !this.lengthHint() && !this.submitting());
  /** Submitted before this visit while the step is still open (completing failed earlier). */
  protected readonly showContinue = computed(() => !this.review() && this.submitted() && !this.justSubmitted());

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
  }

  private get id(): string {
    return this.lessonId() || (this.route.snapshot?.paramMap?.get('id') ?? '');
  }

  ngOnInit(): void {
    void this.load();
  }

  private async load(): Promise<void> {
    try {
      const v = await firstValueFrom(this.api.lessonWriting(this.id));
      this.view.set(v);
      this.text.set(v.writing?.text ?? '');
    } catch (err) {
      this.loadError.set(loadErrorMessage(err));
    }
  }

  protected onInput(event: Event): void {
    this.text.set((event.target as HTMLTextAreaElement).value);
    this.submitError.set(null);
    clearTimeout(this.timer);
    this.timer = setTimeout(() => void this.saveDraft(), DRAFT_DELAY);
  }

  private async saveDraft(): Promise<void> {
    if (!this.editable()) {
      return;
    }
    this.saveState.set('saving');
    try {
      const w = await firstValueFrom(this.api.saveDraft(this.id, this.text()));
      this.setWriting(w);
      this.saveState.set('saved');
    } catch (err) {
      if (err instanceof ApiError && (err.body as { error?: string } | null)?.error === 'already_submitted') {
        await this.load();
      }
      this.saveState.set('error');
    }
  }

  protected async submit(): Promise<void> {
    if (!this.canSubmit()) {
      return;
    }
    clearTimeout(this.timer);
    this.submitting.set(true);
    this.submitError.set(null);
    try {
      const w = await firstValueFrom(this.api.submit(this.id, this.text()));
      this.setWriting(w);
      this.justSubmitted.set(true);
      this.notifier.submitted();
      this.completed.emit();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      this.submitError.set(body?.fields?.['text'] ?? body?.message ?? 'Chưa nộp được, vui lòng thử lại.');
    } finally {
      this.submitting.set(false);
    }
  }

  protected continue(): void {
    this.completed.emit();
  }

  protected skip(): void {
    this.skipped.emit();
  }

  private setWriting(w: WritingData): void {
    this.view.update((v) => (v ? { ...v, writing: w } : v));
  }
}
