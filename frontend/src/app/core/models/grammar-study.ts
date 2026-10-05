import { Level } from './lesson';

/** Types of the learner's grammar section (F20): GET /api/grammar, /api/grammar/{id}, POST …/attempts. */

export type GrammarStatus = 'new' | 'learning' | 'mastered';
export type ExerciseKind = 'choice' | 'fill' | 'reorder';
export type AttemptKind = 'practice' | 'mastery';

/** Score (in percent) a mastery test needs for a point to count as mastered. */
export const MASTERY_PERCENT = 80;

/** One row of the list of grammar points. */
export interface GrammarListItem {
  id: string;
  level: Level;
  titleVi: string;
  titleEn: string;
  hintVi: string;
  /** A lesson was published for this point. */
  available: boolean;
  status: GrammarStatus;
  bestMastery: number;
}

export interface GrammarList {
  points: GrammarListItem[];
  /** The first available point not yet mastered; '' when none. */
  next: string;
}

/** A self-graded exercise; the fields used depend on `kind`. */
export interface GrammarExercise {
  id: string;
  kind: ExerciseKind;
  promptVi?: string;
  text?: string;
  options?: string[];
  answerIndex?: number;
  answers?: string[];
  sentence?: string;
  words?: string[];
  explanationVi: string;
}

export interface GrammarStructure {
  label: string;
  pattern: string;
  example: string;
}

export interface GrammarExample {
  en: string;
  vi: string;
}

export interface GrammarMistake {
  wrong: string;
  right: string;
  noteVi: string;
}

export interface GrammarContent {
  objective: string;
  explanation: string[];
  usage: string[];
  structures: GrammarStructure[];
  examples: GrammarExample[];
  mistakes: GrammarMistake[];
  practice: GrammarExercise[];
  mastery: GrammarExercise[];
}

export interface GrammarProgress {
  status: GrammarStatus;
  practiceAttempts: number;
  lastPractice: number;
  masteryAttempts: number;
  bestMastery: number;
  mastered: boolean;
  /** Ids of the practice exercises answered wrong the last time. */
  weak: string[];
}

export interface GrammarPointInfo {
  id: string;
  level: Level;
  titleVi: string;
  titleEn: string;
  pattern: string;
  hintVi: string;
  examples: string[];
}

export interface GrammarDetail {
  point: GrammarPointInfo;
  content: GrammarContent;
  progress: GrammarProgress;
  /** Topic lessons that use this point. */
  lessonCount: number;
}

export interface AttemptBody {
  kind: AttemptKind;
  correct: number;
  total: number;
  wrong: string[];
}

export interface AttemptResult {
  progress: GrammarProgress;
  /** Only meaningful for a mastery test. */
  passed: boolean;
}

/** F21: why a learner reports an exercise (POST /api/grammar/{id}/reports). */
export type ReportReason = 'wrong_answer' | 'ambiguous' | 'typo' | 'other';

export const REPORT_REASONS: readonly { value: ReportReason; label: string }[] = [
  { value: 'wrong_answer', label: 'Đáp án sai' },
  { value: 'ambiguous', label: 'Câu mơ hồ' },
  { value: 'typo', label: 'Lỗi chính tả' },
  { value: 'other', label: 'Khác' },
];

/** Longest note a report may carry. */
export const MAX_REPORT_NOTE = 300;

export interface ReportBody {
  exerciseId: string;
  reason: ReportReason;
  note: string;
}

export const STATUS_LABELS: Record<GrammarStatus, string> = {
  new: 'Mới',
  learning: 'Đang học',
  mastered: 'Đã nắm vững',
};
