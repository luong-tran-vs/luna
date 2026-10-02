import { Level } from '../../../core/models/lesson';
import { FillBlank, PracticeView } from '../../../core/models/practice';

/** Pure helpers of the lesson practice (F17): shuffling, checking and the summary. */

/** A new random seed for shuffling the word banks (a new one on every retry). */
export function newSeed(): number {
  return Math.floor(Math.random() * 0x7fffffff);
}

/** Small seeded PRNG (mulberry32): the same seed always gives the same numbers in [0, 1). */
function random(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** Returns a shuffled copy of `items` (Fisher–Yates); the same seed gives the same order. */
export function shuffle<T>(items: readonly T[], seed: number): T[] {
  const next = random(seed);
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(next() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

/** "Cơ bản" for A1–A2, "Trung cấp" for B1–B2, "Nâng cao" for C1–C2. */
export function levelLabel(level: Level): string {
  switch (level) {
    case 'A1':
    case 'A2':
      return 'Cơ bản';
    case 'B1':
    case 'B2':
      return 'Trung cấp';
    default:
      return 'Nâng cao';
  }
}

export function hasExamples(p: PracticeView | null): boolean {
  return (p?.examples.length ?? 0) > 0;
}

export function hasDialogue(p: PracticeView | null): boolean {
  return (p?.dialogue?.turns.length ?? 0) > 0;
}

export function hasFill(p: PracticeView | null): boolean {
  return (p?.fill?.blanks.length ?? 0) > 0;
}

export function hasTranslations(p: PracticeView | null): boolean {
  return (p?.translations.length ?? 0) > 0;
}

/** Case and surrounding spaces do not matter when checking a blank. */
function normalize(text: string): string {
  return text.trim().toLowerCase();
}

export interface Score {
  correct: number;
  total: number;
}

export interface FillCheck extends Score {
  /** Per blank: right or wrong (an empty blank is wrong). */
  results: boolean[];
}

/** Checks the words put in the blanks against their answers. */
export function checkFill(
  answers: readonly (string | null)[],
  blanks: readonly FillBlank[],
): FillCheck {
  const results = blanks.map((b, i) => {
    const a = answers[i];
    return a !== null && a !== undefined && normalize(a) === normalize(b.answer);
  });
  return { results, correct: results.filter(Boolean).length, total: blanks.length };
}

/** The first empty blank after `from` (wrapping around), or -1 when all are filled. */
export function nextEmptyBlank(answers: readonly unknown[], from = -1): number {
  const n = answers.length;
  for (let k = 1; k <= n; k++) {
    const i = (((from + k) % n) + n) % n;
    if (answers[i] === null) {
      return i;
    }
  }
  return -1;
}

/**
 * A built sentence is right when its tiles match the answer in order. Tiles are compared by text,
 * so two tiles with the same text can replace each other.
 */
export function checkTranslation(chosen: readonly string[], answer: readonly string[]): boolean {
  return chosen.length === answer.length && chosen.every((t, i) => t === answer[i]);
}

export interface Summary {
  /** Null when the lesson has no fill-in step. */
  fill: Score | null;
  /** Null when the lesson has no translations. */
  translate: Score | null;
}

/**
 * Totals for the summary. A fill step that was not checked counts every blank wrong, and a
 * sentence that was not checked (null) counts wrong.
 */
export function summary(
  fillTotal: number,
  fillResult: Score | null,
  translateResults: readonly (boolean | null)[],
): Summary {
  return {
    fill: fillTotal > 0 ? { correct: fillResult?.correct ?? 0, total: fillTotal } : null,
    translate:
      translateResults.length > 0
        ? {
            correct: translateResults.filter((r) => r === true).length,
            total: translateResults.length,
          }
        : null,
  };
}

/** Seconds as mm:ss (e.g. 00:07, 01:12). */
export function clock(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds));
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(Math.floor(s / 60))}:${pad(s % 60)}`;
}
