import { AttemptBody, AttemptKind, GrammarExercise, MASTERY_PERCENT } from '../../core/models/grammar-study';
import { normalizeAnswer } from '../../shared/utils/answer-match';
import { shuffle } from '../lesson/lesson-detail/practice-logic';

/** Pure helpers of the grammar exercises (F20): grading in the browser, shuffling and scoring. */

/** A multiple-choice answer is right when the chosen option is the marked one. */
export function gradeChoice(ex: GrammarExercise, chosen: number): boolean {
  return ex.answerIndex !== undefined && chosen === ex.answerIndex;
}

/**
 * A typed answer is right when it equals one accepted answer. Both sides are normalised: curly
 * apostrophes become straight, runs of spaces become one, outer spaces go, case is ignored.
 */
export function gradeFill(ex: GrammarExercise, typed: string): boolean {
  const t = normalizeAnswer(typed);
  return t !== '' && (ex.answers ?? []).some((a) => normalizeAnswer(a) === t);
}

/** The right answer of an exercise as text (the first accepted answer of a fill). */
export function correctAnswerText(ex: GrammarExercise): string {
  switch (ex.kind) {
    case 'choice':
      return ex.options?.[ex.answerIndex ?? -1] ?? '';
    case 'fill':
      return ex.answers?.[0] ?? '';
    default:
      return ex.sentence ?? '';
  }
}

/** A built sentence is right when its words, in order, are the words of the sentence. */
export function gradeReorder(ex: GrammarExercise, built: readonly string[]): boolean {
  const target = normalizeAnswer(ex.sentence ?? '').split(' ');
  return built.length === target.length && built.every((w, i) => normalizeAnswer(w) === target[i]);
}

/** The word tiles of a reorder exercise in a random order; the same seed gives the same order. */
export function shuffleWords(words: readonly string[], seed: number): string[] {
  return shuffle(words, seed);
}

/** The text of a fill exercise around its blank: [before, after]; no blank gives [text, '']. */
export function splitBlank(text: string): [string, string] {
  const i = text.indexOf('___');
  return i < 0 ? [text, ''] : [text.slice(0, i), text.slice(i + 3)];
}

export interface Outcome {
  id: string;
  correct: boolean;
  /** What the learner answered, as text (for the review of a test). */
  given?: string;
}

/** The body of POST /api/grammar/{id}/attempts for a finished round. */
export function toAttempt(kind: AttemptKind, outcomes: readonly Outcome[]): AttemptBody {
  const wrong = outcomes.filter((o) => !o.correct).map((o) => o.id);
  return { kind, correct: outcomes.length - wrong.length, total: outcomes.length, wrong };
}

export function percent(correct: number, total: number): number {
  return total > 0 ? Math.round((correct * 100) / total) : 0;
}

/** How many more right answers a mastery test needed to reach the pass mark (0 when it passed). */
export function missingToPass(correct: number, total: number): number {
  const needed = Math.ceil((total * MASTERY_PERCENT) / 100);
  return Math.max(0, needed - correct);
}
