import { Level } from './lesson';

/** Types for the writing step (specs/014-writing-step/contracts/writing-api.md). */

export type WritingStatus = 'draft' | 'submitted';
export type GradeStatus = 'pending' | 'done' | 'failed';
export type CriterionName = 'task' | 'grammar' | 'vocabulary' | 'coherence';

export interface Criterion {
  name: CriterionName;
  score: number;
  commentVi: string;
}

export interface Grade {
  status: GradeStatus;
  error: string;
  criteria: Criterion[];
  /** Mean of the four scores, one decimal; null until graded. */
  average: number | null;
  overallVi: string;
  correctedText: string;
  gradedAt: string | null;
}

export interface Writing {
  id: string;
  lessonId: string;
  lessonTitle: string;
  prompt: string;
  text: string;
  status: WritingStatus;
  submittedAt: string | null;
  /** null while it is a draft. */
  grade: Grade | null;
  /** How many of its gradings (submit, resubmit, regrade) the writing used; at most max (2). */
  gradings?: { used: number; max: number };
}

/** GET /api/lessons/{id}/writing. */
export interface LessonWriting {
  prompt: string;
  level: Level;
  /** The lesson being studied, at the Write step. */
  canWrite: boolean;
  writing: Writing | null;
}

export interface WritingSummary {
  id: string;
  lessonId: string;
  lessonTitle: string;
  submittedAt: string;
  gradeStatus: GradeStatus;
  average: number | null;
  seen: boolean;
}

export interface UnseenCount {
  unseen: number;
  pending: number;
  latest: { id: string; status: GradeStatus } | null;
}

export const MIN_WRITING_WORDS = 5;
export const MAX_WRITING_WORDS = 400;

/** Suggested length per level; only a hint, the limits are 5–400 words. */
export const SUGGESTED_WORDS: Record<Level, { min: number; max: number }> = {
  A1: { min: 30, max: 60 },
  A2: { min: 50, max: 80 },
  B1: { min: 80, max: 120 },
  B2: { min: 120, max: 180 },
  C1: { min: 150, max: 250 },
  C2: { min: 150, max: 250 },
};

export const CRITERIA_LABELS: Record<CriterionName, string> = {
  task: 'Hoàn thành yêu cầu',
  grammar: 'Ngữ pháp',
  vocabulary: 'Từ vựng',
  coherence: 'Mạch lạc',
};

/** "3,8" — scores are shown with a Vietnamese decimal comma. */
export function formatScore(score: number): string {
  return score.toFixed(1).replace('.', ',');
}
