import { SpotQuestion } from '../../../core/models/games';

/** Lives of a round (FR-026). */
export const SPOT_LIVES = 3;
/** The time of one sentence, ms (FR-026). */
export const SPOT_MS = 20_000;
/** The server's highest score of "Bắn lỗi" (MaxScore grammar). */
export const SPOT_MAX_SCORE = 5000;

/**
 * Where the current sentence is: `ask` waits for a tap or "Không có lỗi"; `fix` shows the three
 * fixes of the mistake tapped; `shown` tells the outcome and the rule until "Tiếp".
 */
export type SpotStage = 'ask' | 'fix' | 'shown';

/**
 * How a sentence ended: `fixed` (the mistake fixed), `clean` (a right sentence called right),
 * `bad-fix` (the wrong fix chosen), `bad-token` (a right word tapped), `missed` ("Không có lỗi" on a
 * sentence with a mistake), `false-alarm` (a word tapped in a right sentence), `timeout`.
 */
export type SpotOutcome =
  | 'fixed'
  | 'clean'
  | 'bad-fix'
  | 'bad-token'
  | 'missed'
  | 'false-alarm'
  | 'timeout';

/** A round of "Bắn lỗi ngữ pháp" (FR-023..FR-026), as a pure state. */
export interface SpotState {
  questions: readonly SpotQuestion[];
  index: number;
  lives: number;
  score: number;
  right: number;
  wrong: number;
  /** The sentences got wrong, in order. */
  missed: SpotQuestion[];
  stage: SpotStage;
  outcome: SpotOutcome | null;
  /** The token tapped in the current sentence; null before a tap. */
  tapped: number | null;
  /** The fix chosen; null before choosing. */
  chosen: number | null;
  /** The points the current sentence gave. */
  gained: number;
}

export function startSpot(questions: readonly SpotQuestion[]): SpotState {
  return {
    questions,
    index: 0,
    lives: SPOT_LIVES,
    score: 0,
    right: 0,
    wrong: 0,
    missed: [],
    stage: 'ask',
    outcome: null,
    tapped: null,
    chosen: null,
    gained: 0,
  };
}

/** The sentence being asked; null once the round is over. */
export function currentSpot(s: SpotState): SpotQuestion | null {
  return spotFinished(s) ? null : (s.questions[s.index] ?? null);
}

/** Whether every sentence was asked or no life is left. */
export function spotFinished(s: SpotState): boolean {
  return s.index >= s.questions.length || s.lives <= 0;
}

/** Whether token i lies in the sentence's mistake. */
export function inMistake(q: SpotQuestion, i: number): boolean {
  return !!q.wrong && i >= q.wrong.from && i < q.wrong.to;
}

/** The points of a right sentence answered with msLeft to spare: 100 + 10 × whole seconds left. */
export function spotPoints(msLeft: number): number {
  return 100 + 10 * Math.max(0, Math.ceil(msLeft / 1000));
}

function settle(s: SpotState, outcome: SpotOutcome, msLeft: number, extra: Partial<SpotState> = {}): SpotState {
  const q = s.questions[s.index];
  const right = outcome === 'fixed' || outcome === 'clean';
  const gained = right ? Math.min(spotPoints(msLeft), SPOT_MAX_SCORE - s.score) : 0;
  return {
    ...s,
    ...extra,
    stage: 'shown',
    outcome,
    gained,
    score: s.score + gained,
    right: s.right + (right ? 1 : 0),
    wrong: s.wrong + (right ? 0 : 1),
    lives: right ? s.lives : s.lives - 1,
    missed: right ? s.missed : [...s.missed, q],
  };
}

/**
 * Taps token i: a token of the mistake opens its fixes; any other token loses a life and reveals
 * the mistake (or, in a right sentence, says it was right).
 */
export function tapToken(s: SpotState, i: number, msLeft: number): SpotState {
  const q = currentSpot(s);
  if (!q || s.stage !== 'ask') {
    return s;
  }
  if (inMistake(q, i)) {
    return { ...s, stage: 'fix', tapped: i };
  }
  return settle(s, q.wrong ? 'bad-token' : 'false-alarm', msLeft, { tapped: i });
}

/** Chooses fix i of the mistake tapped: the right one scores, another loses a life. */
export function chooseFix(s: SpotState, i: number, msLeft: number): SpotState {
  const q = currentSpot(s);
  if (!q || s.stage !== 'fix') {
    return s;
  }
  return settle(s, i === q.answer ? 'fixed' : 'bad-fix', msLeft, { chosen: i });
}

/** "Không có lỗi": right for a right sentence, a lost life for one with a mistake. */
export function callClean(s: SpotState, msLeft: number): SpotState {
  const q = currentSpot(s);
  if (!q || s.stage !== 'ask') {
    return s;
  }
  return settle(s, q.wrong ? 'missed' : 'clean', msLeft);
}

/** The sentence's time ran out before it was answered: wrong. */
export function timeOut(s: SpotState): SpotState {
  if (!currentSpot(s) || s.stage === 'shown') {
    return s;
  }
  return settle(s, 'timeout', 0);
}

/** Goes on to the next sentence. */
export function nextSpot(s: SpotState): SpotState {
  return { ...s, index: s.index + 1, stage: 'ask', outcome: null, tapped: null, chosen: null, gained: 0 };
}

/** Whether the outcome counts as right. */
export function outcomeRight(o: SpotOutcome | null): boolean {
  return o === 'fixed' || o === 'clean';
}

const CLOSERS = new Set(['.', ',', '!', '?', ';', ':', ')', ']', '}', '”', '’', '…']);
const OPENERS = new Set(['(', '[', '{', '“', '‘']);

/** Whether a token is a punctuation mark the tokenizer split off (shown as text, not a button). */
export function isMark(token: string): boolean {
  return [...token].length === 1 && /[\p{P}–—]/u.test(token) && token !== "'" && token !== '’';
}

/**
 * Whether a space goes before token i: none before closing marks, none after opening marks, and a
 * straight quote " opens and closes in turn.
 */
export function spaceBefore(tokens: readonly string[], i: number): boolean {
  if (i === 0) {
    return false;
  }
  const t = tokens[i];
  const prev = tokens[i - 1];
  const quotesBefore = (end: number) => tokens.slice(0, end).filter((x) => x === '"').length;
  if (CLOSERS.has(t) || (t === '"' && quotesBefore(i) % 2 === 1)) {
    return false;
  }
  if (OPENERS.has(prev) || (prev === '"' && quotesBefore(i - 1) % 2 === 0)) {
    return false;
  }
  return true;
}

/** Tokens joined back into a sentence. */
export function joinTokens(tokens: readonly string[]): string {
  return tokens.map((t, i) => (spaceBefore(tokens, i) ? ' ' : '') + t).join('');
}

/** The words of the mistake ("go", "she aren't"); "" for a right sentence. */
export function mistakeText(q: SpotQuestion): string {
  return q.wrong ? joinTokens(q.tokens.slice(q.wrong.from, q.wrong.to)) : '';
}

/** The right fix; "" for a right sentence. */
export function rightFix(q: SpotQuestion): string {
  return q.wrong ? (q.options[q.answer] ?? '') : '';
}

/** The sentence with its mistake fixed (the sentence itself when right). */
export function correctedSentence(q: SpotQuestion): string {
  if (!q.wrong) {
    return joinTokens(q.tokens);
  }
  const fixed = rightFix(q).split(/\s+/).filter(Boolean);
  return joinTokens([...q.tokens.slice(0, q.wrong.from), ...fixed, ...q.tokens.slice(q.wrong.to)]);
}
