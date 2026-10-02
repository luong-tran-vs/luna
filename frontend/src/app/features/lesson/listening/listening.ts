import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  OnInit,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, firstValueFrom, forkJoin, of } from 'rxjs';

import { DictationInput, DictationSummary } from '../../../core/models/dictation';
import { Sentence } from '../../../core/models/lesson';
import { ReadingLesson } from '../../../core/models/reading';
import { AudioPlayer } from '../../../shared/components/audio-player/audio-player';
import { Icon } from '../../../shared/components/icon/icon';
import { Comparison, compareDictation } from '../../../shared/utils/dictation-compare';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { ListeningApiService } from '../listening-api.service';

interface Checked {
  typed: string;
  comparison: Comparison;
}

const SPEEDS = [0.5, 0.75, 1, 1.25];


/**
 * The Listening step (F4): play the lesson sentence by sentence at a chosen speed, type what
 * was heard, and see which words were right, wrong or missing. Results are saved per sentence.
 */
@Component({
  selector: 'lu-listening',
  imports: [AudioPlayer, Icon, RouterLink],
  templateUrl: './listening.html',
  styleUrl: './listening.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Listening implements OnInit {
  private readonly readingApi = inject(ReadingApiService);
  private readonly api = inject(ListeningApiService);
  private readonly route = inject(ActivatedRoute).snapshot;

  /** The lesson to open; defaults to the route's :id (used by the daily flow, L). */
  readonly lessonId = input('');
  /** Sentence to open first (saved position); defaults to the first unchecked sentence. */
  readonly startSentence = input<number | null>(null);
  /** "review" replays a finished lesson without saving; defaults to ?review=1. */
  readonly mode = input<'study' | 'review' | null>(null);

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
  protected readonly loadError = signal<string | null>(null);
  protected readonly current = signal(0);
  protected readonly rate = signal(1);
  protected readonly showText = signal(false);
  protected readonly typed = signal('');
  protected readonly answerError = signal<string | null>(null);
  protected readonly playError = signal<string | null>(null);
  protected readonly checked = signal<ReadonlyMap<number, Checked>>(new Map());
  protected readonly saveError = signal(false);

  protected readonly sentences = computed(() => this.lesson()?.sentences ?? []);
  protected readonly sentence = computed<Sentence | null>(() => this.sentences()[this.current()] ?? null);
  protected readonly hasAudio = computed(() => this.sentences().some((s) => s.audioUrl));
  protected readonly result = computed(() => this.checked().get(this.current()) ?? null);
  protected readonly checkedCount = computed(() => this.checked().size);
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

  private readonly player = viewChild(AudioPlayer);
  /** Unsaved checks by sentence index, oldest first; only the latest check of a sentence is kept. */
  private readonly pending = new Map<number, DictationInput>();
  private saving = false;
  private emitted = false;

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
    this.current.set(index);
    this.typed.set(this.checked().get(index)?.typed ?? '');
    this.answerError.set(null);
    this.playError.set(null);
  }

  protected go(index: number): void {
    if (index < 0 || index >= this.sentences().length || index === this.current()) {
      return;
    }
    this.show(index);
    this.position.emit(index);
    const src = this.sentence()?.audioUrl ?? null;
    if (src) {
      this.player()?.replay(src);
    } else {
      this.player()?.stop();
    }
  }

  protected listen(): void {
    this.playError.set(null);
    this.player()?.replay(this.sentence()?.audioUrl ?? null);
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
    this.typed.set((event.target as HTMLInputElement).value);
    this.answerError.set(null);
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
