/** Types for the dictation API (specs/005-lesson-listening/contracts/dictation-api.md). */

export interface DictationResult {
  sentenceIndex: number;
  typed: string;
  correctWords: number;
  totalWords: number;
  checkedAt: string;
}

export interface DictationSummary {
  sentenceCount: number;
  checkedCount: number;
  correctWords: number;
  totalWords: number;
  /** correctWords / totalWords, 0 when nothing is checked. */
  rate: number;
  completed: boolean;
  results: DictationResult[];
}

export interface DictationInput {
  sentenceIndex: number;
  typed: string;
  correctWords: number;
  totalWords: number;
}
