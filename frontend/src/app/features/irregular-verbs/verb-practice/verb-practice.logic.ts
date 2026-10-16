import { VerbProgress } from '../../../core/models/verbs';
import { normalizeAnswer } from '../../../shared/utils/answer-match';
import { shuffle } from '../../lesson/lesson-detail/practice-logic';
import { IrregularVerb } from '../irregular-verbs.data';
import { patternOf, spellings, VerbPattern } from '../irregular-verbs.logic';

/** Pure helpers of the verb practice: which verbs a round asks and how an answer is graded. */

/** Verbs asked in one round. */
export const ROUND_SIZE = 10;

/** Where a round takes its verbs from. */
export type VerbSet = 'common' | 'all' | VerbPattern;

export function inSet(verbs: readonly IrregularVerb[], set: VerbSet): IrregularVerb[] {
  if (set === 'common') {
    return verbs.filter((v) => v.common);
  }
  return set === 'all' ? [...verbs] : verbs.filter((v) => patternOf(v) === set);
}

export function isKnown(p: VerbProgress | undefined): boolean {
  return !!p && p.level > 0;
}

/** A known verb whose wait is over. */
export function isDue(p: VerbProgress | undefined, now: Date): boolean {
  return isKnown(p) && !!p!.dueAt && new Date(p!.dueAt) <= now;
}

export interface SetSummary {
  total: number;
  known: number;
  due: number;
}

export function summarize(
  verbs: readonly IrregularVerb[],
  progress: ReadonlyMap<string, VerbProgress>,
  now: Date,
): SetSummary {
  let known = 0;
  let due = 0;
  for (const v of verbs) {
    const p = progress.get(v.base);
    known += isKnown(p) ? 1 : 0;
    due += isDue(p, now) ? 1 : 0;
  }
  return { total: verbs.length, known, due };
}

/**
 * The verbs of a round: the known ones that are due first (the longest waiting first), then the
 * ones not known yet (those already got wrong before the new ones), up to ROUND_SIZE, in a shuffled
 * order. When every verb is known and none is due, it reviews the ones due soonest instead.
 */
export function pickRound(
  verbs: readonly IrregularVerb[],
  progress: ReadonlyMap<string, VerbProgress>,
  now: Date,
  seed: number,
  size = ROUND_SIZE,
): IrregularVerb[] {
  const dueAt = (v: IrregularVerb) => new Date(progress.get(v.base)?.dueAt ?? 0).getTime();
  const due = verbs.filter((v) => isDue(progress.get(v.base), now)).sort((a, b) => dueAt(a) - dueAt(b));
  const fresh = verbs.filter((v) => !isKnown(progress.get(v.base)));
  const missed = fresh.filter((v) => (progress.get(v.base)?.wrong ?? 0) > 0);
  const unseen = shuffle(
    fresh.filter((v) => !missed.includes(v)),
    seed,
  );
  let round = [...due, ...missed, ...unseen].slice(0, size);
  if (round.length === 0) {
    round = [...verbs].sort((a, b) => dueAt(a) - dueAt(b)).slice(0, size);
  }
  return shuffle(round, seed + 1);
}

/** Whether a typed form is one of the spellings of the form ("burnt/burned": either one). */
export function gradeForm(typed: string, form: string): boolean {
  const t = normalizeAnswer(typed);
  return t !== '' && (t === normalizeAnswer(form) || spellings(form).some((s) => normalizeAnswer(s) === t));
}
