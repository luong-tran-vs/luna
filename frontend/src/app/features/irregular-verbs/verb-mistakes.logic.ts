import { VerbMistake, VerbProgress, VerbResult } from '../../core/models/verbs';
import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { spellings } from './irregular-verbs.logic';
import { gradeForm } from './verb-study.logic';

/**
 * "Từ hay sai" (design/de-xuat-dong-tu.png part 6, added 2026-11-10): the verbs a learner keeps
 * typing wrong. The server keeps each verb's last mistakes and the right answers since; these
 * rules read them.
 */

/** A verb is "hay sai" with this many mistakes within MISTAKE_DAYS… */
export const MISTAKE_MIN = 2;
export const MISTAKE_DAYS = 14;
/** …until it is typed right this many times in a row. */
export const CLEAN_RUN = 2;
/** How many verbs one "Luyện" round asks. */
export const PRACTICE_SIZE = 5;

const DAY = 24 * 60 * 60 * 1000;

/** The mistakes of the last MISTAKE_DAYS, oldest first. */
export function recentMistakes(p: VerbProgress | undefined, now: Date): VerbMistake[] {
  const since = now.getTime() - MISTAKE_DAYS * DAY;
  return (p?.mistakes ?? []).filter((m) => new Date(m.at).getTime() >= since);
}

/** Whether a verb is one the learner keeps typing wrong. */
export function isStruggling(p: VerbProgress | undefined, now: Date): boolean {
  return recentMistakes(p, now).length >= MISTAKE_MIN && (p?.cleanRun ?? 0) < CLEAN_RUN;
}

export interface StrugglingVerb {
  verb: IrregularVerb;
  /** The recent mistakes, newest first. */
  mistakes: VerbMistake[];
}

/** The verbs the learner keeps typing wrong, the most mistakes first. */
export function strugglingVerbs(
  progress: ReadonlyMap<string, VerbProgress>,
  now: Date,
  verbs: readonly IrregularVerb[] = IRREGULAR_VERBS,
): StrugglingVerb[] {
  return verbs
    .filter((v) => isStruggling(progress.get(v.base), now))
    .map((v) => ({ verb: v, mistakes: recentMistakes(progress.get(v.base), now).reverse() }))
    .sort(
      (a, b) => b.mistakes.length - a.mistakes.length || a.verb.base.localeCompare(b.verb.base),
    );
}

/** The forms of a typed answer that are wrong, with what was typed. */
export function typedMistakes(r: {
  verb: IrregularVerb;
  past: string;
  participle: string;
}): NonNullable<VerbResult['wrong']> {
  const out: NonNullable<VerbResult['wrong']> = [];
  if (!gradeForm(r.past, r.verb.past)) {
    out.push({ form: 'past', typed: r.past.trim() });
  }
  if (!gradeForm(r.participle, r.verb.participle)) {
    out.push({ form: 'participle', typed: r.participle.trim() });
  }
  return out;
}

/** What is sent for a typed answer: right, and the forms typed wrong. */
export function typedResult(
  r: { verb: IrregularVerb; past: string; participle: string; right: boolean },
  practice = false,
): VerbResult {
  const wrong = r.right ? [] : typedMistakes(r);
  return {
    base: r.verb.base,
    right: r.right,
    ...(wrong.length ? { wrong } : {}),
    ...(practice ? { practice } : {}),
  };
}

/** The right spelling of the form a mistake was on. */
export function rightForm(v: IrregularVerb, m: VerbMistake): string {
  return spellings(m.form === 'past' ? v.past : v.participle)[0];
}

/** A piece of a typed word compared with the right one. */
export interface LetterPart {
  text: string;
  /** same: in both; extra: typed but not in the right word; missing: in the right word only. */
  kind: 'same' | 'extra' | 'missing';
}

/**
 * Where a typed word differs from the right one, letter by letter (longest common subsequence):
 * "choosen" against "chosen" → cho + [o extra] + sen.
 */
export function letterDiff(typed: string, right: string): LetterPart[] {
  const a = typed.toLowerCase();
  const b = right.toLowerCase();
  const n = a.length;
  const m = b.length;
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i][j] = a[i] === b[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1]);
    }
  }
  const out: LetterPart[] = [];
  const push = (text: string, kind: LetterPart['kind']) => {
    const last = out[out.length - 1];
    if (last?.kind === kind) {
      last.text += text;
    } else {
      out.push({ text, kind });
    }
  };
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      push(typed[i], 'same');
      i++;
      j++;
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      push(typed[i++], 'extra');
    } else {
      push(right[j++], 'missing');
    }
  }
  while (i < n) {
    push(typed[i++], 'extra');
  }
  while (j < m) {
    push(right[j++], 'missing');
  }
  return out;
}
