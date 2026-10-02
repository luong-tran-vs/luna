import { Level } from './lesson';

/** Types for the study API (specs/008-daily-study-flow/contracts/study-api.md). */

/** The steps of a lesson that count toward progress (L); the review step was dropped on 2026-10-02. */
export type Step = 'read' | 'listen' | 'write';
export const STEPS: readonly Step[] = ['read', 'listen', 'write'];

export type StepState = 'done' | 'current' | 'locked';

export interface GoalView {
  topicId: string;
  topicName: string;
  level: Level;
  completedLessons: number;
  totalLessons: number;
  status: 'active' | 'paused';
  effectiveFrom: string;
}

export interface Goals {
  active: GoalView | null;
  others: GoalView[];
}

/** Choosing a goal takes effect at once. */
export interface SetGoalResult {
  active: GoalView;
}

/** Where the learner stands in the goal's roadmap. */
export type StudyKind = 'noGoal' | 'studying' | 'noNewLesson';

export interface LessonRef {
  id: string;
  title: string;
}

/** A lesson for the learner: the one being studied, a completed one, or another one. */
export type LessonStatus = 'studying' | 'completed' | 'other';

/** GET /api/lessons/:id/study: the lesson's steps (none for "other") and what comes next. */
export interface LessonStudy {
  status: LessonStatus;
  steps: Partial<Record<Step, StepState>>;
  currentStep: Step | 'done' | '';
  sentenceIndex: number;
  /** The lesson to study now, once this one is completed. */
  next: LessonRef | null;
  goal: GoalView | null;
  goalCompleted: boolean;
  streak: number;
}

export interface CompletedLesson extends LessonRef {
  topicName: string;
  completedAt: string;
}

export interface MyLessons {
  /** The lesson being studied: the first one of the roadmap not completed. */
  current: LessonRef | null;
  completed: CompletedLesson[];
  upcoming: LessonRef[];
}

/** A topic as learners see it (GET /api/topics). */
export interface PublicTopic {
  id: string;
  name: string;
  level: Level;
  description: string;
  lessonCount: number;
}
