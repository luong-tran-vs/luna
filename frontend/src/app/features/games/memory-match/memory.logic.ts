import { GameWord } from '../../../core/models/games';

/** The best score of a round (the server's MaxScore for memory). */
export const MEMORY_MAX_SCORE = 10_000;
/** Points lost for each move over the minimum (one per pair). */
const MOVE_COST = 300;
/** Points lost for each second. */
const SECOND_COST = 20;

/** What a card shows face up: the English word, the picture, or the Vietnamese meaning. */
export type CardFace = 'word' | 'image' | 'meaning';

export interface MemoryCard {
  /** Which pair the card belongs to (the index in the round's pairs). */
  pair: number;
  face: CardFace;
  /** The word for a word card, the meaning for a meaning card, the lemma for a picture card. */
  text: string;
  /** The picture's path ("" unless face is 'image'). */
  imageUrl: string;
}

/** A round of "Lật thẻ" (FR-019, FR-021), as a pure state. */
export interface MemoryState {
  cards: readonly MemoryCard[];
  /** The cards face up and not matched (0, 1 or 2), in the order they were flipped. */
  flipped: readonly number[];
  /** Whether each card was matched. */
  matched: readonly boolean[];
  /** How many moves were made (a move is two cards flipped). */
  moves: number;
  /** Whether the two flipped cards do not match and wait to be turned back. */
  mismatch: boolean;
  /** The pair matched by the last move; null otherwise. */
  justMatched: number | null;
}

/**
 * The 12 cards of the pairs, shuffled: for each pair one card with the English word and one with
 * the picture, or the Vietnamese meaning when the word has no picture (FR-019).
 */
export function dealCards(pairs: readonly GameWord[], random: () => number = Math.random): MemoryCard[] {
  const cards: MemoryCard[] = [];
  pairs.forEach((p, pair) => {
    cards.push({ pair, face: 'word', text: p.lemma, imageUrl: '' });
    cards.push(
      p.imageUrl
        ? { pair, face: 'image', text: p.lemma, imageUrl: p.imageUrl }
        : { pair, face: 'meaning', text: p.meaningVi, imageUrl: '' },
    );
  });
  for (let i = cards.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [cards[i], cards[j]] = [cards[j], cards[i]];
  }
  return cards;
}

export function startMemory(cards: readonly MemoryCard[]): MemoryState {
  return { cards, flipped: [], matched: cards.map(() => false), moves: 0, mismatch: false, justMatched: null };
}

/** Whether card i is face up (flipped or matched). */
export function isUp(s: MemoryState, i: number): boolean {
  return s.matched[i] || s.flipped.includes(i);
}

/**
 * Flips card i. The second card of a move counts the move: the same pair stays matched, another
 * leaves both up as a mismatch. Flipping while a mismatch is up turns it back at once and flips
 * the new card (no waiting). A matched card, or one already up, changes nothing.
 */
export function flipCard(s: MemoryState, i: number): MemoryState {
  if (i < 0 || i >= s.cards.length || s.matched[i] || memoryFinished(s)) {
    return s;
  }
  // A tap while a mismatch shows turns it down at once and flips the card tapped, even one of the
  // two: it starts a new move.
  if (s.mismatch) {
    return { ...s, flipped: [i], mismatch: false, justMatched: null };
  }
  if (isUp(s, i)) {
    return s;
  }
  if (s.flipped.length === 0) {
    return { ...s, flipped: [i], mismatch: false, justMatched: null };
  }
  const first = s.flipped[0];
  const moves = s.moves + 1;
  if (s.cards[first].pair === s.cards[i].pair) {
    const matched = s.matched.map((m, k) => m || k === first || k === i);
    return { ...s, flipped: [], matched, moves, mismatch: false, justMatched: s.cards[i].pair };
  }
  return { ...s, flipped: [first, i], moves, mismatch: true, justMatched: null };
}

/** Turns a mismatched pair back down (after its second on screen). */
export function hideMismatch(s: MemoryState): MemoryState {
  return s.mismatch ? { ...s, flipped: [], mismatch: false } : s;
}

/** How many pairs were matched. */
export function pairsFound(s: MemoryState): number {
  return s.matched.filter(Boolean).length / 2;
}

/** Whether every pair was matched. */
export function memoryFinished(s: MemoryState): boolean {
  return s.cards.length > 0 && s.matched.every(Boolean);
}

/**
 * The score of a finished round (FR-021): 10 000, minus 300 for each move over one per pair and 20
 * for each whole second, never below 0 nor above 10 000.
 */
export function memoryScore(moves: number, ms: number, pairs = 6): number {
  const extra = Math.max(0, moves - pairs);
  const seconds = Math.floor(Math.max(0, ms) / 1000);
  const score = MEMORY_MAX_SCORE - MOVE_COST * extra - SECOND_COST * seconds;
  return Math.min(MEMORY_MAX_SCORE, Math.max(0, Math.round(score)));
}

/** "0:12" for 12 400 ms elapsed (whole seconds, rounded down). */
export function elapsedLabel(ms: number): string {
  const s = Math.floor(Math.max(0, ms) / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
