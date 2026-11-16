import { VerbMiss } from '../../../core/models/games';
import { shuffle } from '../../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';
import { isStruggling } from '../../irregular-verbs/verb-mistakes.logic';
import { gradeForm, isDue, isLearned, Progress } from '../../irregular-verbs/verb-study.logic';

/** "Đấu động từ" (FR-013 – FR-015) as pure state. */

/** Lives of a round. */
export const DUEL_LIVES = 3;
/** The most verb misses sent with a result. */
export const MAX_MISSES = 50;
/** The server's highest score of the game. */
export const DUEL_MAX_SCORE = 500;

/**
 * The verbs of a round, in order (FR-015): learned verbs that are due or "hay sai" first, then the
 * other learned ones, then the common verbs not learned yet (so a learner with fewer than 10 learned
 * verbs still gets a full round). Each group is shuffled; no verb appears twice.
 */
export function duelOrder(
  progress: Progress,
  now: Date,
  seed: number,
  verbs: readonly IrregularVerb[] = IRREGULAR_VERBS,
): IrregularVerb[] {
  const first: IrregularVerb[] = [];
  const learned: IrregularVerb[] = [];
  const common: IrregularVerb[] = [];
  for (const v of verbs) {
    const p = progress.get(v.base);
    if (isLearned(p)) {
      (isDue(p, now) || isStruggling(p, now) ? first : learned).push(v);
    } else if (v.common) {
      common.push(v);
    }
  }
  return [...shuffle(first, seed), ...shuffle(learned, seed + 1), ...shuffle(common, seed + 2)];
}

/** The grade of one verb: whether each typed form is right. */
export interface DuelGrade {
  past: boolean;
  participle: boolean;
  typedPast: string;
  typedParticiple: string;
}

/** Grades V2 and V3: any spelling of a form, any case, extra spaces ignored. */
export function gradeDuel(verb: IrregularVerb, typedPast: string, typedParticiple: string): DuelGrade {
  return {
    past: gradeForm(typedPast, verb.past),
    participle: gradeForm(typedParticiple, verb.participle),
    typedPast,
    typedParticiple,
  };
}

export function bothRight(g: DuelGrade): boolean {
  return g.past && g.participle;
}

export interface DuelState {
  queue: readonly IrregularVerb[];
  index: number;
  lives: number;
  streak: number;
  /** Verbs with both forms right. */
  score: number;
  right: number;
  wrong: number;
  /** The verbs answered wrong, each once, in order. */
  missed: IrregularVerb[];
  /** Each wrong form with what was typed (at most MAX_MISSES). */
  misses: VerbMiss[];
  /** The grade of the current verb; null before checking. */
  checked: DuelGrade | null;
}

export function startDuel(queue: readonly IrregularVerb[]): DuelState {
  return {
    queue,
    index: 0,
    lives: DUEL_LIVES,
    streak: 0,
    score: 0,
    right: 0,
    wrong: 0,
    missed: [],
    misses: [],
    checked: null,
  };
}

export function currentVerb(s: DuelState): IrregularVerb | null {
  return s.queue[s.index] ?? null;
}

/**
 * Checks the current verb: both right adds to the score and the streak; a wrong form (one box or
 * both) costs one life and ends the streak. A second check is ignored, as is one after the end.
 */
export function answerDuel(s: DuelState, typedPast: string, typedParticiple: string): DuelState {
  const v = currentVerb(s);
  if (!v || s.checked !== null || duelOver(s)) {
    return s;
  }
  const g = gradeDuel(v, typedPast, typedParticiple);
  if (bothRight(g)) {
    return { ...s, checked: g, score: s.score + 1, streak: s.streak + 1, right: s.right + 1 };
  }
  const misses: VerbMiss[] = [...s.misses];
  if (!g.past) {
    misses.push({ base: v.base, form: 'past', typed: typedPast.trim() });
  }
  if (!g.participle) {
    misses.push({ base: v.base, form: 'participle', typed: typedParticiple.trim() });
  }
  return {
    ...s,
    checked: g,
    lives: s.lives - 1,
    streak: 0,
    wrong: s.wrong + 1,
    missed: s.missed.some((m) => m.base === v.base) ? s.missed : [...s.missed, v],
    misses: misses.slice(0, MAX_MISSES),
  };
}

/** The next verb; when the list runs out, `more` (a new order) is added after it. */
export function nextDuel(s: DuelState, more: readonly IrregularVerb[] = []): DuelState {
  const index = s.index + 1;
  const queue = index >= s.queue.length ? [...s.queue, ...more] : s.queue;
  return { ...s, queue, index, checked: null };
}

/** The round ends when the lives are gone. */
export function duelOver(s: DuelState): boolean {
  return s.lives <= 0;
}

/** "V2 went · V3 gone", as the answer block and the result list show it. */
export function duelAnswer(v: IrregularVerb): string {
  return `V2 ${v.past} · V3 ${v.participle}`;
}
