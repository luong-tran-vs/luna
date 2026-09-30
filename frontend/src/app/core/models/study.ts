import { Level } from './lesson';

/** Types for the study API (specs/008-daily-study-flow/contracts/study-api.md). */

export type Step = 'review' | 'read' | 'listen';
export const STEPS: readonly Step[] = ['review', 'read', 'listen'];

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

export interface SetGoalResult {
  active: GoalView;
  effectiveFrom: string;
  /** True when today's lesson had already started: the new topic starts tomorrow. */
  startsTomorrow: boolean;
}

export type TodayKind = 'noGoal' | 'studying' | 'doneToday' | 'noNewLesson';

export interface LessonRef {
  id: string;
  title: string;
}

export interface Today {
  kind: TodayKind;
  goal: GoalView | null;
  lesson: LessonRef | null;
  steps: Record<Step, StepState>;
  currentStep: Step | 'done';
  sentenceIndex: number;
  /** Cards to review in the review step today (due cards within the daily limit). */
  reviewCount: number;
  streak: number;
  goalCompleted: boolean;
}

export interface CompletedLesson extends LessonRef {
  topicName: string;
  completedAt: string;
}

export interface MyLessons {
  today: LessonRef | null;
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
