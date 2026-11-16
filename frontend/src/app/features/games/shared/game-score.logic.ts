/** Scores shared by the games (FR-011). */

/** The highest combo of "Ảnh nhanh tay". */
export const MAX_COMBO = 5;

/** The combo after an answer: +1 when right (up to MAX_COMBO), back to x1 when wrong. */
export function nextCombo(combo: number, right: boolean): number {
  return right ? Math.min(combo + 1, MAX_COMBO) : 1;
}

/** The points of a right picture answer given at combo: 100 × combo. */
export function pictureScore(combo: number): number {
  return 100 * combo;
}

/** Whether score beats the best (a first round with points is a best). */
export function isNewBest(score: number, best: number): boolean {
  return score > best;
}

/** "2.100": thousands with a dot, as in the sketch. */
export function formatScore(n: number): string {
  return Math.round(n)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, '.');
}
