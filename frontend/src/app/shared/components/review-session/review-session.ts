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
  { rating: 1, name: 'Again', key: 'again' },
  { rating: 2, name: 'Hard', key: 'hard' },
  { rating: 3, name: 'Good', key: 'good' },
  { rating: 4, name: 'Easy', key: 'easy' },
];

/**
 * One review session over the given cards (F5, reused by the daily flow L). Flip mode shows the
 * word and flips to the meaning; listen mode plays the word and checks what the learner types.
 * Each rating is saved right away; a card rated Again comes back once at the end.
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
  protected readonly index = signal(0);
  protected readonly current = computed<DueCard | null>(() => this.queue()[this.index()] ?? null);
  protected readonly revealed = signal(false);
  protected readonly typed = signal('');
  protected readonly verdict = signal<'correct' | 'wrong' | null>(null);
  protected readonly answerError = signal<string | null>(null);
  protected readonly playError = signal(false);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  protected readonly summary = signal<ReviewSummary | null>(null);

  private readonly face = viewChild<ElementRef<HTMLButtonElement>>('face');
  private readonly answer = viewChild<ElementRef<HTMLInputElement>>('answer');
  private readonly requeued = new Set<string>();
  private readonly reviewed = new Set<string>();
  private counts: Record<Rating, number> = { 1: 0, 2: 0, 3: 0, 4: 0 };
  private lastRating: Rating | null = null;

  constructor() {
    // Listen mode plays each new card on its own.
    effect(() => {
      const card = this.current();
      if (card && this.mode() === 'listen') {
        untracked(() => this.play());
      }
    });
  }

  protected label(card: DueCard, r: RatingButton): string {
    return `${r.name} · ${intervalLabel(card.intervals[r.key])}`;
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
    if (!card || !this.revealed() || this.saving()) {
      return;
    }
    this.lastRating = rating;
    this.saving.set(true);
    this.saveError.set(false);
    try {
      const updated = await firstValueFrom(this.api.review(card.id, { rating, mode: this.mode(), reps: card.reps, context: this.context() }));
      this.counts = { ...this.counts, [rating]: this.counts[rating] + 1 };
      this.reviewed.add(card.id);
      if (rating === 1 && !this.requeued.has(card.id)) {
        this.requeued.add(card.id);
        this.queue.update((q) => [...q, updated]);
      }
      this.advance();
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        this.advance(); // already rated (e.g. a retry after a lost response)
      } else {
        this.saveError.set(true);
      }
    } finally {
      this.saving.set(false);
    }
  }

  private advance(): void {
    this.revealed.set(false);
    this.typed.set('');
    this.verdict.set(null);
    this.answerError.set(null);
    this.playError.set(false);
    this.lastRating = null;
    if (this.index() + 1 >= this.queue().length) {
      const summary: ReviewSummary = { reviewed: this.reviewed.size, counts: this.counts };
      this.summary.set(summary);
      this.finished.emit(summary);
      return;
    }
    this.index.update((i) => i + 1);
    afterNextRender(() => (this.answer() ?? this.face())?.nativeElement.focus(), { injector: this.injector });
  }

}
