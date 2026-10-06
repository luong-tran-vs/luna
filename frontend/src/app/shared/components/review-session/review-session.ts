import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  Injector,
  input,
  linkedSignal,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { DueCard, Intervals, Rating, ReviewContext, ReviewMode, ReviewSummary } from '../../../core/models/vocab';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { isCorrectAnswer } from '../../utils/answer-match';
import { intervalLabel } from '../../utils/interval-label';
import { SpeechService } from '../../../core/services/speech.service';

interface RatingButton {
  rating: Rating;
  name: string;
  key: keyof Intervals;
}

const RATINGS: RatingButton[] = [
  { rating: 1, name: 'Quên', key: 'again' },
  { rating: 2, name: 'Khó', key: 'hard' },
  { rating: 3, name: 'Nhớ', key: 'good' },
  { rating: 4, name: 'Dễ', key: 'easy' },
];

/**
 * One review session over the given cards (F5, reused by the daily flow L). Flip mode shows the
 * word and flips to the meaning; listen mode plays the word and checks what the learner types.
 * Each rating is saved right away; a card rated Again comes back once at the end. Back and
 * forward move through the cards already rated (view only); forward on the current card skips
 * it to the end of the session.
 */
@Component({
  selector: 'lu-review-session',
  templateUrl: './review-session.html',
  styleUrl: './review-session.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(keydown)': 'onKeydown($event)' },
})
export class ReviewSession {
  private readonly api = inject(VocabApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);

  readonly cards = input.required<DueCard[]>();
  readonly mode = input.required<ReviewMode>();
  /** 'daily' in the review step of today's lesson (counted against the daily limit). */
  readonly context = input<ReviewContext>('free');
  /** Emitted once when every card has been rated. */
  readonly finished = output<ReviewSummary>();

  protected readonly ratings = RATINGS;
  protected readonly queue = linkedSignal(() => [...this.cards()]);
  /** The card on screen. */
  protected readonly index = signal(0);
  /** The first card not rated yet; every card before it is rated and only viewed again. */
  private readonly frontier = signal(0);
  protected readonly current = computed<DueCard | null>(() => this.queue()[this.index()] ?? null);
  protected readonly reviewing = computed(() => this.index() < this.frontier());
  protected readonly shown = computed(() => this.reviewing() || this.revealed());
  /** The rating given to each card before the frontier, by position. */
  protected readonly given = signal<Rating[]>([]);
  protected readonly revealed = signal(false);
  protected readonly typed = signal('');
  protected readonly verdict = signal<'correct' | 'wrong' | null>(null);
  protected readonly answerError = signal<string | null>(null);
  protected readonly playError = signal(false);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  protected readonly summary = signal<ReviewSummary | null>(null);
  protected readonly canBack = computed(() => this.index() > 0 && !this.saving());
  protected readonly canForward = computed(
    () => !this.saving() && (this.reviewing() || this.frontier() + 1 < this.queue().length),
  );

  private readonly face = viewChild<ElementRef<HTMLButtonElement>>('face');
  private readonly answer = viewChild<ElementRef<HTMLInputElement>>('answer');
  private readonly requeued = new Set<string>();
  private readonly reviewed = new Set<string>();
  private counts: Record<Rating, number> = { 1: 0, 2: 0, 3: 0, 4: 0 };
  private lastRating: Rating | null = null;

  constructor() {
    // With the natural voice, generate the words of the session ahead.
    effect(() => {
      const texts = this.cards().map((c) => c.text);
      if (this.speech.natural()) {
        this.speech.prefetch(texts);
      }
    });
    // Listen mode plays each new card on its own (not the rated ones viewed again).
    effect(() => {
      const card = this.current();
      if (card && this.mode() === 'listen' && !this.reviewing()) {
        untracked(() => this.play());
      }
    });
  }

  protected label(card: DueCard, r: RatingButton): string {
    return `${r.name} · ${intervalLabel(card.intervals[r.key])}`;
  }

  protected ratingName(rating: Rating): string {
    return RATINGS[rating - 1].name;
  }

  protected back(): void {
    if (this.canBack()) {
      this.index.update((i) => i - 1);
    }
  }

  /** Forward through rated cards, or skip the current one to the end of the session. */
  protected forward(): void {
    if (!this.canForward()) {
      return;
    }
    if (this.reviewing()) {
      this.index.update((i) => i + 1);
      if (!this.reviewing()) {
        this.focusAnswer();
      }
      return;
    }
    const at = this.frontier();
    this.queue.update((q) => [...q.slice(0, at), ...q.slice(at + 1), q[at]]);
    this.resetCard();
    this.focusAnswer();
  }

  protected play(): void {
    const card = this.current();
    if (card) {
      this.playError.set(false);
      this.speech.speak(card.text, 1, { failed: () => this.playError.set(true) });
    }
  }

  protected flip(): void {
    if (!this.revealed()) {
      this.revealed.set(true);
    }
  }

  protected onInput(event: Event): void {
    this.typed.set((event.target as HTMLInputElement).value);
    this.answerError.set(null);
  }

  protected onFocus(event: FocusEvent): void {
    (event.target as HTMLElement).scrollIntoView({ block: 'nearest' });
  }

  protected check(event?: Event): void {
    event?.preventDefault();
    const card = this.current();
    if (!card || this.revealed()) {
      return;
    }
    if (!this.typed().trim()) {
      this.answerError.set('Hãy gõ từ bạn nghe được');
      return;
    }
    this.verdict.set(isCorrectAnswer(this.typed(), card.text) ? 'correct' : 'wrong');
    this.revealed.set(true);
  }

  protected onKeydown(event: KeyboardEvent): void {
    const target = event.target as HTMLElement;
    if (target.tagName === 'INPUT' || this.summary()) {
      return;
    }
    if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
      event.preventDefault();
      if (event.key === 'ArrowLeft') {
        this.back();
      } else {
        this.forward();
      }
      return;
    }
    if (this.reviewing()) {
      return;
    }
    if (!this.revealed()) {
      if (this.mode() === 'flip' && (event.key === ' ' || event.key === 'Enter') && target.tagName !== 'BUTTON') {
        event.preventDefault();
        this.flip();
      }
      return;
    }
    const rating = Number(event.key);
    if (rating >= 1 && rating <= 4) {
      event.preventDefault();
      void this.rate(rating as Rating);
    }
  }

  protected retry(): void {
    if (this.lastRating) {
      void this.rate(this.lastRating);
    }
  }

  protected async rate(rating: Rating): Promise<void> {
    const card = this.current();
    if (!card || !this.revealed() || this.saving() || this.reviewing()) {
      return;
    }
    this.lastRating = rating;
    this.saving.set(true);
    this.saveError.set(false);
    try {
      const updated = await firstValueFrom(this.api.review(card.id, { rating, mode: this.mode(), reps: card.reps, context: this.context() }));
      this.counts = { ...this.counts, [rating]: this.counts[rating] + 1 };
      this.reviewed.add(card.id);
      this.given.update((g) => [...g, rating]);
      if (rating === 1 && !this.requeued.has(card.id)) {
        this.requeued.add(card.id);
        this.queue.update((q) => [...q, updated]);
      }
      this.advance();
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        this.given.update((g) => [...g, rating]);
        this.advance(); // already rated (e.g. a retry after a lost response)
      } else {
        this.saveError.set(true);
      }
    } finally {
      this.saving.set(false);
    }
  }

  private advance(): void {
    this.resetCard();
    if (this.frontier() + 1 >= this.queue().length) {
      const summary: ReviewSummary = { reviewed: this.reviewed.size, counts: this.counts };
      this.summary.set(summary);
      this.finished.emit(summary);
      return;
    }
    this.frontier.update((f) => f + 1);
    this.index.set(this.frontier());
    this.focusAnswer();
  }

  /** Clears what the learner did on the current card. */
  private resetCard(): void {
    this.revealed.set(false);
    this.typed.set('');
    this.verdict.set(null);
    this.answerError.set(null);
    this.playError.set(false);
    this.saveError.set(false);
    this.lastRating = null;
  }

  private focusAnswer(): void {
    afterNextRender(() => (this.answer() ?? this.face())?.nativeElement.focus(), { injector: this.injector });
  }

}
