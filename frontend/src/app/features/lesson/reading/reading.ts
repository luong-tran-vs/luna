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
import { ReadingLesson, Token } from '../../../core/models/reading';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { CardInput } from '../../../core/models/vocab';
import { PopupState, WordPopup } from '../../../shared/components/word-popup/word-popup';
import { selectWords, TouchedWord } from '../../../shared/utils/selection';
import { tokenize } from '../../../shared/utils/tokenize';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { LessonVocabulary } from './lesson-vocabulary/lesson-vocabulary';

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

/**
 * The Reading step (F3): read the lesson, tap a word or select a phrase to look it up, hear
 * it, save it to the notebook, and finish when the end of the lesson has been seen.
 */
@Component({
  selector: 'lu-reading',
  imports: [CdkConnectedOverlay, LessonVocabulary, RouterLink, WordPopup],
  templateUrl: './reading.html',
  styleUrl: './reading.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '(document:selectionchange)': 'onSelectionChange()',
    // Word interactions are delegated from the host; each word is a focusable role="button".
    '(pointerup)': 'onPointerUp()',
    '(keydown)': 'onArticleKeydown($event)',
    '(focusin)': 'onFocusIn($event)',
  },
})
export class Reading implements OnInit {
  private readonly api = inject(ReadingApiService);
  private readonly vocab = inject(VocabApiService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly route = inject(ActivatedRoute).snapshot;
  private readonly injector = inject(Injector);

  /** The lesson to open; defaults to the route's :id (used by the daily flow, L). */
  readonly lessonId = input('');
  /** Sentence to scroll to first (saved position). */
  readonly startSentence = input<number | null>(null);
  /** "review" rereads a finished lesson (no "Đã đọc xong"); defaults to ?review=1. */
  readonly mode = input<'study' | 'review' | null>(null);

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
  protected readonly hint = signal<string | null>(null);
  protected readonly popupSaved = computed(() => {
    const state = this.popupState();
    const sel = this.selected();
    const lemma = state.kind === 'result' ? state.result.lemma : sel ? normalize(sel.text) : '';
    return !!lemma && this.saved().has(normalize(lemma));
  });

  protected readonly reachedEnd = signal(false);
  protected readonly done = signal(false);

  private readonly article = viewChild<ElementRef<HTMLElement>>('article');
  private readonly end = viewChild<ElementRef<HTMLElement>>('end');
  private readonly audio = viewChild<ElementRef<HTMLAudioElement>>('audio');
  private lookupSub?: Subscription;
  private selectionTimer?: ReturnType<typeof setTimeout>;

  ngOnInit(): void {
    // Inputs are set by now.
    forkJoin({
      lesson: this.api.getLesson(this.id),
      words: this.vocab.words().pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ lesson, words }) => {
        this.saved.set(new Set(words.map((w) => normalize(w.lemma))));
        this.lesson.set(lesson);
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

    // Report the first sentence on screen (saved as the position in the daily flow, L).
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
      clearTimeout(this.selectionTimer);
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
    this.open(s, t, t, event.currentTarget as HTMLElement);
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

  protected onPointerUp(): void {
    if (this.anchor() === null || document.getSelection()?.isCollapsed === false) {
      this.handleSelection();
    }
  }

  protected onSelectionChange(): void {
    // Touch selection handles do not end with pointerup on a word: wait until the learner stops.
    clearTimeout(this.selectionTimer);
    this.selectionTimer = setTimeout(() => this.handleSelection(), 150);
  }

  private handleSelection(): void {
    const sel = document.getSelection();
    const article = this.article()?.nativeElement;
    if (!sel || sel.isCollapsed || sel.rangeCount === 0 || !article) {
      return;
    }
    const range = sel.getRangeAt(0);
    if (!article.contains(range.commonAncestorContainer)) {
      return;
    }
    const spans = Array.from(article.querySelectorAll<HTMLElement>('.w')).filter((w) => range.intersectsNode(w));
    const touched: TouchedWord[] = spans.map((w) => ({ s: Number(w.dataset['s']), t: Number(w.dataset['t']) }));
    const result = selectWords(touched);
    if (!result) {
      return;
    }
    if ('error' in result) {
      this.hint.set('Chọn tối đa 6 từ để tra cụm.');
      return;
    }
    this.hint.set(null);
    const anchor = spans.find((w) => Number(w.dataset['s']) === result.sentence && Number(w.dataset['t']) === result.first);
    sel.removeAllRanges(); // the system selection menu would cover the popup
    this.open(result.sentence, result.first, result.last, anchor ?? article);
  }

  private open(sentence: number, first: number, last: number, anchor: HTMLElement): void {
    const text = this.tokens()[sentence].slice(first, last + 1).map((t) => t.text).join('').trim().replace(/\s+/g, ' ');
    const current = this.selected();
    if (current && current.sentence === sentence && current.first === first && current.last === last && this.anchor()) {
      return; // same selection reported twice (click + selection)
    }
    this.selected.set({ sentence, first, last, text });
    this.anchor.set(anchor);
    this.popupState.set({ kind: 'loading' });
    this.popupError.set(null);

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
    this.anchor.set(null);
    this.selected.set(null);
    anchor?.focus();
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
      source: result ? result.source : 'manual',
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

  protected onVocabularyAdded(lemmas: string[]): void {
    this.saved.update((s) => new Set([...s, ...lemmas.map(normalize)]));
  }

  protected play(): void {
    const audio = this.audio()?.nativeElement;
    const state = this.popupState();
    const text = state.kind === 'result' ? state.result.lemma : this.selected()?.text;
    if (!audio || !text) {
      return;
    }
    const url = this.vocab.wordAudioUrl(text);
    if (audio.getAttribute('src') === url) {
      try {
        audio.currentTime = 0; // replay from the start instead of overlapping
      } catch {
        // some environments cannot seek before metadata loads; playing again is enough
      }
    } else {
      audio.src = url;
    }
    this.popupError.set(null);
    audio.play().catch(() => this.popupError.set('Chưa phát được âm thanh.'));
  }

  // --- finishing ---

  protected finish(): void {
    if (!this.reachedEnd() || this.done()) {
      return;
    }
    this.done.set(true);
    this.completed.emit();
  }

  protected readonly hostElement = this.host.nativeElement;
}
