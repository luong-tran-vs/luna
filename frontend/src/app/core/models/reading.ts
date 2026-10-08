import { GrammarNote, Level, Sentence } from './lesson';

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
  /** Comprehension questions (F15); null when the lesson has none. */
  quiz: Quiz | null;
  grammarNote: GrammarNote | null;
  /** The grammar point of the lesson (F20); '' or missing when none is assigned. */
  grammarPointId?: string;
  /** A dialogue lesson as its "Name: text" lines; null (or missing) for a reading text. */
  turns?: TextTurn[] | null;
}

/** One line of a dialogue lesson. */
export interface TextTurn {
  speaker: string;
  /** What the speaker says, without the name. */
  text: string;
  /** Indexes of the lesson sentences the line is made of. */
  sentences: number[];
}

/** A question as learners see it: no answer until they answer. */
export interface QuizQuestion {
  prompt: string;
  options: string[];
}

export interface QuizAnswer {
  questionIndex: number;
  choice: number;
  correct: boolean;
  answerIndex: number;
  explanationVi: string;
}

export interface Quiz {
  version: number;
  questions: QuizQuestion[];
  /** This learner's answers to the current version. */
  answers: QuizAnswer[];
}

/** POST /api/lessons/{id}/answers. */
export interface AnswerInput {
  version: number;
  questionIndex: number;
  choice: number;
}

export interface AnswerResult {
  answer: QuizAnswer;
  answered: number;
  total: number;
  correct: number;
}

export type LookupSource = 'ai' | 'dictionary';

export interface LookupResult {
  source: LookupSource;
  text: string;
  lemma: string;
  ipa: string;
  meanings: { pos: string; text: string }[];
  /** Short Vietnamese explanation of an AI answer in context (F9); empty otherwise. */
  note?: string;
}

/** POST /api/lessons/{id}/ask (F9). */
export interface AskResult {
  result: LookupResult;
  /** Served from the stored answers, without calling the AI. */
  cached: boolean;
}

/** One piece of a sentence: a clickable word or the text between words. */
export interface Token {
  text: string;
  isWord: boolean;
  /** Position in the sentence's token list. */
  index: number;
}
