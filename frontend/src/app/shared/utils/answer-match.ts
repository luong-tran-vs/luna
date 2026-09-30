/** Lowercase, trimmed, single-spaced, with curly apostrophes made straight. */
export function normalizeAnswer(text: string): string {
  return text.replace(/[‘’]/g, "'").toLowerCase().trim().replace(/\s+/g, ' ');
}

/** Whether a typed answer is the card's word (Listen and type review, case-insensitive). */
export function isCorrectAnswer(typed: string, expected: string): boolean {
  const answer = normalizeAnswer(typed);
  return answer !== '' && answer === normalizeAnswer(expected);
}
