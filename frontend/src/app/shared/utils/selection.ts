/** A word token touched by a text selection: sentence index and token index. */
export interface TouchedWord {
  s: number;
  t: number;
}

export interface WordRange {
  sentence: number;
  first: number;
  last: number;
  /** Number of words in the range. */
  count: number;
}

export const MAX_SELECTED_WORDS = 6;

/**
 * Turns the word tokens a selection touches into one range of whole words inside the first
 * sentence. Null when nothing is selected; an error when more than 6 words are selected.
 */
export function selectWords(touched: TouchedWord[]): WordRange | { error: 'too-long' } | null {
  if (touched.length === 0) {
    return null;
  }
  const sorted = [...touched].sort((a, b) => a.s - b.s || a.t - b.t);
  const sentence = sorted[0].s;
  const inSentence = sorted.filter((w) => w.s === sentence);
  if (inSentence.length > MAX_SELECTED_WORDS) {
    return { error: 'too-long' };
  }
  return { sentence, first: inSentence[0].t, last: inSentence[inSentence.length - 1].t, count: inSentence.length };
}
