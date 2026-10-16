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

/** One piece of a structure's pattern; `tone` (0–3) is the same for the same piece in every structure. */
export interface PatternPart {
  text: string;
  tone: number;
}

/** How many colours the pieces of the patterns cycle through. */
const PATTERN_TONES = 4;

/**
 * The patterns of a point's structures cut at "+": "S + am/is/are + V-ing" gives three pieces. The
 * same piece (case and a closing "?" or "." ignored) gets the same tone in every structure, so
 * "am/is/are" and "Am/Is/Are?" are coloured alike in the affirmative and the question.
 */
export function patternParts(patterns: readonly string[]): PatternPart[][] {
  const tones = new Map<string, number>();
  return patterns.map((pattern) =>
    pattern
      .split('+')
      .map((s) => s.trim())
      .filter((s) => s !== '')
      .map((text) => {
        const key = text.toLowerCase().replace(/[?.!]+$/, '');
        if (!tones.has(key)) {
          tones.set(key, tones.size % PATTERN_TONES);
        }
        return { text, tone: tones.get(key)! };
      }),
  );
}

export type StructureTone = 'positive' | 'negative' | 'question' | 'other';

/** The colour of a structure's label: affirmative, negative, question, or anything else. */
export function structureTone(label: string): StructureTone {
  const l = label.toLowerCase();
  if (l.includes('phủ định')) {
    return 'negative';
  }
  if (l.includes('nghi vấn') || l.includes('câu hỏi')) {
    return 'question';
  }
  return l.includes('khẳng định') ? 'positive' : 'other';
}
