/**
 * Dictation comparison (F4, research R1): word-level edit distance between the sentence and what was
 * typed. Numbers in digits or words and hyphenated or closed compounds count as the same word.
 */
import { differsByEnding, isSpellingSlip, sameSkeleton } from './answer-match';

/** `close`: heard right but spelt a little wrong ("musik"), or two words typed as one ("alot"). */
export type WordStatus = 'ok' | 'close' | 'wrong' | 'missing';

/**
 * One word of the result, in sentence order. `wrong` without `expected` is an extra typed word;
 * `missing` has only `expected`. A `close` word typed as one for two words of the sentence has
 * both in `expected` ("a lot") and `count` 2.
 */
export interface ComparedWord {
  status: WordStatus;
  word?: string;
  expected?: string;
  /** How many words of the sentence it stands for; 1 when absent. */
  count?: number;
}

export interface Comparison {
  words: ComparedWord[];
  /** Words heard right, the close ones included. */
  correctWords: number;
  /** Of correctWords, those spelt a little wrong or joined (updated 2026-10-20). */
  closeWords: number;
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

const NUMBER_WORDS = [
  'zero', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten',
  'eleven', 'twelve', 'thirteen', 'fourteen', 'fifteen', 'sixteen', 'seventeen', 'eighteen',
  'nineteen', 'twenty',
];
const TENS = ['', '', 'twenty', 'thirty', 'forty', 'fifty', 'sixty', 'seventy', 'eighty', 'ninety'];

/**
 * The form two words are compared in: a number in digits as its word ("12" = "twelve"), and
 * "oclock" as "o'clock". Other spellings must match exactly.
 */
function sameAs(word: string): string {
  if (/^\d+$/.test(word)) {
    const n = Number(word);
    if (n <= 20) {
      return NUMBER_WORDS[n];
    }
    if (n < 100 && n % 10 === 0) {
      return TENS[n / 10];
    }
  }
  return word === 'oclock' ? "o'clock" : word;
}

/**
 * Writes a hyphenated word of `text` as one word when `other` has it so ("co-workers" when the
 * sentence says "coworkers"); other hyphens stay, and split the word in two.
 */
function joinCompounds(text: string, other: string): string {
  const words = new Set(normalizeWords(other.replace(/[-–—]/g, '')));
  return text.replace(/(\p{L}+)-(\p{L}+)/gu, (whole, a: string, b: string) =>
    words.has((a + b).toLowerCase()) ? a + b : whole,
  );
}

/**
 * Splits a typed word that is two words of the sentence run together ("alot" for "a lot"): the
 * typed words, and for each whether it is the first or second half of such a word.
 */
function splitJoined(typed: string[], expected: string[]): { words: string[]; joined: ('first' | 'second' | null)[] } {
  const pairs = new Map<string, [string, string]>();
  for (let i = 0; i + 1 < expected.length; i++) {
    pairs.set(expected[i] + expected[i + 1], [expected[i], expected[i + 1]]);
  }
  const words: string[] = [];
  const joined: ('first' | 'second' | null)[] = [];
  for (const w of typed) {
    const pair = expected.includes(w) ? undefined : pairs.get(w);
    if (pair) {
      words.push(pair[0], pair[1]);
      joined.push('first', 'second');
    } else {
      words.push(w);
      joined.push(null);
    }
  }
  return { words, joined };
}

/**
 * Aligns the typed words to the sentence; ties prefer match, then a spelling slip, then wrong, then
 * missing, then extra. A slip costs 2, a wrong word close to the right one (an
 * ending, a vowel) 3 so the alignment pairs them, any other edit 4. With slips false
 * (speech recognition, which only writes real words) nothing is close: a near word was said wrong.
 */
export function compareDictation(expected: string, typed: string, { slips = true } = {}): Comparison {
  const e = normalizeWords(joinCompounds(expected, typed));
  const ec = e.map(sameAs);
  const typedWords = normalizeWords(joinCompounds(typed, expected));
  const split = slips ? splitJoined(typedWords, e) : { words: typedWords, joined: typedWords.map(() => null) };
  const t = split.words;
  const tc = t.map(sameAs);
  const n = e.length;
  const m = t.length;
  const sub = (a: string, b: string) =>
    a === b ? 0 : slips && isSpellingSlip(b, a) ? 2 : differsByEnding(a, b) || sameSkeleton(a, b) ? 3 : 4;

  // d[i][j]: cost to turn t[0..j) into e[0..i).
  const d: number[][] = Array.from({ length: n + 1 }, (_, i) =>
    Array.from({ length: m + 1 }, (_, j) => (i === 0 ? 4 * j : j === 0 ? 4 * i : 0)),
  );
  for (let i = 1; i <= n; i++) {
    for (let j = 1; j <= m; j++) {
      d[i][j] = Math.min(d[i - 1][j - 1] + sub(ec[i - 1], tc[j - 1]), d[i - 1][j] + 4, d[i][j - 1] + 4);
    }
  }

  const words: ComparedWord[] = [];
  let i = n;
  let j = m;
  while (i > 0 || j > 0) {
    const cost = i > 0 && j > 0 ? sub(ec[i - 1], tc[j - 1]) : -1;
    if (cost === 0 && d[i][j] === d[i - 1][j - 1]) {
      const half = split.joined[j - 1];
      if (half === 'second' && i > 1 && split.joined[j - 2] === 'first' && ec[i - 2] === tc[j - 2] && d[i - 1][j - 1] === d[i - 2][j - 2]) {
        words.push({ status: 'close', word: t[j - 2] + t[j - 1], expected: `${e[i - 2]} ${e[i - 1]}`, count: 2 });
        i -= 2;
        j -= 2;
        continue;
      }
      words.push({ status: half ? 'close' : 'ok', word: t[j - 1], expected: e[i - 1] });
      i--;
      j--;
    } else if (cost >= 1 && d[i][j] === d[i - 1][j - 1] + cost) {
      words.push({ status: cost === 2 ? 'close' : 'wrong', word: t[j - 1], expected: e[i - 1] });
      i--;
      j--;
    } else if (i > 0 && d[i][j] === d[i - 1][j] + 4) {
      words.push({ status: 'missing', expected: e[i - 1] });
      i--;
    } else {
      words.push({ status: 'wrong', word: t[j - 1] });
      j--;
    }
  }
  words.reverse();

  const extras = words.filter((w) => w.status === 'wrong' && w.expected === undefined).length;
  const counted = (status: WordStatus) =>
    words.filter((w) => w.status === status).reduce((sum, w) => sum + (w.count ?? 1), 0);
  const closeWords = counted('close');
  return { words, correctWords: counted('ok') + closeWords, closeWords, totalWords: n + extras };
}
