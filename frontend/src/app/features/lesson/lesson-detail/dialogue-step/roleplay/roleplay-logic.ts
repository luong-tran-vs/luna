import { PracticeDialogue } from '../../../../../core/models/practice';
import { Comparison, compareDictation } from '../../../../../shared/utils/dictation-compare';

/** Pure helpers of the role play (added 2026-10-20): the learner says one speaker's lines. */

export type Verdict = 'ok' | 'close' | 'bad';

/** The learner's answer to one of their lines. */
export interface RoleAnswer {
  said: string;
  comparison: Comparison;
  /** Whether it was typed rather than spoken. */
  typed: boolean;
}

/** Every word right is ok, at least half is close, less is bad (as in Luyện nói). */
export function verdictOf(c: Comparison): Verdict {
  if (c.totalWords > 0 && c.correctWords === c.totalWords) {
    return 'ok';
  }
  return c.correctWords * 2 >= c.totalWords ? 'close' : 'bad';
}

/**
 * An answer compared with the line. A spoken one is not forgiven spelling slips (the recogniser
 * spells correctly or hears another word); a typed one is.
 */
export function compareAnswer(line: string, said: string, typed: boolean): Comparison {
  return compareDictation(line, said, { slips: typed });
}

/** Indexes of the turns of `speaker`. */
export function linesOf(dialogue: PracticeDialogue, speaker: number): number[] {
  return dialogue.turns.flatMap((t, i) => (t.speaker === speaker ? [i] : []));
}

/** The line with only its first `shown` words visible, the others as "___": the Gợi ý button. */
export function hintText(line: string, shown: number): string {
  return line
    .split(/\s+/)
    .filter((w) => w !== '')
    .map((w, i) => (i < shown ? w : '___'))
    .join(' ');
}

/** How many words the line has, i.e. how many times Gợi ý can be pressed. */
export function wordCount(line: string): number {
  return line.split(/\s+/).filter((w) => w !== '').length;
}

export interface RoleSummary {
  /** Lines of the learner. */
  lines: number;
  /** Lines answered (spoken or typed). */
  answered: number;
  /** Lines answered with every word right. */
  right: number;
  /** Words of the lines missed or said wrong, once each, in order. */
  missedWords: string[];
}

/** The end of a role play: how it went over the learner's lines. */
export function summarize(
  lines: readonly number[],
  answers: ReadonlyMap<number, RoleAnswer>,
): RoleSummary {
  const missed: string[] = [];
  let right = 0;
  let answered = 0;
  for (const i of lines) {
    const a = answers.get(i);
    if (!a) {
      continue;
    }
    answered++;
    if (verdictOf(a.comparison) === 'ok') {
      right++;
    }
    for (const w of a.comparison.words) {
      if (
        (w.status === 'missing' || w.status === 'wrong') &&
        w.expected &&
        !missed.includes(w.expected)
      ) {
        missed.push(w.expected);
      }
    }
  }
  return { lines: lines.length, answered, right, missedWords: missed };
}
