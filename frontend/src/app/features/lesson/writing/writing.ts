import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  OnInit,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { countWords } from '../../../core/models/generate';
import { WritingHelp } from '../../../core/models/practice';
import {
  CRITERIA_LABELS,
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
import { wordDiff } from '../../../shared/utils/word-diff';
import { Corrections } from '../../writings/corrections/corrections';
import { FrequentMistakes } from '../../writings/frequent-mistakes/frequent-mistakes';
import { containsAny, containsWord, pickWords, segments } from './checklist';
import { loadErrorMessage } from '../load-error';
import { Loading } from '../../../shared/components/loading/loading';

/** Delay after the last keystroke before the draft is saved (khối 2: 1 s). */
const DRAFT_DELAY = 1000;

type SaveState = 'idle' | 'saving' | 'saved' | 'error';

/**
 * The Write step (F8): the lesson's prompt, a text box that saves a draft while typing, and
 * Nộp. Submitting finishes the step at once; the AI grades in the background. Writing is
 * optional: on the lesson page, Bỏ qua finishes the lesson without a writing. In review mode
 * (or once submitted) it only shows the writing and its result: once graded, the scores and the
 * corrections marked right in the text, so the learner sees them where the mistakes are.
 */
@Component({
  selector: 'lu-writing',
  imports: [Loading, Icon, RouterLink, Corrections, FrequentMistakes],
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
  /** The grammar requirement, outline and model answer (six-step lesson, 2026-10-22); null for none. */
  readonly help = input<WritingHelp | null>(null);
  /** The lesson's words (texts and lemmas), checked off and underlined as they are used. */
  readonly targetWords = input<string[]>([]);
  /** Emitted once the writing is submitted (the page then completes the step). */
  readonly completed = output<void>();
  /** Emitted when the learner skips writing (lesson page only; it then finishes the lesson). */
  readonly skipped = output<void>();

  protected readonly minWords = MIN_WRITING_WORDS;
  protected readonly maxWords = MAX_WRITING_WORDS;
  protected readonly formatScore = formatScore;
  protected readonly labels = CRITERIA_LABELS;

  protected readonly view = signal<LessonWriting | null>(null);
  /** Back to the path of the lesson's part, where the learner came from. */
  protected readonly backLink = computed(() => {
    const level = this.view()?.level;
    return level ? ['/lessons/levels', level] : ['/lessons'];
  });
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
  /** The graded text against its corrected version, word by word; empty until graded. */
  protected readonly diff = computed(() => {
    const w = this.writing();
    return w?.grade?.status === 'done' ? wordDiff(w.text, w.grade.correctedText) : [];
  });
  /** Places to fix: a removal and the addition right after it count as one. */
  protected readonly fixCount = computed(() =>
    this.diff().filter((p, i, all) => p.kind === 'del' || (p.kind === 'add' && all[i - 1]?.kind !== 'del')).length,
  );

  // The "Cần có" checklist while writing (design 6a). It only guides: submitting needs just the word minimum.
  /** Up to 3 lesson words to use, each ticked once the text has it (or an inflection of it). */
  protected readonly wordChecks = computed(() => {
    const text = this.text();
    return pickWords(this.targetWords()).map((word) => ({ word, used: containsWord(text, word) }));
  });
  protected readonly wordsDone = computed(() => this.wordChecks().every((c) => c.used));
  /** The grammar words; Go may send null for an empty list. */
  private readonly anyOf = computed(() => this.help()?.anyOf ?? []);
  protected readonly grammarLabel = computed(() => (this.anyOf().length ? (this.help()?.grammarLabelVi ?? '') : ''));
  protected readonly grammarDone = computed(() => containsAny(this.text(), this.anyOf()));
  protected readonly rangeDone = computed(() => {
    const s = this.suggestion();
    const n = this.words();
    return !!s && n >= s.min && n <= s.max;
  });
  /** The text behind the (transparent) text box, with the lesson words underlined. */
  protected readonly underlined = computed(() => segments(this.text(), this.targetWords()));
  protected readonly outline = computed(() => (this.help()?.outlineVi ?? []).filter((s) => s.trim()));
  protected readonly outlineOpen = signal(false);

  // After submitting (design 6c): the model answer and the learner's frequent mistakes.
  protected readonly modelAnswer = computed(() => (this.help()?.modelAnswer ?? '').trim());
  protected readonly modelParts = computed(() =>
    segments(this.modelAnswer(), this.targetWords(), this.anyOf()),
  );
  protected readonly modelWords = computed(() => countWords(this.modelAnswer()));
  protected readonly modelOpen = signal(false);
  private readonly backdrop = viewChild<ElementRef<HTMLElement>>('backdrop');
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
  /** Advice against the suggested length; unlike lengthHint it does not stop the submission. */
  protected readonly suggestionHint = computed(() => {
    const s = this.suggestion();
    const n = this.words();
    if (!s || this.lengthHint() || n === 0) {
      return null;
    }
    if (n < s.min) {
      return `Còn ngắn so với gợi ý ${s.min}–${s.max} từ, vẫn nộp được.`;
    }
    if (n > s.max) {
      return `Dài hơn gợi ý ${s.min}–${s.max} từ, vẫn nộp được.`;
    }
    return null;
  });
  protected readonly canSubmit = computed(() => this.editable() && !this.lengthHint() && !this.submitting());
  /** Submitted before this visit while the step is still open (completing failed earlier). */
  protected readonly showContinue = computed(() => !this.review() && this.submitted() && !this.justSubmitted());

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
    // While the writing is graded, reload as soon as the notifier sees its result.
    effect(() => {
      const latest = this.notifier.latest();
      const w = untracked(() => this.writing());
      if (latest && w && latest.id === w.id && latest.status !== 'pending' && w.grade?.status === 'pending') {
        untracked(() => void this.load());
      }
    });
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

  /** Keeps the underlines behind the text box in line with its scrolled text. */
  protected syncScroll(event: Event): void {
    const backdrop = this.backdrop()?.nativeElement;
    if (backdrop) {
      backdrop.scrollTop = (event.target as HTMLTextAreaElement).scrollTop;
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
