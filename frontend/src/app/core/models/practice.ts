import { GrammarExercise } from './grammar-study';

/** Types for the lesson practice (F17, specs/016-vocab-practice/contracts/practice-api.md). */

/** `none` when the lesson has no practice yet. */
export type PracticeStatus = 'none' | 'running' | 'done' | 'failed';

/** An example sentence for one word of the lesson. */
export interface PracticeExample {
  lemma: string;
  sentence: string;
  /** The sentence in Vietnamese; empty for practice generated before it was asked for. */
  meaningVi?: string;
}

export interface PracticeTurn {
  /** Index into the dialogue's `speakers` (0 or 1). */
  speaker: number;
  text: string;
  meaningVi: string;
}

export interface PracticeDialogue {
  speakers: string[];
  turns: PracticeTurn[];
}

/** A piece of a fill-in turn: plain text or the number of a blank. */
export type FillPart = { text: string } | { blank: number };

export interface FillTurn {
  speaker: number;
  /** Index of the dialogue turn this line comes from. */
  turnIndex: number;
  meaningVi: string;
  parts: FillPart[];
}

export interface FillBlank {
  answer: string;
}

/** Step 3: the dialogue with vocabulary words replaced by blanks. */
export interface PracticeFill {
  turns: FillTurn[];
  blanks: FillBlank[];
  /** Answers plus up to 2 other lesson words, in a fixed order (the page shuffles them). */
  wordBank: string[];
}

/** Step 4: a Vietnamese sentence to build in English from tiles. */
export interface PracticeTranslation {
  vi: string;
  /** The English answer split into tiles, punctuation kept on the words. */
  answer: string[];
  /** Answer tiles plus distractors, in a fixed order (the page shuffles them). */
  tiles: string[];
}

/** GET /api/lessons/{id}/practice. */
export interface PracticeView {
  status: PracticeStatus;
  /** Position in the topic roadmap (from 1); 0 when the lesson is not in it. */
  lessonNumber: number;
  objectiveVi: string;
  examples: PracticeExample[];
  dialogue: PracticeDialogue | null;
  fill: PracticeFill | null;
  grammarTipVi: string;
  translations: PracticeTranslation[];
  /** The content of the six-step lesson (added 2026-10-22); null for a practice made before it. */
  steps?: LessonSteps | null;
  /** The sentences of the Listening step to write down (every sentence for older lessons). */
  dictation?: DictationItem[];
  /** The lesson's grammar point (or the one picked for it); '' when none. */
  grammarPointId?: string;
  grammarTitleVi?: string;
}

/** A question on the dialogue of the Listening step; a tfn question has three fixed options. */
export interface ListenQuestion {
  kind: 'gist' | 'detail' | 'tfn';
  prompt: string;
  /** The Vietnamese of an English prompt, '' when the prompt is Vietnamese already. */
  promptVi: string;
  options: string[];
  answerIndex: number;
  explanationVi: string;
  /** The dialogue turn that holds the answer, -1 for the whole dialogue. */
  turnIndex: number;
}

/** A sentence of the lesson with the words that show its grammar point. */
export interface Noticing {
  sentence: string;
  highlights: string[];
}

/** The grammar step: notice, ask why, the rule in one card, a few exercises. */
export interface GrammarStepContent {
  titleVi: string;
  noticing: Noticing[];
  /** '' when there is no question. */
  questionVi: string;
  options: string[];
  answerIndex: number;
  explanationVi: string;
  whenVi: string;
  table: { subject: string; form: string; negative: string }[];
  mistakes: { wrong: string; right: string }[];
  noteVi: string;
  exercises: GrammarExercise[];
}

export interface ReadAloud {
  text: string;
  tipVi: string;
}

export interface SpeakQuestion {
  question: string;
  questionVi: string;
  /** An answer frame with "___" for the learner's own words; may be ''. */
  frame: string;
}

/** What the Writing step shows next to its prompt. */
export interface WritingHelp {
  /** The requirement on the lesson's grammar, '' when none; met when the text has one of anyOf. */
  grammarLabelVi: string;
  anyOf: string[];
  outlineVi: string[];
  modelAnswer: string;
}

export interface LessonSteps {
  listeningQuestions: ListenQuestion[];
  readingGuideVi: string;
  grammar: GrammarStepContent | null;
  readAloud: ReadAloud[];
  speakQuestions: SpeakQuestion[];
  writing: WritingHelp | null;
}

export interface DictationItem {
  text: string;
  /** The lesson word the sentence was written for; '' for a sentence of the text. */
  lemma: string;
}

/** POST /api/lessons/{id}/speak/check: one answer said aloud, and its check. */
export interface SpeakAnswer {
  question: string;
  said: string;
}

export interface SpeakResult {
  said: string;
  /** Equals said when the answer is fine. */
  corrected: string;
  noteVi: string;
}

/** Practice content as admins see it (GET /api/admin/lessons/{id}). */
export interface AdminPractice {
  objectiveVi: string;
  examples: PracticeExample[];
  dialogue: {
    speakers: string[];
    turns: { speaker: number; text: string; meaningVi: string }[];
  } | null;
  grammarTipVi: string;
  translations: { vi: string; en: string; distractors: string[] }[];
}
