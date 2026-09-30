import { Level, Sentence } from './lesson';

/** A lesson as a learner reads it (GET /api/lessons/{id}). */
export interface ReadingLesson {
  id: string;
  title: string;
  level: Level;
  topic: string;
  sentences: Sentence[];
  /** Sentence indexes grouped by paragraph. */
  paragraphs: number[][];
  /** Lowercase word → base form, for highlighting saved words. */
  lemmas: Record<string, string>;
  /** Multi-word annotations, for highlighting saved phrases. */
  phrases: { text: string; lemma: string }[];
}

export type LookupSource = 'ai' | 'dictionary';

export interface LookupResult {
  source: LookupSource;
  text: string;
  lemma: string;
  ipa: string;
  meanings: { pos: string; text: string }[];
}

/** One piece of a sentence: a clickable word or the text between words. */
export interface Token {
  text: string;
  isWord: boolean;
  /** Position in the sentence's token list. */
  index: number;
}
