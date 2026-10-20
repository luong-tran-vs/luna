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
import { DueCard, Intervals, PoolMeaning, Rating, ReviewContext, ReviewMode, ReviewSummary } from '../../../core/models/vocab';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { letterHint, matchAnswer } from '../../utils/answer-match';
import { makeChoices, seedOf } from '../../utils/choices';
import { intervalLabel } from '../../utils/interval-label';
import { SpeechService } from '../../../core/services/speech.service';
import { SpeakButton } from '../../directives/speak-button';
import { Asker } from '../asker/asker';
import { ChoiceList } from '../choice-list/choice-list';
import { Icon } from '../icon/icon';

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

/** The speed of "Nghe chậm" in listen mode. */
const SLOW_RATE = 0.6;

/**
 * One review session over the given cards (F5, reused by the daily flow L). Flip mode shows the
 * word and four meanings to pick from (design/tat-ca-de-xuat.png (mục on-tap-theo-bai)): "Nộp" checks the pick, "Quên"
 * shows the answer, and the card is rated for the learner (right: Good, else Again) on "Tiếp". With
 * fewer than two meanings to choose from it falls back to flipping the card and rating it by hand.
 * Listen mode plays the word and checks what the learner types. Each rating is saved right away; a card rated Again comes back once at the end. Back and
 * forward move through the cards already rated (view only); forward on the current card skips
 * it to the end of the session. Laid out like a lesson step (design, 2026-10-15): ✕ and the
 * progress at the top, the boy asking in a bubble, the buttons in a bar at the bottom.
 */
@Component({
  selector: 'lu-review-session',
  imports: [Asker, ChoiceList, Icon, SpeakButton],
  templateUrl: './review-session.html',
  styleUrl: './review-session.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(keydown)': 'onKeydown($event)', '[class.has-bar]': '!summary()' },
})
export class ReviewSession {
  private readonly api = inject(VocabApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);

  readonly cards = input.required<DueCard[]>();
  readonly mode = input.required<ReviewMode>();
  /** 'daily' in the review step of today's lesson (counted against the daily limit). */
  readonly context = input<ReviewContext>('free');
  /** More meanings to pick the wrong options from, besides the other cards of the session. */
  readonly pool = input<readonly PoolMeaning[]>([]);
  /** Emitted once when every card has been rated. */
  readonly finished = output<ReviewSummary>();
  /** ✕: the learner stops the session (the ratings given so far are saved). */
  readonly closed = output();

  protected readonly ratings = RATINGS;
  protected readonly slowRate = SLOW_RATE;
  protected readonly queue = linkedSignal(() => [...this.cards()]);
  /** The card on screen. */
  protected readonly index = signal(0);
  /** The first card not rated yet; every card before it is rated and only viewed again. */
  private readonly frontier = signal(0);
  protected readonly current = computed<DueCard | null>(() => this.queue()[this.index()] ?? null);
  protected readonly reviewing = computed(() => this.index() < this.frontier());
  protected readonly shown = computed(() => this.reviewing() || this.revealed());
  /**
   * The card's example sentence cut around the word (case ignored, else its lemma) so the word can be
   * marked; flip mode shows it on the front to guess the meaning from context. Empty without one.
   */
  protected readonly example = computed(() => exampleParts(this.current()));
  /**
   * Flip mode: the card's meaning among others, the same order while the card is on screen. Wrong
   * meanings of the same part of speech come first (updated 2026-10-20): "chạy" next to "đi" is
   * harder to rule out than "ngôi nhà".
   */
  protected readonly choices = computed(() => {
    const card = this.current();
    if (!card || this.mode() !== 'flip') {
      return null;
    }
    const pool: PoolMeaning[] = [...this.cards(), ...this.pool()];
    const same = card.pos ? pool.filter((p) => p.pos === card.pos).map((p) => p.meaningVi) : [];
    return makeChoices(card.meaningVi, pool.map((p) => p.meaningVi), seedOf(card.id), same);
  });
  /** Flip mode as a multiple choice question; false when there is nothing to choose between. */
  protected readonly quiz = computed(() => (this.choices()?.options.length ?? 0) >= 2);
  /** The meaning picked, null for none. */
  protected readonly picked = signal<number | null>(null);
  /** The rating given to each card before the frontier, by position. */
  protected readonly given = signal<Rating[]>([]);
  protected readonly revealed = signal(false);
  protected readonly typed = signal('');
  /** How the answer went ('close': a typed word a little off); 'forgot' when the learner pressed "Quên". */
  protected readonly verdict = signal<'correct' | 'close' | 'wrong' | 'forgot' | null>(null);
  /** Recall mode: the first letter of each word and a blank for each other letter, "l _ _ _". */
  protected readonly letters = computed(() => letterHint(this.current()?.text ?? ''));
  protected readonly letterCount = computed(() => (this.current()?.text ?? '').replace(/\s/g, '').length);
  /** The rating "Tiếp" saves for a multiple choice answer. */
  protected readonly autoRating = computed<Rating>(() => (this.verdict() === 'correct' ? 3 : 1));
  protected readonly answerError = signal<string | null>(null);
  protected readonly playError = signal(false);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  protected readonly summary = signal<ReviewSummary | null>(null);
  /** The small line over the bubble: the question, then how the typed word went. */
  protected readonly askLabel = computed(() => {
    const v = this.reviewing() ? null : this.verdict();
    if (v) {
      return { correct: '✓ Đúng rồi', close: '≈ Gần đúng', wrong: '✗ Chưa đúng', forgot: 'Nghĩa của từ' }[v];
    }
    return { listen: 'Nghe và gõ lại từ', recall: 'Gõ từ tiếng Anh', flip: 'Từ này nghĩa là gì?' }[this.mode()];
  });
  protected readonly tone = computed(() => {
    const v = this.reviewing() ? null : this.verdict();
    return v === 'correct' ? 'right' : v === 'wrong' || v === 'forgot' ? 'wrong' : null;
  });
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

  protected interval(card: DueCard, r: RatingButton): string {
    return intervalLabel(card.intervals[r.key]);
  }

  /** When the card comes back with the rating "Tiếp" saves, e.g. "10 phút". */
  protected autoInterval(card: DueCard): string {
    return intervalLabel(card.intervals[this.autoRating() === 3 ? 'good' : 'again']);
  }

  protected pick(i: number): void {
    if (!this.revealed() && !this.reviewing()) {
      this.picked.set(i);
    }
  }

  /** "Nộp": checks the meaning picked. */
  protected submit(): void {
    const choices = this.choices();
    const picked = this.picked();
    if (!choices || picked === null || this.revealed() || this.reviewing()) {
      return;
    }
    this.verdict.set(picked === choices.answer ? 'correct' : 'wrong');
    this.revealed.set(true);
    this.focusNext();
  }

  /** "Quên": shows the answer without a guess. */
  protected forget(): void {
    if (this.revealed() || this.reviewing()) {
      return;
    }
    this.picked.set(null);
    this.verdict.set('forgot');
    this.revealed.set(true);
    this.focusNext();
  }

  /** "Tiếp": saves the rating the answer earned and moves on. */
  protected next(): void {
    void this.rate(this.autoRating());
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

  protected play(rate = 1): void {
    const card = this.current();
    if (card) {
      this.playError.set(false);
      this.speech.speak(card.text, rate, { failed: () => this.playError.set(true) });
    }
  }

  protected playExample(): void {
    const sentence = this.current()?.contextSentence;
    if (sentence) {
      this.playError.set(false);
      this.speech.speak(sentence, 1, { failed: () => this.playError.set(true) });
    }
  }

  protected flip(): void {
    if (!this.revealed()) {
      this.revealed.set(true);
    }
  }

  /** Flip mode, not rated yet: back to the front, to think again before rating. */
  protected unflip(): void {
    if (this.mode() === 'flip' && this.revealed() && !this.reviewing() && !this.saving()) {
      this.revealed.set(false);
      this.focusAnswer();
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
      this.answerError.set(this.mode() === 'recall' ? 'Hãy gõ từ tiếng Anh' : 'Hãy gõ từ bạn nghe được');
      return;
    }
    const match = matchAnswer(this.typed(), card.text);
    this.verdict.set(match === 'exact' ? 'correct' : match);
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
    if (this.quiz()) {
      this.onQuizKeydown(event, target);
      return;
    }
    if (!this.revealed()) {
      if (this.mode() === 'flip' && (event.key === ' ' || event.key === 'Enter') && target.tagName !== 'BUTTON') {
        event.preventDefault();
        this.flip();
      }
      return;
    }
    if (this.mode() === 'flip' && event.key === ' ' && target.tagName !== 'BUTTON') {
      event.preventDefault();
      this.unflip();
      return;
    }
    const rating = Number(event.key);
    if (rating >= 1 && rating <= 4) {
      event.preventDefault();
      void this.rate(rating as Rating);
    }
  }

  /** Multiple choice: 1–4 pick a meaning, Enter checks it, then Enter goes on. */
  private onQuizKeydown(event: KeyboardEvent, target: HTMLElement): void {
    if (event.key === 'Enter' && target.tagName !== 'BUTTON') {
      event.preventDefault();
      if (this.revealed()) {
        this.next();
      } else {
        this.submit();
      }
      return;
    }
    const n = Number(event.key);
    if (!this.revealed() && n >= 1 && n <= (this.choices()?.options.length ?? 0)) {
      event.preventDefault();
      this.picked.set(n - 1);
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
    this.picked.set(null);
    this.typed.set('');
    this.verdict.set(null);
    this.answerError.set(null);
    this.playError.set(false);
    this.saveError.set(false);
    this.lastRating = null;
  }

  private readonly nextButton = viewChild<ElementRef<HTMLButtonElement>>('nextButton');

  private focusNext(): void {
    afterNextRender(() => this.nextButton()?.nativeElement.focus(), { injector: this.injector });
  }

  private focusAnswer(): void {
    afterNextRender(() => (this.answer() ?? this.face())?.nativeElement.focus(), { injector: this.injector });
  }

}

/** The example sentence of a card as [before, word, after]; null without a sentence. */
export function exampleParts(card: Pick<DueCard, 'text' | 'lemma' | 'contextSentence'> | null): [string, string, string] | null {
  const sentence = card?.contextSentence?.trim();
  if (!card || !sentence) {
    return null;
  }
  const lower = sentence.toLowerCase();
  for (const w of [card.text, card.lemma]) {
    const at = w ? lower.indexOf(w.trim().toLowerCase()) : -1;
    if (at >= 0) {
      const end = at + w.trim().length;
      return [sentence.slice(0, at), sentence.slice(at, end), sentence.slice(end)];
    }
  }
  return [sentence, '', ''];
}
