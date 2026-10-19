import { VerbProgress } from '../../core/models/verbs';
import { normalizeAnswer } from '../../shared/utils/answer-match';
import { shuffle } from '../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { spellings } from './irregular-verbs.logic';
import { VerbLesson } from './verb-lessons.data';

/** Pure helpers of learning the verbs (lessons) and reviewing the learned ones. */

/** Verbs asked in one review round. */
export const REVIEW_SIZE = 10;

const DAY = 24 * 60 * 60 * 1000;

export type Progress = ReadonlyMap<string, VerbProgress>;

export function isLearned(p: VerbProgress | undefined): boolean {
  return !!p?.learnedAt;
}

/** A learned verb whose wait is over. */
export function isDue(p: VerbProgress | undefined, now: Date): boolean {
  return isLearned(p) && !!p!.dueAt && new Date(p!.dueAt) <= now;
}

/** The verbs of a lesson, in its order. */
export function lessonVerbs(lesson: VerbLesson, verbs: readonly IrregularVerb[] = IRREGULAR_VERBS): IrregularVerb[] {
  return lesson.bases.map((b) => verbs.find((v) => v.base === b)).filter((v): v is IrregularVerb => !!v);
}

/** How many verbs of a lesson are learned. */
export function learnedIn(lesson: VerbLesson, progress: Progress): number {
  return lesson.bases.filter((b) => isLearned(progress.get(b))).length;
}

/** The lesson to learn next: the first one with a verb not learned yet; null when all are. */
export function nextLesson(lessons: readonly VerbLesson[], progress: Progress): VerbLesson | null {
  return lessons.find((l) => learnedIn(l, progress) < l.bases.length) ?? null;
}

export interface ReviewSummary {
  learned: number;
  dueToday: number;
  dueTomorrow: number;
}

/** Learned verbs, those due now and those due within the next day. */
export function reviewSummary(progress: Progress, now: Date): ReviewSummary {
  let learned = 0;
  let dueToday = 0;
  let dueTomorrow = 0;
  for (const p of progress.values()) {
    if (!isLearned(p)) {
      continue;
    }
    learned++;
    const at = p.dueAt ? new Date(p.dueAt).getTime() : 0;
    if (at <= now.getTime()) {
      dueToday++;
    } else if (at <= now.getTime() + DAY) {
      dueTomorrow++;
    }
  }
  return { learned, dueToday, dueTomorrow };
}

/** The learned verbs, soonest due first, with whether each is due now. */
export function learnedVerbs(
  progress: Progress,
  now: Date,
  verbs: readonly IrregularVerb[] = IRREGULAR_VERBS,
): { verb: IrregularVerb; progress: VerbProgress; due: boolean }[] {
  const dueAt = (p: VerbProgress) => (p.dueAt ? new Date(p.dueAt).getTime() : 0);
  return verbs
    .filter((v) => isLearned(progress.get(v.base)))
    .map((v) => ({ verb: v, progress: progress.get(v.base)!, due: isDue(progress.get(v.base), now) }))
    .sort((a, b) => dueAt(a.progress) - dueAt(b.progress));
}

/**
 * The verbs of a review round: the due ones, longest waiting first, up to REVIEW_SIZE; with none
 * due, the learned ones due soonest (an early review). In a shuffled order.
 */
export function reviewRound(progress: Progress, now: Date, seed: number, size = REVIEW_SIZE): IrregularVerb[] {
  const learned = learnedVerbs(progress, now);
  const due = learned.filter((l) => l.due);
  return shuffle(
    (due.length > 0 ? due : learned).slice(0, size).map((l) => l.verb),
    seed,
  );
}

/** When a learned verb is reviewed again, in words: "hôm nay", "quá 2 ngày", "ngày mai", "sau 6 ngày". */
export function dueLabel(dueAt: string | null, now: Date): string {
  if (!dueAt) {
    return '';
  }
  const days = Math.round((startOfDay(new Date(dueAt)) - startOfDay(now)) / DAY);
  if (days < 0) {
    return `quá ${-days} ngày`;
  }
  if (days === 0) {
    return 'hôm nay';
  }
  return days === 1 ? 'ngày mai' : `sau ${days} ngày`;
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** Whether a typed form is one of the spellings of the form ("burnt/burned": either one). */
export function gradeForm(typed: string, form: string): boolean {
  const t = normalizeAnswer(typed);
  return t !== '' && (t === normalizeAnswer(form) || spellings(form).some((s) => normalizeAnswer(s) === t));
}

/** The form a learner who does not know the verb would guess: base + -ed ("think" → "thinked"). */
export function regularPast(base: string): string {
  if (base.endsWith('e')) {
    return base + 'd';
  }
  if (/[^aeiou]y$/.test(base)) {
    return base.slice(0, -1) + 'ied';
  }
  return base + 'ed';
}

export type Form = 'past' | 'participle';

export interface ChoiceQuestion {
  verb: IrregularVerb;
  form: Form;
  answer: string;
  options: string[];
}

/**
 * A multiple-choice question on one verb of a lesson: its V2 or V3, the right first spelling among
 * up to four options. The wrong ones are the regular guess (thinked), the base and the other form
 * when they differ, then the same form of the other verbs of the lesson.
 */
export function choiceQuestion(
  verb: IrregularVerb,
  form: Form,
  others: readonly IrregularVerb[],
  seed: number,
): ChoiceQuestion {
  const first = (f: string) => spellings(f)[0];
  const answer = first(form === 'past' ? verb.past : verb.participle);
  const other = first(form === 'past' ? verb.participle : verb.past);
  const all = new Set([...spellings(verb.past), ...spellings(verb.participle)]);
  const candidates = [
    regularPast(verb.base),
    verb.base,
    other,
    ...others.filter((o) => o.base !== verb.base).map((o) => first(form === 'past' ? o.past : o.participle)),
  ];
  const wrong: string[] = [];
  for (const c of candidates) {
    if (!all.has(c) && !wrong.includes(c) && wrong.length < 3) {
      wrong.push(c);
    }
  }
  return { verb, form, answer, options: shuffle([answer, ...wrong], seed) };
}

/** A form split where it stops matching the base, to colour what changes: buy / bought → b + ought. */
export function changedPart(base: string, form: string): [string, string] {
  let i = 0;
  while (i < base.length && i < form.length && base[i] === form[i]) {
    i++;
  }
  return [form.slice(0, i), form.slice(i)];
}

/** A lesson's pattern shown on one verb, its first: "send → sent → sent". */
export function modelForms(lesson: VerbLesson, verbs: readonly IrregularVerb[] = IRREGULAR_VERBS): string {
  const v = verbs.find((x) => x.base === lesson.bases[0]);
  return v ? [v.base, spellings(v.past)[0], spellings(v.participle)[0]].join(' → ') : '';
}

/**
 * An example cut into pieces, the forms of the verb marked, so the verb can be highlighted:
 * "I bought a bike." → [{"I ", false}, {"bought", true}, {" a bike.", false}].
 */
export function exampleParts(sentence: string, verb: IrregularVerb): { text: string; verb: boolean }[] {
  // The forms are plain letters ("bought"), so they need no escaping; longest first ("began" before "be").
  const forms = [...new Set([verb.base, ...spellings(verb.past), ...spellings(verb.participle)])].sort(
    (a, b) => b.length - a.length,
  );
  const re = new RegExp(`\\b(${forms.join('|')})\\b`, 'gi');
  const out: { text: string; verb: boolean }[] = [];
  let last = 0;
  for (const m of sentence.matchAll(re)) {
    if (m.index > last) {
      out.push({ text: sentence.slice(last, m.index), verb: false });
    }
    out.push({ text: m[0], verb: true });
    last = m.index + m[0].length;
  }
  if (last < sentence.length) {
    out.push({ text: sentence.slice(last), verb: false });
  }
  return out;
}
