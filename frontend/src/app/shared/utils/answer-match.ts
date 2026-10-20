/** Lowercase, trimmed, single-spaced, with curly apostrophes made straight. */
export function normalizeAnswer(text: string): string {
  return text.replace(/[‘’]/g, "'").toLowerCase().trim().replace(/\s+/g, ' ');
}

/** Whether a typed answer is the card's word (Listen and type review, case-insensitive). */
export function isCorrectAnswer(typed: string, expected: string): boolean {
  const answer = normalizeAnswer(typed);
  return answer !== '' && answer === normalizeAnswer(expected);
}

/** Levenshtein distance: the letters to add, remove or change to turn a into b. */
export function editDistance(a: string, b: string): number {
  let prev = Array.from({ length: b.length + 1 }, (_, j) => j);
  for (let i = 1; i <= a.length; i++) {
    const row = [i];
    for (let j = 1; j <= b.length; j++) {
      row[j] = Math.min(prev[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1), prev[j] + 1, row[j - 1] + 1);
    }
    prev = row;
  }
  return prev[b.length];
}

/**
 * Whether a typed word is a slip in spelling the expected one rather than another word (updated
 * 2026-10-20): one letter off in a word of 5 letters or more ("musik" for "music"), two in one of
 * 8 or more. A short word one letter off is most often another word ("want" for "went"), so it is
 * not a slip; nor is a missing or added "-s" or "-ed" ("like" for "likes"), an ending not heard.
 */
export function isSpellingSlip(typed: string, expected: string): boolean {
  if (typed === expected || /\d/.test(typed + expected) || differsByEnding(typed, expected)) {
    return false;
  }
  const allowed = expected.length >= 8 ? 2 : expected.length >= 5 ? 1 : 0;
  return allowed > 0 && editDistance(typed, expected) <= allowed;
}

const ENDINGS = ['s', 'es', 'ed', 'd'];

/** One word is the other with an "-s" or "-ed" ending ("like"/"likes", "play"/"played"). */
export function differsByEnding(a: string, b: string): boolean {
  const [short, long] = a.length < b.length ? [a, b] : [b, a];
  return ENDINGS.some((e) => long === short + e);
}

/** Two words of the same length with the same consonants in order, "went" and "want". */
export function sameSkeleton(a: string, b: string): boolean {
  const consonants = (w: string) => w.replace(/[aeiouy]/g, '');
  return a !== b && a.length > 1 && a.length === b.length && consonants(a) === consonants(b);
}

/**
 * How a typed answer went: exactly the word, "close" (a spelling slip, or the right letters with a
 * space left out or added, "alot" for "a lot"), or wrong.
 */
export function matchAnswer(typed: string, expected: string): 'exact' | 'close' | 'wrong' {
  const t = normalizeAnswer(typed);
  const e = normalizeAnswer(expected);
  if (t === '') {
    return 'wrong';
  }
  if (t === e) {
    return 'exact';
  }
  if (t.replace(/ /g, '') === e.replace(/ /g, '') || isSpellingSlip(t, e)) {
    return 'close';
  }
  return 'wrong';
}

/**
 * The hint of a word to type: its first letter and a blank for each other letter, word by word
 * ("l _ _ _", "t _ _ _ / a / b _ _ _ _"), so a phrase shows how many words it has.
 */
export function letterHint(word: string): string {
  if (!word.trim()) {
    return '';
  }
  return word
    .trim()
    .split(/\s+/)
    .map((w) => w[0] + ' _'.repeat(w.length - 1))
    .join(' / ');
}
