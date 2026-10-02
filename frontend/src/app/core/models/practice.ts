/** Types for the lesson practice (F17, specs/016-vocab-practice/contracts/practice-api.md). */

/** `none` when the lesson has no practice yet. */
export type PracticeStatus = 'none' | 'running' | 'done' | 'failed';

/** An example sentence for one word of the lesson. */
export interface PracticeExample {
  lemma: string;
  sentence: string;
  /** Null until the audio file exists. */
  audioUrl: string | null;
}

export interface PracticeTurn {
  /** Index into the dialogue's `speakers` (0 or 1). */
  speaker: number;
  text: string;
  meaningVi: string;
  audioUrl: string | null;
}

export interface PracticeDialogue {
  speakers: string[];
  turns: PracticeTurn[];
}

/** A piece of a fill-in turn: plain text or the number of a blank. */
export type FillPart = { text: string } | { blank: number };

export interface FillTurn {
  speaker: number;
  /** Index of the dialogue turn this line comes from (its audio). */
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
  audioUrl: string | null;
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
}

/** Practice content as admins see it (GET /api/admin/lessons/{id}). */
export interface AdminPractice {
  objectiveVi: string;
  examples: { lemma: string; sentence: string }[];
  dialogue: {
    speakers: string[];
    turns: { speaker: number; text: string; meaningVi: string }[];
  } | null;
  grammarTipVi: string;
  translations: { vi: string; en: string; distractors: string[] }[];
}
