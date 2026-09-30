/** Dictation comparison (F4, research R1): word-level edit distance between the sentence and what was typed. */

export type WordStatus = 'ok' | 'wrong' | 'missing';

/**
 * One word of the result, in sentence order. `wrong` without `expected` is an extra typed word;
 * `missing` has only `expected`.
 */
export interface ComparedWord {
  status: WordStatus;
  word?: string;
  expected?: string;
}

export interface Comparison {
  words: ComparedWord[];
  correctWords: number;
  /** Words of the sentence plus extra typed words. */
  totalWords: number;
}

const LETTER = /\p{L}/u;
const DIGIT = /\p{N}/u;
const WORD_CHAR = /[\p{L}\p{N}\s]/u;

/** Lowercase words without punctuation; keeps `don't`, `9.30` and `1,000` whole, splits `well-known`. */
export function normalizeWords(text: string): string[] {
  const s = text
    .replace(/[‘’]/g, "'")
    .toLowerCase()
    .replace(/[-–—]/g, ' ');
  let out = '';
  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    const prev = s[i - 1] ?? '';
    const next = s[i + 1] ?? '';
    const keep =
      WORD_CHAR.test(c) ||
      ((c === '.' || c === ',') && DIGIT.test(prev) && DIGIT.test(next)) ||
      (c === "'" && LETTER.test(prev) && LETTER.test(next));
    out += keep ? c : ' ';
  }
  return out.split(/\s+/).filter((w) => w !== '');
}

/** Aligns the typed words to the sentence; ties prefer match, then wrong, then missing, then extra. */
export function compareDictation(expected: string, typed: string): Comparison {
  const e = normalizeWords(expected);
  const t = normalizeWords(typed);
  const n = e.length;
  const m = t.length;

  // d[i][j]: edits to turn t[0..j) into e[0..i).
  const d: number[][] = Array.from({ length: n + 1 }, (_, i) =>
    Array.from({ length: m + 1 }, (_, j) => (i === 0 ? j : j === 0 ? i : 0)),
  );
  for (let i = 1; i <= n; i++) {
    for (let j = 1; j <= m; j++) {
      const diag = d[i - 1][j - 1] + (e[i - 1] === t[j - 1] ? 0 : 1);
      d[i][j] = Math.min(diag, d[i - 1][j] + 1, d[i][j - 1] + 1);
    }
  }

  const words: ComparedWord[] = [];
  let i = n;
  let j = m;
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && e[i - 1] === t[j - 1] && d[i][j] === d[i - 1][j - 1]) {
      words.push({ status: 'ok', word: t[j - 1], expected: e[i - 1] });
      i--;
      j--;
    } else if (i > 0 && j > 0 && d[i][j] === d[i - 1][j - 1] + 1) {
      words.push({ status: 'wrong', word: t[j - 1], expected: e[i - 1] });
      i--;
      j--;
    } else if (i > 0 && d[i][j] === d[i - 1][j] + 1) {
      words.push({ status: 'missing', expected: e[i - 1] });
      i--;
    } else {
      words.push({ status: 'wrong', word: t[j - 1] });
      j--;
    }
  }
  words.reverse();

  const extras = words.filter((w) => w.status === 'wrong' && w.expected === undefined).length;
  return { words, correctWords: words.filter((w) => w.status === 'ok').length, totalWords: n + extras };
}
