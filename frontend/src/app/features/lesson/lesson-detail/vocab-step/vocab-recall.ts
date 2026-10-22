import { VocabItem } from '../../../../core/models/vocab';
import { shuffle } from '../practice-logic';

/**
 * The quick recall after the words (six-step lesson, 2026-10-22): one question per word, asked in two
 * ways in turn (hear the word, or read its meaning, then pick it among four), so the learner retrieves
 * each word once right after seeing it. Pure helpers, the step keeps the state.
 */

export type RecallKind = 'listen' | 'meaning';

export interface RecallItem {
  kind: RecallKind;
  /** The word to find. */
  word: VocabItem;
  /** Lemmas to choose from, the word's own among them. */
  options: string[];
}

/** How many options a question has, at most (fewer when the lesson has fewer words). */
const OPTIONS = 4;

/**
 * One question per word, in a shuffled order, alternating listening and meaning; the options are the
 * word and up to three other words of the lesson. Fewer than two words give no questions: there would be
 * nothing to choose between.
 */
export function buildRecall(words: readonly VocabItem[], seed: number): RecallItem[] {
  if (words.length < 2) {
    return [];
  }
  return shuffle(words, seed).map((word, i) => {
    const others = shuffle(
      words.filter((w) => w.lemma !== word.lemma),
      seed + i + 1,
    ).slice(0, OPTIONS - 1);
    return {
      kind: i % 2 === 0 ? 'listen' : 'meaning',
      word,
      options: shuffle([word, ...others], seed + 100 + i).map((w) => w.lemma),
    };
  });
}
