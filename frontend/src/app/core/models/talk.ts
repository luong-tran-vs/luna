/** Talk with the AI (added 2026-10-20): a conversation about a topic the learner studied. */

export type TalkRole = 'ai' | 'learner';
export type TalkStatus = 'open' | 'ended';

/** A topic the learner can talk about (GET /api/talk/topics). */
export interface TalkTopic {
  id: string;
  name: string;
  /** Level of the highest lesson studied in it. */
  level: string;
  lessons: number;
  sessions: number;
  lastAt?: string;
}

export interface TalkTurn {
  role: TalkRole;
  text: string;
  /** Vietnamese of the AI's lines. */
  textVi?: string;
  spoken?: boolean;
  at: string;
}

export interface TalkGood {
  said: string;
  noteVi: string;
}

export interface TalkFix {
  said: string;
  better: string;
  noteVi: string;
}

export interface TalkReview {
  overallVi: string;
  good: TalkGood[];
  fixes: TalkFix[];
  wordsUsed: string[];
  wordsUnused: string[];
}

export interface TalkSession {
  id: string;
  topicId: string;
  topicName: string;
  level: string;
  scenarioVi: string;
  turns: TalkTurn[];
  /** An answer to the AI's last line, for Gợi ý. */
  hint?: string;
  hintVi?: string;
  status: TalkStatus;
  review?: TalkReview;
  /** How many lines the learner may say. */
  maxTurns: number;
  createdAt: string;
  endedAt?: string;
}

/** A session in the history. */
export interface TalkSummary {
  id: string;
  topicId: string;
  topicName: string;
  scenarioVi: string;
  status: TalkStatus;
  learnerTurns: number;
  fixes: number;
  createdAt: string;
}
