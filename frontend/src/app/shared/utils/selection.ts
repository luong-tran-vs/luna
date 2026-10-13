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
  /** The selection went on into another sentence: only its part in the first one is kept. */
  crossed: boolean;
}

/**
 * Turns the word tokens a selection touches into one range of whole words inside one sentence:
 * a lookup is limited to a sentence (any number of its words), so a selection over several
 * sentences keeps its part in the first one. Null when nothing is selected.
 */
export function selectWords(touched: TouchedWord[]): WordRange | null {
  if (touched.length === 0) {
    return null;
  }
  const sorted = [...touched].sort((a, b) => a.s - b.s || a.t - b.t);
  const sentence = sorted[0].s;
  const inSentence = sorted.filter((w) => w.s === sentence);
  return {
    sentence,
    first: inSentence[0].t,
    last: inSentence[inSentence.length - 1].t,
    count: inSentence.length,
    crossed: inSentence.length < sorted.length,
  };
}
