import { GameWord } from '../../../core/models/games';

/** "Chữ rơi" (FR-016, FR-017, FR-018) as a pure state: the page only animates the drops. */

export const LIVES = 3;
/** Words of one level. */
export const WORDS_PER_LEVEL = 5;
/** The server's highest score for "letters"; a score never goes over it. */
export const MAX_LETTERS_SCORE = 20_000;

/** A random number in [0, 1), injected so tests are deterministic. */
export type Rand = () => number;

const ALPHABET = 'abcdefghijklmnopqrstuvwxyz';

/** A falling letter: x is the left position in % of the free width, times in ms. */
export interface Drop {
  id: number;
  letter: string;
  x: number;
  duration: number;
  delay: number;
}

/** What happened last, for the feedback line and the screen reader. */
export type LetterEvent = 'right' | 'wrong' | 'fell' | 'solved' | null;

export interface LetterState {
  words: readonly GameWord[];
  /** The word being spelt (also the number of words spelt before it). */
  index: number;
  /** How many letters of the word are filled. */
  filled: number;
  lives: number;
  score: number;
  /** Words spelt to the end. */
  spelt: number;
  /** Words where a life was lost, each once, in order. */
  missed: GameWord[];
  drops: Drop[];
  nextId: number;
  /** The word is spelt: the page shows it, then calls nextWord. */
  solved: boolean;
  /** The drop just tapped wrong. */
  wrongId: number | null;
  last: LetterEvent;
  /** The points the last spelt word gave. */
  gained: number;
}

/** Level of the word at index: 5 words a level, from 1. */
export function levelOf(index: number): number {
  return Math.floor(index / WORDS_PER_LEVEL) + 1;
}

/** How long a letter takes to fall: faster each level, never under 2.2 s. */
export function fallDuration(level: number): number {
  return Math.max(2200, 6000 - (level - 1) * 600);
}

/** How many wrong letters fall with the word: one more each level, at most 8. */
export function distractorCount(level: number): number {
  return Math.min(8, level + 1);
}

/** The points of a word spelt at level, plus a bonus for the last word of a level. */
export function wordPoints(level: number, endsLevel: boolean): number {
  return 100 * level + (endsLevel ? 250 * level : 0);
}

/** Letters that may fall as distractors: none still missing from the word. */
export function distractorLetters(word: string, filled: number): string[] {
  const missing = new Set(word.slice(filled));
  return [...ALPHABET].filter((c) => !missing.has(c));
}

function pick<T>(items: readonly T[], rand: Rand): T {
  return items[Math.min(items.length - 1, Math.floor(rand() * items.length))];
}

function shuffle<T>(items: readonly T[], rand: Rand): T[] {
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.min(i, Math.floor(rand() * (i + 1)));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

/**
 * The lanes letters fall in, as left positions in % of the width left once the letter is placed:
 * five lanes of a 48px letter fit a 360px phone side by side, so letters in different lanes never
 * cover each other.
 */
export const LANES: readonly number[] = [0, 25, 50, 75, 100];

/** A lane at random. */
function randomX(rand: Rand): number {
  return LANES[Math.min(LANES.length - 1, Math.floor(Math.max(0, rand()) * LANES.length))];
}

/** A lane at random, not the one of the letter before nor a lane next to it (they fall close). */
function laneAfter(prev: number | null, rand: Rand): number {
  const free = LANES.filter((x) => prev === null || Math.abs(x - prev) > 25);
  return free[Math.min(free.length - 1, Math.floor(Math.max(0, rand()) * free.length))];
}

/**
 * The drops of a word at level: every letter of the word (repeats included) and the level's
 * distractors, shuffled, at random x, one after another.
 */
export function planWord(word: string, level: number, rand: Rand, firstId: number): Drop[] {
  const pool = distractorLetters(word, 0);
  const letters = [...word];
  for (let i = 0; i < distractorCount(level); i++) {
    letters.push(pick(pool, rand));
  }
  const duration = fallDuration(level);
  const gap = Math.round((duration * 0.8) / letters.length);
  let prev: number | null = null;
  return shuffle(letters, rand).map((letter, i) => {
    const x = laneAfter(prev, rand);
    prev = x;
    return { id: firstId + i, letter, x, duration, delay: i * gap + Math.round(rand() * gap * 0.5) };
  });
}

function current(s: LetterState): GameWord | null {
  return s.words[s.index] ?? null;
}

function withWord(s: LetterState, rand: Rand): LetterState {
  const w = current(s);
  if (!w || s.lives <= 0) {
    return { ...s, drops: [] };
  }
  const drops = planWord(w.lemma, levelOf(s.index), rand, s.nextId);
  return { ...s, drops, nextId: s.nextId + drops.length };
}

function miss(s: LetterState): GameWord[] {
  const w = current(s);
  return !w || s.missed.some((m) => m.lemma === w.lemma) ? s.missed : [...s.missed, w];
}

export function startLetters(words: readonly GameWord[], rand: Rand): LetterState {
  return withWord(
    {
      words,
      index: 0,
      filled: 0,
      lives: LIVES,
      score: 0,
      spelt: 0,
      missed: [],
      drops: [],
      nextId: 1,
      solved: false,
      wrongId: null,
      last: null,
      gained: 0,
    },
    rand,
  );
}

/** The word being spelt; null once the round is over. */
export function currentWord(s: LetterState): GameWord | null {
  return current(s);
}

/** The letter to tap next; null when the word is spelt. */
export function neededLetter(s: LetterState): string | null {
  return current(s)?.lemma[s.filled] ?? null;
}

/** The level being played. */
export function currentLevel(s: LetterState): number {
  return levelOf(Math.min(s.index, Math.max(0, s.words.length - 1)));
}

/** Whether the round is over: no life left or every word spelt. */
export function lettersFinished(s: LetterState): boolean {
  return s.lives <= 0 || s.index >= s.words.length;
}

/**
 * Taps drop id: the needed letter fills the next box and leaves (the last one spells the word and
 * scores it); any other letter costs a life and fills nothing.
 */
export function tapLetter(s: LetterState, id: number): LetterState {
  const drop = s.drops.find((d) => d.id === id);
  const need = neededLetter(s);
  if (!drop || !need || s.solved || lettersFinished(s)) {
    return s;
  }
  if (drop.letter !== need) {
    return { ...s, lives: s.lives - 1, missed: miss(s), wrongId: id, last: 'wrong', gained: 0 };
  }
  const filled = s.filled + 1;
  const drops = s.drops.filter((d) => d.id !== id);
  const word = current(s)!;
  if (filled < word.lemma.length) {
    return { ...s, filled, drops, wrongId: null, last: 'right', gained: 0 };
  }
  const level = levelOf(s.index);
  const gained = wordPoints(level, (s.index + 1) % WORDS_PER_LEVEL === 0);
  return {
    ...s,
    filled,
    drops: [],
    solved: true,
    spelt: s.spelt + 1,
    score: Math.min(MAX_LETTERS_SCORE, s.score + gained),
    wrongId: null,
    last: 'solved',
    gained,
  };
}

/**
 * Drop id fell off the bottom: it falls again from the top at a new x; when it was the letter
 * needed now, that costs a life.
 */
export function dropFell(s: LetterState, id: number, rand: Rand): LetterState {
  const drop = s.drops.find((d) => d.id === id);
  if (!drop || s.solved || lettersFinished(s)) {
    return s;
  }
  const again: Drop = { ...drop, id: s.nextId, x: randomX(rand), delay: Math.round(rand() * 400) };
  const drops = s.drops.map((d) => (d.id === id ? again : d));
  const base = { ...s, drops, nextId: s.nextId + 1, wrongId: s.wrongId === id ? null : s.wrongId };
  if (drop.letter !== neededLetter(s)) {
    return base;
  }
  return { ...base, lives: s.lives - 1, missed: miss(s), last: 'fell', gained: 0 };
}

/** After a spelt word: the next word, faster with more distractors when a level starts. */
export function nextWord(s: LetterState, rand: Rand): LetterState {
  return withWord(
    { ...s, index: s.index + 1, filled: 0, solved: false, wrongId: null, last: null, gained: 0 },
    rand,
  );
}
