import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  Injector,
  input,
  linkedSignal,
  OnInit,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, firstValueFrom, forkJoin, of } from 'rxjs';

import { DictationInput, DictationSummary } from '../../../core/models/dictation';
import { Sentence } from '../../../core/models/lesson';
import { ReadingLesson } from '../../../core/models/reading';
import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { AudioBar } from '../../../shared/components/audio-bar/audio-bar';
import { estimateSeconds } from '../../../shared/utils/fake-waveform';
import { Comparison, compareDictation } from '../../../shared/utils/dictation-compare';
import { mishearings } from '../../../shared/utils/mishearings';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { ListeningApiService } from '../listening-api.service';
import { ListeningHero } from './listening-hero/listening-hero';
import { Loading } from '../../../shared/components/loading/loading';

interface Checked {
  typed: string;
  comparison: Comparison;
}

const SPEEDS = [0.5, 0.75, 1, 1.25];

/** How a check went: every word right, at least half of them, or fewer. */
type Verdict = 'ok' | 'close' | 'bad';

const CHECK_LABEL = 'Kiểm tra câu trả lời';

/** One word of a sentence as a hint: how long it is and its first letter. */
export interface Slot {
  length: number;
  first: string;
}

/**
 * The hint of a sentence (updated 2026-10-20): one slot per word, as long as the word, and the mark
 * it ends with. Unlike its first words given in order, it never suggests where the sentence stops.
 */
export function sentenceSlots(text: string): { slots: Slot[]; end: string } {
  const slots = text
    .trim()
    .split(/\s+/)
    .map((w) => w.replace(/^[^\p{L}\p{N}]+|[^\p{L}\p{N}']+$/gu, ''))
    .filter(Boolean)
    .map((w) => ({ length: w.length, first: w[0] }));
  return { slots, end: /[.?!]$/.exec(text.trim())?.[0] ?? '' };
}

const VERDICT_LABELS: Record<Verdict, string> = { ok: 'Chính xác!', close: 'Gần đúng', bad: 'Chưa đúng' };

/**
 * The Listening step (F4): the browser reads the lesson sentence by sentence at a chosen speed
 * (no prepared audio), with an illustrative waveform that follows the sentence; the learner types
 * what was heard and sees which words were right, wrong or missing. Results are saved per sentence.
 */
@Component({
  selector: 'lu-listening',
  imports: [Loading, Icon, RouterLink, AudioBar, ListeningHero],
  templateUrl: './listening.html',
  styleUrl: './listening.css',
  host: { '[class.standalone]': 'mode() === null' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Listening implements OnInit {
  private readonly readingApi = inject(ReadingApiService);
  private readonly api = inject(ListeningApiService);
  private readonly route = inject(ActivatedRoute).snapshot;
  protected readonly speech = inject(SpeechService);

  /** The lesson to open; defaults to the route's :id (used by the lesson page, L). */
  readonly lessonId = input('');
  /** Sentence to open first (saved position); defaults to the first unchecked sentence. */
  readonly startSentence = input<number | null>(null);
  /** "review" replays a finished lesson without saving; defaults to ?review=1. */
  readonly mode = input<'study' | 'review' | null>(null);

  /** On the lesson page: leave the step still to do for later ("Để sau, sang bước …"). */
  readonly laterLabel = input('');
  /** On the lesson page: the main button once every sentence is checked ("Sang bước …"). */
  readonly nextLabel = input('');

  /** The lesson goes on: for later (laterLabel) or with the step done (nextLabel). */
  readonly leave = output<void>();
  /** Emitted once when every sentence has been checked (feature L records progress). */
  readonly completed = output<void>();
  /** The sentence the learner moved to (feature L saves it). */
  readonly position = output<number>();

  private get id(): string {
    return this.lessonId() || (this.route.paramMap.get('id') ?? '');
  }

  protected readonly review = computed(
    () => (this.mode() ?? (this.route.queryParamMap?.get('review') === '1' ? 'review' : 'study')) === 'review',
  );

  protected readonly speeds = SPEEDS;
  protected readonly lesson = signal<ReadingLesson | null>(null);
  /** Back to the path of the lesson's part, where the learner came from. */
  protected readonly backLink = computed(() => {
    const level = this.lesson()?.level;
    return level ? ['/lessons/levels', level] : ['/lessons'];
  });
  protected readonly loadError = signal<string | null>(null);
  protected readonly current = signal(0);
  protected readonly rate = signal(1);
  protected readonly showText = signal(false);
  protected readonly typed = signal('');
  protected readonly answerError = signal<string | null>(null);
  protected readonly playError = signal<string | null>(null);
  /** Whether the browser is reading the current sentence, and how much of it (0–1). */
  protected readonly speaking = signal(false);
  protected readonly progress = signal(0);
  protected readonly checked = signal<ReadonlyMap<number, Checked>>(new Map());
  protected readonly saveError = signal(false);
  protected readonly restartError = signal(false);
  /** Sentences whose latest check is not saved yet (mirrors pending, for the template). */
  private readonly unsaved = signal<ReadonlySet<number>>(new Set());
  /** Bumped by every check: the result card is drawn anew, so it shows up again even when unchanged. */
  protected readonly checks = signal(0);
  /** Read out after a check (the result card is drawn anew, so it cannot be the live region). */
  protected readonly announcement = signal('');
  private readonly resultCard = viewChild<ElementRef<HTMLElement>>('resultCard');
  private readonly chipRow = viewChild<ElementRef<HTMLElement>>('chipRow');
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);

  protected readonly sentences = computed(() => this.lesson()?.sentences ?? []);
  protected readonly sentence = computed<Sentence | null>(() => this.sentences()[this.current()] ?? null);
  /** Estimated reading time of the current sentence at the chosen speed, in seconds. */
  protected readonly duration = computed(() => {
    const s = this.sentence();
    return s ? estimateSeconds(s.text) / this.rate() : 0;
  });
  protected readonly result = computed(() => this.checked().get(this.current()) ?? null);
  protected readonly verdict = computed<Verdict | null>(() => {
    const c = this.result()?.comparison;
    if (!c) {
      return null;
    }
    if (c.totalWords > 0 && c.correctWords === c.totalWords && c.closeWords === 0) {
      return 'ok';
    }
    return c.correctWords * 2 >= c.totalWords ? 'close' : 'bad';
  });
  protected readonly verdictLabel = computed(() => {
    const v = this.verdict();
    return v ? VERDICT_LABELS[v] : '';
  });
  /** Whether the current sentence's result is saved, being saved, or (review) not saved at all. */
  protected readonly saveStatus = computed<'review' | 'saving' | 'failed' | 'saved'>(() => {
    if (this.review()) {
      return 'review';
    }
    if (this.unsaved().has(this.current())) {
      return this.saveError() ? 'failed' : 'saving';
    }
    return 'saved';
  });
  protected readonly isLastSentence = computed(() => this.current() >= this.sentences().length - 1);
  protected readonly hint = computed(() => {
    const s = this.sentence();
    return s ? sentenceSlots(s.text) : { slots: [], end: '' };
  });
  /** "Hé lộ chữ cái đầu": shows the first letter of each slot, until the sentence changes. */
  protected readonly firstLetters = linkedSignal(() => {
    this.current();
    return false;
  });
  protected readonly hintText = computed(() => {
    const { slots } = this.hint();
    const letters = this.firstLetters() ? `, chữ cái đầu: ${slots.map((s) => s.first).join(', ')}` : '';
    return `Câu có ${slots.length} từ${letters}`;
  });
  /**
   * What the main button at the bottom does: check the answer, or, once the shown answer is
   * checked (and not changed since), go on to the next sentence, the first one left, or the lesson.
   */
  private readonly mainStep = computed<'check' | 'next' | 'left' | 'leave'>(() => {
    const r = this.result();
    if (!r || this.typed().trim() !== r.typed) {
      return 'check';
    }
    if (this.done()) {
      return this.nextLabel() ? 'leave' : this.isLastSentence() ? 'check' : 'next';
    }
    return this.isLastSentence() ? 'left' : 'next';
  });
  protected readonly mainIsCheck = computed(() => this.mainStep() === 'check');
  protected readonly mainLabel = computed(() => {
    switch (this.mainStep()) {
      case 'next':
        return 'Câu tiếp theo';
      case 'left':
        return 'Câu chưa làm';
      case 'leave':
        return this.nextLabel();
      default:
        return CHECK_LABEL;
    }
  });
  protected readonly checkedCount = computed(() => this.checked().size);
  /** Once every sentence is checked: what the learner most often heard wrong (added 2026-10-20). */
  protected readonly slips = computed(() => mishearings([...this.checked().values()].map((c) => c.comparison)));
  /** Checked sentences typed with every word right (✓ on their number; the others get ✗). */
  protected readonly rightOnes = computed(() => {
    const right = new Set<number>();
    for (const [index, c] of this.checked()) {
      if (c.comparison.totalWords > 0 && c.comparison.correctWords === c.comparison.totalWords) {
        right.add(index);
      }
    }
    return right;
  });
  protected readonly done = computed(() => this.sentences().length > 0 && this.checkedCount() === this.sentences().length);
  protected readonly ratePercent = computed(() => {
    let correct = 0;
    let total = 0;
    for (const c of this.checked().values()) {
      correct += c.comparison.correctWords;
      total += c.comparison.totalWords;
    }
    return total ? Math.round((correct / total) * 100) : 0;
  });

  /** Unsaved checks by sentence index, oldest first; only the latest check of a sentence is kept. */
  private readonly pending = new Map<number, DictationInput>();
  private saving = false;
  private emitted = false;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.silence());
    // Keep the current sentence number (and the ones after it) in view.
    effect(() => {
      this.current();
      if (this.chipRow()) {
        afterNextRender(() => this.centerChip(), { injector: this.injector });
      }
    });
    // With the natural voice, generate the sentences ahead, from the current one.
    effect(() => {
      const texts = this.sentences().map((s) => s.text);
      if (!this.speech.natural()) {
        return;
      }
      const from = untracked(() => this.current());
      this.speech.prefetch([...texts.slice(from), ...texts.slice(0, from)]);
    });
  }

  /**
   * Slides the row of sentence numbers sideways so the current one is in the middle, with the
   * next ones in view (only the row moves, not the page).
   */
  private centerChip(): void {
    const row = this.chipRow()?.nativeElement;
    const chip = row?.querySelector<HTMLElement>('.chip.current')?.parentElement;
    if (!row || !chip || row.scrollWidth <= row.clientWidth) {
      return;
    }
    const left = chip.offsetLeft - (row.clientWidth - chip.offsetWidth) / 2;
    const smooth = !window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    row.scrollTo?.({ left: Math.max(0, left), behavior: smooth ? 'smooth' : 'auto' });
  }

  ngOnInit(): void {
    // Inputs are set by now; review mode neither restores nor saves results.
    const summary = this.review() ? of(null) : this.api.summary(this.id).pipe(catchError(() => of(null)));
    forkJoin({ lesson: this.readingApi.getLesson(this.id), summary }).subscribe({
      next: ({ lesson, summary }) => this.start(lesson, summary),
      error: (err: unknown) => this.loadError.set(loadErrorMessage(err)),
    });
  }

  private start(lesson: ReadingLesson, summary: DictationSummary | null): void {
    const checked = new Map<number, Checked>();
    for (const r of summary?.results ?? []) {
      const s = lesson.sentences[r.sentenceIndex];
      if (s) {
        checked.set(r.sentenceIndex, { typed: r.typed, comparison: compareDictation(s.text, r.typed) });
      }
    }
    this.emitted = checked.size === lesson.sentences.length;
    this.checked.set(checked);
    this.lesson.set(lesson);
    const firstUnchecked = lesson.sentences.findIndex((s) => !checked.has(s.index));
    const start = this.startSentence();
    this.show(start !== null && start >= 0 && start < lesson.sentences.length ? start : Math.max(firstUnchecked, 0));
  }

  // --- listening ---

  private show(index: number): void {
    this.silence();
    this.current.set(index);
    this.typed.set(this.checked().get(index)?.typed ?? '');
    this.answerError.set(null);
    this.playError.set(null);
  }

  /** Làm lại câu này: the answer is cleared to type again; the new check replaces the result. */
  protected redoSentence(): void {
    this.typed.set('');
    this.answerError.set(null);
    this.host.nativeElement.querySelector<HTMLTextAreaElement>('#answer')?.focus();
    this.listen();
  }

  /**
   * Làm lại bước này: every result is forgotten (on the server too, outside review mode) and the
   * step starts again at the first sentence.
   */
  protected async restart(): Promise<void> {
    this.restartError.set(false);
    if (!this.review()) {
      try {
        await firstValueFrom(this.api.reset(this.id));
      } catch {
        this.restartError.set(true);
        return;
      }
    }
    this.pending.clear();
    this.unsaved.set(new Set());
    this.saveError.set(false);
    this.checked.set(new Map());
    this.emitted = false;
    this.announcement.set('');
    this.show(0);
    this.position.emit(0);
  }

  protected go(index: number): void {
    if (index < 0 || index >= this.sentences().length || index === this.current()) {
      return;
    }
    this.show(index);
    this.position.emit(index);
    this.listen();
  }

  /** Reads the current sentence from the start; the player bar follows it. */
  protected listen(): void {
    const sentence = this.sentence();
    if (!sentence || !this.speech.supported) {
      return;
    }
    this.silence();
    this.playError.set(null);
    this.speaking.set(true);
    this.speech.speak(sentence.text, this.rate(), {
      progress: (share) => this.progress.set(share),
      ended: () => this.speaking.set(false),
      failed: () => {
        this.speaking.set(false);
        this.progress.set(0);
        this.playError.set('Chưa đọc được câu. Hãy thử lại.');
      },
    });
  }

  /** The player button: stops the reading, or reads the sentence again from the start. */
  protected togglePlay(): void {
    if (this.speaking()) {
      this.silence();
    } else {
      this.listen();
    }
  }

  /** Stops reading and empties the player bar. */
  private silence(): void {
    this.speech.stop();
    this.speaking.set(false);
    this.progress.set(0);
  }

  protected setRate(rate: number): void {
    this.rate.set(rate);
  }

  protected onSpeedKeydown(event: KeyboardEvent): void {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const i = (SPEEDS.indexOf(this.rate()) + step + SPEEDS.length) % SPEEDS.length;
    this.rate.set(SPEEDS[i]);
    const group = (event.currentTarget as HTMLElement).closest('[role="radiogroup"]');
    group?.querySelectorAll<HTMLElement>('[role="radio"]')[i]?.focus();
  }

  // --- dictation ---

  protected onInput(event: Event): void {
    // A pasted line break is a space: the answer is one sentence.
    this.typed.set((event.target as HTMLTextAreaElement).value.replace(/\s*\n\s*/g, ' '));
    this.answerError.set(null);
  }

  protected clearAnswer(): void {
    this.typed.set('');
    this.answerError.set(null);
    this.host.nativeElement.querySelector<HTMLTextAreaElement>('#answer')?.focus();
  }

  /** The main button at the bottom (see mainStep). */
  protected mainAction(event?: Event): void {
    event?.preventDefault();
    switch (this.mainStep()) {
      case 'next':
        this.go(this.current() + 1);
        return;
      case 'left':
        this.go(this.sentences().findIndex((s) => !this.checked().has(s.index)));
        return;
      case 'leave':
        this.leave.emit();
        return;
      default:
        this.check();
    }
  }

  protected onFocus(event: FocusEvent): void {
    // Keep the answer bar visible above the on-screen keyboard.
    (event.target as HTMLElement).scrollIntoView({ block: 'nearest' });
  }

  protected check(event?: Event): void {
    event?.preventDefault();
    const sentence = this.sentence();
    const typed = this.typed().trim();
    if (!sentence) {
      return;
    }
    if (!typed) {
      this.answerError.set('Hãy gõ câu bạn nghe được');
      return;
    }
    const comparison = compareDictation(sentence.text, typed);
    this.checked.update((m) => new Map(m).set(sentence.index, { typed, comparison }));
    this.checks.update((n) => n + 1);
    this.announcement.set(`${this.verdictLabel()} ${comparison.correctWords}/${comparison.totalWords} từ đúng.`);
    // Bring the result into view: on a phone it is often below the fold, under the answer bar.
    afterNextRender(
      () => {
        const smooth = !window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
        this.resultCard()?.nativeElement.scrollIntoView?.({ block: 'nearest', behavior: smooth ? 'smooth' : 'auto' });
      },
      { injector: this.injector },
    );

    if (this.review()) {
      return; // replaying a finished lesson: nothing is saved
    }
    if (this.done() && !this.emitted) {
      this.emitted = true;
      this.completed.emit();
    }

    this.pending.delete(sentence.index); // keep insertion order: the latest check goes last
    this.pending.set(sentence.index, {
      sentenceIndex: sentence.index,
      typed,
      correctWords: comparison.correctWords,
      totalWords: comparison.totalWords,
    });
    this.unsaved.set(new Set(this.pending.keys()));
    void this.flush();
  }

  /** Sends unsaved checks one by one; stops at the first failure and keeps the rest. */
  protected async flush(): Promise<void> {
    if (this.saving) {
      return;
    }
    this.saving = true;
    try {
      for (const [index, input] of this.pending) {
        await firstValueFrom(this.api.record(this.id, input));
        if (this.pending.get(index) === input) {
          this.pending.delete(index);
          this.unsaved.set(new Set(this.pending.keys()));
        }
      }
      this.saveError.set(false);
    } catch {
      this.saveError.set(true);
    } finally {
      this.saving = false;
    }
  }
}
