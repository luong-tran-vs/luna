import { CdkConnectedOverlay, ConnectedPosition } from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  afterNextRender,
  inject,
  Injector,
  input,
  OnInit,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, firstValueFrom, forkJoin, of, Subscription } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AskQuota, ReadingLesson, Token } from '../../../core/models/reading';
import { SpeechService } from '../../../core/services/speech.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { CardInput } from '../../../core/models/vocab';
import { Icon } from '../../../shared/components/icon/icon';
import { PopupState, WordPopup } from '../../../shared/components/word-popup/word-popup';
import { tokenize } from '../../../shared/utils/tokenize';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { ComprehensionQuiz } from './comprehension-quiz/comprehension-quiz';
import { SentenceTranslation } from './sentence-translation/sentence-translation';
import { Loading } from '../../../shared/components/loading/loading';

interface Selected {
  sentence: number;
  first: number;
  last: number;
  text: string;
}

/** Popup below the word, or above it when there is no room; never covering the word. */
const POSITIONS: ConnectedPosition[] = [
  { originX: 'center', originY: 'bottom', overlayX: 'center', overlayY: 'top', offsetY: 8 },
  { originX: 'center', originY: 'top', overlayX: 'center', overlayY: 'bottom', offsetY: -8 },
];

const normalize = (s: string) => s.trim().replace(/\s+/g, ' ').toLowerCase();

/** How long a sentence is held to show its translation, and how far the pointer may move meanwhile. */
const LONG_PRESS_MS = 500;
const PRESS_SLOP = 10;

/** Message for a failed "Hỏi AI": the server message (or the field message for a bad selection). */
function askErrorMessage(err: unknown): string {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = (err.body ?? {}) as { message?: unknown; fields?: Record<string, unknown> };
    const field = body.fields?.['text'] ?? body.fields?.['sentenceIndex'];
    if (typeof field === 'string') {
      return field;
    }
    if (typeof body.message === 'string' && err.status !== 400) {
      return body.message;
    }
  }
  return 'Không hỏi được AI, vui lòng thử lại.';
}

/** The tips line ("Chạm từ … giữ câu …") stays hidden once the learner closes it. */
const TIPS_KEY = 'luna.reading-tips-hidden';

/** Sentences per paragraph on screen: a long paragraph is cut in pieces of two or three. */
const CHUNK = 3;

/** A paragraph's sentences in balanced pieces of at most CHUNK (5 → 3 + 2, 8 → 3 + 3 + 2). */
export function chunkParagraph(sentences: readonly number[]): number[][] {
  if (sentences.length <= CHUNK) {
    return [[...sentences]];
  }
  const pieces = Math.ceil(sentences.length / CHUNK);
  const size = Math.ceil(sentences.length / pieces);
  const out: number[][] = [];
  for (let i = 0; i < sentences.length; i += size) {
    out.push(sentences.slice(i, i + size));
  }
  return out;
}

/**
 * The Reading step (F3), in two screens (design, 2026-10-15): first the text alone, cut in short
 * paragraphs (tap a word to look it up and save it, hold a sentence for its translation); then
 * the comprehension questions one at a time, the text one tap away ("Xem lại bài đọc"). Each
 * screen has its bar at the bottom: ← and "Đọc xong, trả lời N câu hỏi", then Kiểm tra. Without
 * questions, the step finishes when the end of the text has been seen.
 */
@Component({
  selector: 'lu-reading',
  imports: [
    Loading,
    CdkConnectedOverlay,
    ComprehensionQuiz,
    Icon,
    RouterLink,
    SentenceTranslation,
    WordPopup,
  ],
  templateUrl: './reading.html',
  styleUrl: './reading.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[class.standalone]': 'mode() === null',
    // Word interactions are delegated from the host; each word is a focusable role="button".
    // A long press on a sentence shows its translation.
    '(pointerdown)': 'onPressStart($event)',
    '(pointermove)': 'onPressMove($event)',
    '(pointerup)': 'onPressEnd()',
    '(pointercancel)': 'onPressEnd()',
    '(contextmenu)': 'onContextMenu($event)',
    // The reading text cannot be copied.
    '(copy)': 'onCopy($event)',
    '(keydown)': 'onArticleKeydown($event)',
    '(focusin)': 'onFocusIn($event)',
  },
})
export class Reading implements OnInit {
  private readonly api = inject(ReadingApiService);
  private readonly vocab = inject(VocabApiService);
  private readonly speech = inject(SpeechService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly route = inject(ActivatedRoute).snapshot;
  private readonly injector = inject(Injector);

  /** The lesson to open; defaults to the route's :id (used by the lesson page, L). */
  readonly lessonId = input('');
  /** Sentence to scroll to first (saved position). */
  readonly startSentence = input<number | null>(null);
  /** F9: the learner's "Hỏi AI" quota in this lesson; null until loaded or when unlimited. */
  private readonly askQuota = signal<AskQuota | null>(null);
  /** AI asks left for the selected text: null when unlimited or already asked (asking again is free). */
  protected readonly askLeft = computed(() => {
    const q = this.askQuota();
    const sel = this.selected();
    if (!q || q.limit === 0 || (sel && q.asked.includes(normalize(sel.text)))) {
      return null;
    }
    return Math.max(0, q.limit - q.asked.length);
  });
  protected readonly askLimit = computed(() => this.askQuota()?.limit ?? 0);
  /** "review" rereads a finished lesson (no "Đã đọc xong"); defaults to ?review=1. */
  readonly mode = input<'study' | 'review' | null>(null);
  /** Position of the step in the lesson, for "2. Đọc"; 0 for none. */
  readonly number = input(0);
  /** Whether the bar shows ← (the lesson goes to the step before). */
  readonly canBack = input(false);
  /** A quiet "Để sau, sang bước Nghe" under the text while the step is to do; empty for none. */
  readonly laterLabel = input('');
  /** The last button when reading again, e.g. "Sang bước Nghe"; empty for none. */
  readonly nextLabel = input('');
  /** ← in the bar on the text. */
  readonly back = output<void>();
  /** The lesson goes on without this step (laterLabel) or after reading it again (nextLabel). */
  readonly leave = output<void>();

  /** Emitted once when the learner finishes the step (feature L records progress). */
  readonly completed = output<void>();
  /** The first sentence on screen, when it changes (feature L saves it). */
  readonly position = output<number>();

  protected get id(): string {
    return this.lessonId() || (this.route.paramMap.get('id') ?? '');
  }

  protected readonly review = computed(
    () => (this.mode() ?? (this.route.queryParamMap?.get('review') === '1' ? 'review' : 'study')) === 'review',
  );

  protected readonly lesson = signal<ReadingLesson | null>(null);
  /** Back to the path of the lesson's part, where the learner came from. */
  protected readonly backLink = computed(() => {
    const level = this.lesson()?.level;
    return level ? ['/lessons/levels', level] : ['/lessons'];
  });
  protected readonly loadError = signal<string | null>(null);
  protected readonly saved = signal<ReadonlySet<string>>(new Set());

  protected readonly tokens = computed<Token[][]>(() => this.lesson()?.sentences.map((s) => tokenize(s.text)) ?? []);
  /** Keys "sentence:token" of words to highlight as saved. */
  protected readonly highlighted = computed(() => this.computeHighlights());
  protected readonly focusKey = signal('0:0');

  protected readonly positions = POSITIONS;
  protected readonly anchor = signal<HTMLElement | null>(null);
  protected readonly selected = signal<Selected | null>(null);
  protected readonly popupState = signal<PopupState>({ kind: 'loading' });
  protected readonly popupError = signal<string | null>(null);
  protected readonly saving = signal(false);
  protected readonly asking = signal(false);
  protected readonly askError = signal<string | null>(null);
  protected readonly popupSaved = computed(() => {
    const state = this.popupState();
    const sel = this.selected();
    const lemma = state.kind === 'result' ? state.result.lemma : sel ? normalize(sel.text) : '';
    return !!lemma && this.saved().has(normalize(lemma));
  });

  /** The sentence being pressed, highlighted while the press lasts. */
  protected readonly pressing = signal<number | null>(null);
  /** The sentence whose translation is shown, and the element it is shown next to. */
  protected readonly translated = signal<{ sentence: number; anchor: HTMLElement } | null>(null);

  protected readonly reachedEnd = signal(false);
  /** The text first, then the questions. */
  protected readonly phase = signal<'read' | 'quiz'>('read');
  /** "Xem lại bài đọc": the text shown again over the questions. */
  protected readonly peek = signal(false);
  protected readonly tipsHidden = signal(readTipsHidden());
  /** The lesson's paragraphs on screen, long ones cut in pieces of two or three sentences. */
  protected readonly chunks = computed(() =>
    (this.lesson()?.paragraphs ?? []).flatMap((p) => chunkParagraph(p)),
  );
  protected readonly done = signal(false);

  private readonly article = viewChild<ElementRef<HTMLElement>>('article');
  /** The questions, for "Làm lại tất cả" in the row with "Xem lại bài đọc". */
  protected readonly quizRef = viewChild(ComprehensionQuiz);
  /** Room at the right of the step's title for the questions' dots (28px each, 2px apart). */
  protected readonly dotsRoom = computed(() => {
    const count = this.phase() === 'quiz' ? (this.lesson()?.quiz?.questions.length ?? 0) : 0;
    return count ? count * 30 + 8 : null;
  });
  /** Questions answered, as the questions report it: "Làm lại tất cả" shows once there is one. */
  protected readonly answered = signal(0);
  private readonly end = viewChild<ElementRef<HTMLElement>>('end');
  private lookupSub?: Subscription;
  private askSub?: Subscription;
  private pressTimer?: ReturnType<typeof setTimeout>;
  private pressFrom = { x: 0, y: 0 };
  /** A long press just ended: the click (and outside click) it may still fire is not a tap. */
  private afterLongPress = false;

  ngOnInit(): void {
    // Inputs are set by now.
    this.load();
  }

  /** Loads (or reloads, after the questions changed) the lesson and the saved words. */
  protected load(): void {
    forkJoin({
      lesson: this.api.getLesson(this.id),
      words: this.vocab.words().pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ lesson, words }) => {
        this.saved.set(new Set(words.map((w) => normalize(w.lemma))));
        this.lesson.set(lesson);
        // Questions taken away meanwhile (reloaded after they changed): back to the text.
        if (!lesson.quiz) {
          this.phase.set('read');
        }
        this.askQuota.set(lesson.askQuota ?? null);
        const firstWord = this.tokens()[0]?.find((t) => t.isWord);
        this.focusKey.set(`0:${firstWord?.index ?? 0}`);
        this.scrollToStart();
      },
      error: (err: unknown) => this.loadError.set(loadErrorMessage(err)),
    });
  }

  /** Scrolls to the saved sentence once the lesson is rendered. */
  private scrollToStart(): void {
    const start = this.startSentence();
    if (start === null || start <= 0) {
      return;
    }
    afterNextRender(
      () => this.article()?.nativeElement.querySelector<HTMLElement>(`.sentence[data-s="${start}"]`)?.scrollIntoView({ block: 'start' }),
      { injector: this.injector },
    );
  }

  constructor() {
    // Enable "Đã đọc xong" once the end of the lesson has been on screen.
    effect((onCleanup) => {
      const end = this.end()?.nativeElement;
      if (!end || this.reachedEnd()) {
        return;
      }
      if (typeof IntersectionObserver === 'undefined') {
        this.reachedEnd.set(true);
        return;
      }
      const observer = new IntersectionObserver((entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          this.reachedEnd.set(true);
          observer.disconnect();
        }
      });
      observer.observe(end);
      onCleanup(() => observer.disconnect());
    });

    // Report the first sentence on screen (saved as the position on the lesson page, L).
    effect((onCleanup) => {
      const article = this.article()?.nativeElement;
      if (!article || typeof IntersectionObserver === 'undefined') {
        return;
      }
      const visible = new Set<number>();
      let reported = -1;
      const observer = new IntersectionObserver((entries) => {
        for (const e of entries) {
          const s = Number((e.target as HTMLElement).dataset['s']);
          if (e.isIntersecting) {
            visible.add(s);
          } else {
            visible.delete(s);
          }
        }
        const first = visible.size ? Math.min(...visible) : -1;
        if (first >= 0 && first !== reported) {
          reported = first;
          this.position.emit(first);
        }
      });
      article.querySelectorAll('.sentence').forEach((el) => observer.observe(el));
      onCleanup(() => observer.disconnect());
    });

    inject(DestroyRef).onDestroy(() => {
      this.lookupSub?.unsubscribe();
      this.askSub?.unsubscribe();
      clearTimeout(this.pressTimer);
    });
  }

  // --- rendering helpers ---

  protected key(s: number, t: number): string {
    return `${s}:${t}`;
  }

  private lemmaOf(word: string): string {
    const w = word.toLowerCase();
    return this.lesson()?.lemmas[w] ?? w;
  }

  private computeHighlights(): ReadonlySet<string> {
    const saved = this.saved();
    const lesson = this.lesson();
    const out = new Set<string>();
    if (!lesson || saved.size === 0) {
      return out;
    }
    // Phrases: annotated phrase texts whose lemma is saved, plus saved phrases verbatim.
    const phraseWords: string[][] = [
      ...lesson.phrases.filter((p) => saved.has(normalize(p.lemma))).map((p) => normalize(p.text).split(' ')),
      ...[...saved].filter((l) => l.includes(' ')).map((l) => l.split(' ')),
    ];

    this.tokens().forEach((tokens, s) => {
      const words = tokens.filter((t) => t.isWord);
      for (const w of words) {
        if (saved.has(this.lemmaOf(w.text))) {
          out.add(this.key(s, w.index));
        }
      }
      for (const phrase of phraseWords) {
        for (let i = 0; i + phrase.length <= words.length; i++) {
          if (phrase.every((p, j) => words[i + j].text.toLowerCase() === p)) {
            for (let j = 0; j < phrase.length; j++) {
              out.add(this.key(s, words[i + j].index));
            }
          }
        }
      }
    });
    return out;
  }

  // --- opening the popup ---

  protected onWordClick(event: MouseEvent, s: number, t: number): void {
    if (this.afterLongPress) {
      return; // the release of a long press, not a tap on the word
    }
    this.open(s, t, t, event.currentTarget as HTMLElement);
  }

  // --- long press on a sentence: its translation ---

  protected onPressStart(event: PointerEvent): void {
    this.cancelPress();
    if (event.button !== 0) {
      return;
    }
    const sentence = (event.target as HTMLElement).closest?.<HTMLElement>('.sentence');
    if (!sentence || !this.article()?.nativeElement.contains(sentence)) {
      return;
    }
    const s = Number(sentence.dataset['s']);
    this.pressFrom = { x: event.clientX, y: event.clientY };
    this.pressing.set(s);
    this.pressTimer = setTimeout(() => {
      this.pressing.set(null);
      this.afterLongPress = true;
      this.showTranslation(s, sentence);
    }, LONG_PRESS_MS);
  }

  /** Moving away (scrolling the page) is not a press. */
  protected onPressMove(event: PointerEvent): void {
    if (this.pressing() !== null && Math.hypot(event.clientX - this.pressFrom.x, event.clientY - this.pressFrom.y) > PRESS_SLOP) {
      this.cancelPress();
    }
  }

  protected onPressEnd(): void {
    this.cancelPress();
    if (this.afterLongPress) {
      // The click of this release comes right after it; later clicks are taps again.
      setTimeout(() => (this.afterLongPress = false));
    }
  }

  protected onCopy(event: ClipboardEvent): void {
    if (this.article()?.nativeElement.contains(event.target as Node)) {
      event.preventDefault();
    }
  }

  /** The browser's long-press menu (copy, share…) would cover the translation. */
  protected onContextMenu(event: Event): void {
    if (this.article()?.nativeElement.contains(event.target as Node)) {
      event.preventDefault();
    }
  }

  private cancelPress(): void {
    clearTimeout(this.pressTimer);
    this.pressing.set(null);
  }

  private showTranslation(sentence: number, anchor: HTMLElement): void {
    if (this.anchor()) {
      this.close();
    }
    this.translated.set({ sentence, anchor });
  }

  protected closeTranslation(): void {
    const t = this.translated();
    this.translated.set(null);
    // Back to the word keyboard users were on, in that sentence when they pressed "T".
    const word = t?.anchor.querySelector<HTMLElement>('.w[tabindex="0"]') ??
      this.article()?.nativeElement.querySelector<HTMLElement>('.w[tabindex="0"]');
    word?.focus();
  }

  protected onTranslationOutsideClick(): void {
    if (!this.afterLongPress) {
      this.closeTranslation();
    }
  }

  protected onTranslationKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      this.closeTranslation();
    }
  }

  protected onArticleKeydown(event: KeyboardEvent): void {
    const target = event.target as HTMLElement;
    if (!target.classList.contains('w') || !this.article()?.nativeElement.contains(target)) {
      return;
    }
    const s = Number(target.dataset['s']);
    const t = Number(target.dataset['t']);
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      this.open(s, t, t, target);
      return;
    }
    // The keyboard's long press: the translation of the word's sentence.
    if (event.key === 't' || event.key === 'T') {
      event.preventDefault();
      const sentence = target.closest<HTMLElement>('.sentence');
      if (sentence) {
        this.showTranslation(s, sentence);
      }
      return;
    }
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const words = Array.from(this.article()?.nativeElement.querySelectorAll<HTMLElement>('.w') ?? []);
    const next = words[words.indexOf(target) + step];
    if (next) {
      this.focusKey.set(this.key(Number(next.dataset['s']), Number(next.dataset['t'])));
      next.focus();
    }
  }

  protected onFocusIn(event: FocusEvent): void {
    const target = event.target as HTMLElement;
    if (target.classList.contains('w')) {
      this.focusKey.set(this.key(Number(target.dataset['s']), Number(target.dataset['t'])));
    }
  }

  private open(sentence: number, first: number, last: number, anchor: HTMLElement): void {
    const text = this.tokens()[sentence].slice(first, last + 1).map((t) => t.text).join('').trim().replace(/\s+/g, ' ');
    const current = this.selected();
    if (current && current.sentence === sentence && current.first === first && current.last === last && this.anchor()) {
      return; // the word already open
    }
    this.selected.set({ sentence, first, last, text });
    this.anchor.set(anchor);
    this.popupState.set({ kind: 'loading' });
    this.popupError.set(null);
    this.stopAsking();

    this.lookupSub?.unsubscribe();
    this.lookupSub = this.api.lookup(this.id, text, sentence).subscribe({
      next: (result) => this.popupState.set({ kind: 'result', result }),
      error: (err: unknown) => {
        this.popupState.set({ kind: 'not-found' });
        if (!(err instanceof ApiError && err.status === 404)) {
          this.popupError.set('Không tra được, vui lòng thử lại.');
        }
      },
    });
  }

  // --- popup actions ---

  protected close(): void {
    const anchor = this.anchor();
    this.lookupSub?.unsubscribe();
    this.stopAsking();
    this.anchor.set(null);
    this.selected.set(null);
    anchor?.focus();
  }

  /** Asks the AI about the selection (F9). A newer selection or closing the popup drops the answer. */
  protected ask(): void {
    const sel = this.selected();
    if (!sel || this.asking()) {
      return;
    }
    this.asking.set(true);
    this.askError.set(null);
    this.askSub = this.api.ask(this.id, sel.text, sel.sentence).subscribe({
      next: ({ result, quota }) => {
        this.asking.set(false);
        this.popupState.set({ kind: 'result', result });
        if (quota) {
          this.askQuota.set(quota);
        }
      },
      error: (err: unknown) => {
        this.asking.set(false);
        this.askError.set(askErrorMessage(err));
        const quota = err instanceof ApiError ? (err.body as { quota?: AskQuota } | null)?.quota : undefined;
        if (quota) {
          this.askQuota.set(quota);
        }
      },
    });
  }

  private stopAsking(): void {
    this.askSub?.unsubscribe();
    this.asking.set(false);
    this.askError.set(null);
  }

  protected onOutsideClick(event: MouseEvent): void {
    // A click on another word opens that word instead (handled by its click listener).
    if (!(event.target instanceof HTMLElement && event.target.classList.contains('w'))) {
      this.close();
    }
  }

  protected onOverlayKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      this.close();
    }
  }

  protected async save(manualMeaning?: string): Promise<void> {
    const sel = this.selected();
    const state = this.popupState();
    const lesson = this.lesson();
    if (!sel || !lesson) {
      return;
    }
    const result = state.kind === 'result' ? state.result : null;
    const input: CardInput = {
      text: sel.text,
      lemma: result?.lemma ?? normalize(sel.text),
      ipa: result?.ipa ?? '',
      meaningVi: manualMeaning ?? result?.meanings.slice(0, 3).map((m) => m.text).join('; ') ?? '',
      contextSentence: lesson.sentences[sel.sentence].text,
      lessonId: lesson.id,
      // With only the sentence's translation shown, the meaning saved is the one the learner typed.
      source: result && result.source !== 'sentence' ? result.source : 'manual',
    };

    this.saving.set(true);
    this.popupError.set(null);
    try {
      await firstValueFrom(this.vocab.saveCard(input));
      this.markSaved(input.lemma);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        this.markSaved(input.lemma);
      } else {
        this.popupError.set('Không lưu được, vui lòng thử lại.');
      }
    } finally {
      this.saving.set(false);
    }
  }

  private markSaved(lemma: string): void {
    this.saved.update((s) => new Set([...s, normalize(lemma)]));
  }

  /** ✕ on the tips line: hidden from now on, on this device. */
  protected hideTips(): void {
    this.tipsHidden.set(true);
    try {
      localStorage.setItem(TIPS_KEY, '1');
    } catch {
      // Storage can be blocked: the tips just come back next time.
    }
  }

  /** "Đọc xong, trả lời N câu hỏi": on to the questions. */
  protected toQuiz(): void {
    this.close();
    this.closeTranslation();
    this.peek.set(false);
    this.phase.set('quiz');
    this.toTop();
  }

  /** ← on the questions: back to the text. */
  protected toText(): void {
    this.phase.set('read');
    this.toTop();
  }

  /** The questions' last button: Tiếp tục (answered before) or on to the next step when reading again. */
  protected onQuizFinished(): void {
    if (this.review()) {
      this.leave.emit();
    } else {
      this.finishQuiz();
    }
  }

  private toTop(): void {
    this.host.nativeElement.scrollIntoView?.({ block: 'start' });
  }

  /** Reads the looked-up word with the browser's voice. */
  protected play(): void {
    const state = this.popupState();
    const text = state.kind === 'result' ? state.result.lemma : this.selected()?.text;
    if (!text) {
      return;
    }
    this.popupError.set(null);
    this.speech.speak(text, 1, { failed: () => this.popupError.set('Chưa đọc được từ này.') });
  }

  // --- finishing ---

  protected finish(): void {
    if (!this.reachedEnd() || this.done()) {
      return;
    }
    this.done.set(true);
    this.completed.emit();
  }

  /** Every comprehension question answered before this visit (e.g. completing failed offline). */
  protected readonly quizAnsweredOnLoad = computed(() => {
    const quiz = this.lesson()?.quiz;
    return !!quiz && quiz.answers.length >= quiz.questions.length;
  });

  /** F15: with questions, answering the last one finishes the step (the server checks it). */
  protected finishQuiz(): void {
    if (this.done() || this.review()) {
      return;
    }
    this.done.set(true);
    this.completed.emit();
  }

  protected readonly hostElement = this.host.nativeElement;
}

function readTipsHidden(): boolean {
  try {
    return localStorage.getItem(TIPS_KEY) === '1';
  } catch {
    return false;
  }
}
