import { PictureQuestion } from '../../../core/models/games';
import { nextCombo, pictureScore } from '../shared/game-score.logic';

/** A round of "Ảnh nhanh tay" (FR-010, FR-011), as a pure state. */
export interface PictureState {
  questions: readonly PictureQuestion[];
  index: number;
  combo: number;
  score: number;
  right: number;
  wrong: number;
  /** The words answered wrong, each once, in order. */
  missed: PictureQuestion[];
  /** The option chosen for the current question; null before answering. */
  chosen: number | null;
  /** The points the last answer gave (0 when wrong). */
  gained: number;
}

export function startPicture(questions: readonly PictureQuestion[]): PictureState {
  return { questions, index: 0, combo: 1, score: 0, right: 0, wrong: 0, missed: [], chosen: null, gained: 0 };
}

/** The question being asked; null once every question was asked. */
export function currentPicture(s: PictureState): PictureQuestion | null {
  return s.questions[s.index] ?? null;
}

/**
 * Answers the current question with option i: right raises the combo (up to x5) and scores
 * 100 × the new combo; wrong drops it to x1 and keeps the word once. A second answer is ignored.
 */
export function answerPicture(s: PictureState, i: number): PictureState {
  const q = currentPicture(s);
  if (!q || s.chosen !== null) {
    return s;
  }
  const right = i === q.answer;
  const combo = nextCombo(s.combo, right);
  const gained = right ? pictureScore(combo) : 0;
  return {
    ...s,
    chosen: i,
    combo,
    gained,
    score: s.score + gained,
    right: s.right + (right ? 1 : 0),
    wrong: s.wrong + (right ? 0 : 1),
    missed: right || s.missed.some((m) => m.lemma === q.lemma) ? s.missed : [...s.missed, q],
  };
}

/** Goes on to the next question. */
export function nextPicture(s: PictureState): PictureState {
  return { ...s, index: s.index + 1, chosen: null, gained: 0 };
}

/** Whether every question was asked. */
export function pictureFinished(s: PictureState): boolean {
  return s.index >= s.questions.length;
}
