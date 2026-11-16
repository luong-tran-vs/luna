import { DailyAnswer, DailyKind, DailyQuestion, SpotQuestion, VerbMiss } from '../../../core/models/games';
import { IRREGULAR_VERBS, IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';

/** Pure helpers of the daily challenge (FR-027 … FR-031). */

/** The questions of a day. */
export const DAILY_COUNT = 10;
/** Each question has its own 20 seconds; no clock for the whole round, no lives. */
export const QUESTION_MS = 20_000;
/** A right answer: 100 + 10 per whole second left, at most 300. */
export const MAX_QUESTION_SCORE = 300;

/** The points of an answer given with msLeft of the question's 20 seconds; a wrong one gives 0. */
export function questionScore(right: boolean, msLeft: number): number {
  if (!right) {
    return 0;
  }
  const seconds = Math.floor(Math.max(0, Math.min(msLeft, QUESTION_MS)) / 1000);
  return Math.min(MAX_QUESTION_SCORE, 100 + 10 * seconds);
}

/** A round of the challenge as a pure state. */
export interface DailyState {
  /** One entry per question answered: whether it was right. */
  answers: boolean[];
  score: number;
  /** Lemmas of the picture and spelling questions missed, each once. */
  wrongWords: string[];
  /** Verb forms typed wrong. */
  wrongVerbs: VerbMiss[];
}

export function startDaily(): DailyState {
  return { answers: [], score: 0, wrongWords: [], wrongVerbs: [] };
}

/** What a wrong answer brings back to the review queue. */
export interface DailyMiss {
  word?: string;
  verbs?: VerbMiss[];
}

/** Records the answer of the current question, given with msLeft. */
export function recordAnswer(s: DailyState, right: boolean, msLeft: number, miss: DailyMiss = {}): DailyState {
  if (s.answers.length >= DAILY_COUNT) {
    return s;
  }
  const word = !right && miss.word && !s.wrongWords.includes(miss.word) ? [miss.word] : [];
  return {
    answers: [...s.answers, right],
    score: s.score + questionScore(right, msLeft),
    wrongWords: [...s.wrongWords, ...word],
    wrongVerbs: right ? s.wrongVerbs : [...s.wrongVerbs, ...(miss.verbs ?? [])],
  };
}

/** The time ran out: the question is wrong. */
export function recordTimeout(s: DailyState, miss: DailyMiss = {}): DailyState {
  return recordAnswer(s, false, 0, miss);
}

/** The answers as the API takes them: the kind of each question and whether it was right. */
export function dailyAnswers(questions: readonly DailyQuestion[], answers: readonly boolean[]): DailyAnswer[] {
  return answers.map((right, i) => ({ kind: (questions[i]?.kind ?? 'picture') as DailyKind, right }));
}

// --- seeded randomness ---

/** A small seeded generator (mulberry32): the same seed gives the same numbers. */
export function seededRandom(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** The generator of question index of the day's seed. */
function questionRandom(seed: number, index: number): () => number {
  return seededRandom((seed ^ Math.imul(index + 1, 0x9e3779b1)) >>> 0);
}

function shuffled<T>(list: readonly T[], rnd: () => number): T[] {
  const out = [...list];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

// --- verbs ---

const COMMON_VERBS = IRREGULAR_VERBS.filter((v) => v.common);

function findVerb(base: string): IrregularVerb | undefined {
  const b = base.trim().toLowerCase();
  return b ? IRREGULAR_VERBS.find((v) => v.base === b) : undefined;
}

/**
 * The verb of each question (null for the other kinds). A question with a base asks that verb; an
 * empty base (or one not in the list) gets a common verb picked from the seed and the question's
 * index, never one already asked that day.
 */
export function resolveVerbs(questions: readonly DailyQuestion[], seed: number): (IrregularVerb | null)[] {
  const used = new Set<string>();
  const given = questions.map((q) => (q.kind === 'verb' ? findVerb(q.verb?.base ?? '') : undefined));
  for (const v of given) {
    if (v) {
      used.add(v.base);
    }
  }
  return questions.map((q, i) => {
    if (q.kind !== 'verb') {
      return null;
    }
    const v = given[i];
    if (v) {
      return v;
    }
    const pool = COMMON_VERBS.filter((c) => !used.has(c.base));
    const list = pool.length ? pool : COMMON_VERBS;
    const picked = list[Math.floor(questionRandom(seed, i)() * list.length)];
    used.add(picked.base);
    return picked;
  });
}

// --- "Ghép chữ" ---

export interface LetterTile {
  id: number;
  ch: string;
}

const ALPHABET = 'abcdefghijklmnopqrstuvwxyz';

/**
 * The tiles of a spelling question: the word's letters with 2–3 letters not in the word, shuffled
 * from the seed and the question's index.
 */
export function letterTiles(lemma: string, seed: number, index: number): LetterTile[] {
  const word = lemma.toLowerCase();
  const rnd = questionRandom(seed, index);
  const extraCount = 2 + Math.floor(rnd() * 2);
  const others = shuffled(
    ALPHABET.split('').filter((c) => !word.includes(c)),
    rnd,
  ).slice(0, extraCount);
  const letters = [...word.split(''), ...others];
  let order = shuffled(letters, rnd);
  // Never hand the word over already spelled.
  if (order.slice(0, word.length).join('') === word) {
    order = [...order.slice(1), order[0]];
  }
  return order.map((ch, id) => ({ id, ch }));
}

/** Whether tile is the next letter after the ones already placed. */
export function rightTile(lemma: string, placed: number, tile: LetterTile): boolean {
  return lemma.toLowerCase()[placed] === tile.ch;
}

// --- grammar ---

/** Whether token i is inside the wrong part (always false for a sentence without a mistake). */
export function hitsWrong(q: SpotQuestion, i: number): boolean {
  return q.wrong !== null && i >= q.wrong.from && i < q.wrong.to;
}

/** "Không có lỗi" is right only when the sentence has no mistake. */
export function noErrorRight(q: SpotQuestion): boolean {
  return q.wrong === null;
}

/** The sentence with the right words in place of the wrong part. */
export function fixedSentence(q: SpotQuestion): string {
  const tokens = q.wrong
    ? [...q.tokens.slice(0, q.wrong.from), q.options[q.answer] ?? '', ...q.tokens.slice(q.wrong.to)]
    : q.tokens;
  return tokens
    .filter((t) => t !== '')
    .join(' ')
    .replace(/\s+([.,!?;:])/g, '$1');
}

// --- dates ---

const WEEKDAYS = ['Chủ Nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy'];
export const WEEK_LABELS = ['T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'CN'];

function parts(day: string): [number, number, number] {
  const [y, m, d] = day.split('-').map(Number);
  return [y || 1970, m || 1, d || 1];
}

/** "13/11" for "2026-11-13". */
export function shortDate(day: string): string {
  const [, m, d] = parts(day);
  return `${String(d).padStart(2, '0')}/${String(m).padStart(2, '0')}`;
}

/** 0 (Sunday) … 6 (Saturday) of "2026-11-13". */
function weekday(day: string): number {
  const [y, m, d] = parts(day);
  return new Date(Date.UTC(y, m - 1, d)).getUTCDay();
}

/** "Thứ Sáu, 13/11". */
export function dayLabel(day: string): string {
  return `${WEEKDAYS[weekday(day)]}, ${shortDate(day)}`;
}

/** Where the day falls in a week starting on Monday: 0 (Monday) … 6 (Sunday). */
export function mondayIndex(day: string): number {
  return (weekday(day) + 6) % 7;
}

export interface WeekCell {
  label: string;
  played: boolean;
  today: boolean;
  /** For the screen reader: "Thứ Hai: đã chơi". */
  name: string;
}

const LONG_WEEKDAYS = ['Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy', 'Chủ Nhật'];

/** The row Monday → Sunday of the start screen. */
export function weekCells(week: readonly boolean[], day: string): WeekCell[] {
  const today = mondayIndex(day);
  return WEEK_LABELS.map((label, i) => {
    const played = !!week[i];
    const state = played ? 'đã chơi' : i === today ? 'hôm nay, chưa chơi' : i < today ? 'chưa chơi' : 'chưa tới';
    return { label, played, today: i === today, name: `${LONG_WEEKDAYS[i]}: ${state}` };
  });
}
