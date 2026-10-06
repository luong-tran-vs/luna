export type CardSource = 'ai' | 'dictionary' | 'manual';

export type CardState = 'new' | 'learning' | 'review' | 'relearning';

/** A notebook card with its review schedule. `lessonId` is null for cards added by hand. */
export interface Card {
  id: string;
  text: string;
  lemma: string;
  ipa: string;
  meaningVi: string;
  contextSentence: string;
  lessonId: string | null;
  source: CardSource;
  createdAt: string;
  due: string;
  reps: number;
  state: CardState;
}

/** Body of POST /api/vocab/cards; `lessonId` is omitted for manual cards. */
export interface CardInput {
  text: string;
  lemma: string;
  ipa: string;
  meaningVi: string;
  contextSentence: string;
  lessonId?: string;
  source: CardSource;
}

/** Fields a learner can edit (PATCH /api/vocab/cards/{id}). */
export type CardDetails = Partial<Pick<Card, 'meaningVi' | 'ipa' | 'contextSentence'>>;

export interface WordRef {
  lemma: string;
  text: string;
}

/** A card in the notebook list; `day` is YYYY-MM-DD in the learner's timezone. */
export interface DayCard extends Card {
  day: string;
}

export interface CardPage {
  cards: DayCard[];
  hasMore: boolean;
  today: string;
  yesterday: string;
}

export interface CardQuery {
  q?: string;
  /** A lesson id, or 'manual' for cards added by hand. */
  lessonId?: string;
  page?: number;
}

export interface LessonCount {
  id: string;
  title: string;
  count: number;
}

export interface LessonCounts {
  lessons: LessonCount[];
  manualCount: number;
}

export type ReviewMode = 'flip' | 'listen';

/** 1 Again, 2 Hard, 3 Good, 4 Easy. */
export type Rating = 1 | 2 | 3 | 4;

/** Seconds until the next review for each rating. */
export interface Intervals {
  again: number;
  hard: number;
  good: number;
  easy: number;
}

export interface DueCard extends Card {
  intervals: Intervals;
}

export interface DueList {
  cards: DueCard[];
  total: number;
  nextDue: string | null;
}

/** Where a review happens: the review step of today's lesson (daily limit) or free review. */
export type ReviewContext = 'daily' | 'free';

export interface ReviewInput {
  rating: Rating;
  mode: ReviewMode;
  reps: number;
  context: ReviewContext;
}

export interface ReviewSummary {
  reviewed: number;
  counts: Record<Rating, number>;
}

/** One annotated word or phrase of a lesson (GET /api/lessons/{id}/vocabulary). */
export interface VocabItem {
  lemma: string;
  text: string;
  meaningVi: string;
  ipa: string;
  sentenceIndex: number;
  sentence: string;
  /** F23: the word's AI-drawn picture; '' or missing when it has none. */
  imageUrl?: string;
}

export interface LessonVocabulary {
  available: boolean;
  items: VocabItem[];
}

export interface BulkResult {
  added: number;
  cards: Card[];
}

/** POST /api/vocab/practice-misses: words got wrong in the practice, now due for review. */
export interface PracticeMisses {
  /** Cards created. */
  added: number;
  /** Saved cards brought forward to now. */
  rescheduled: number;
}
