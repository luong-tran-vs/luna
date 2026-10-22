import { Level } from './lesson';

/** Types for the study API (specs/008-daily-study-flow/contracts/study-api.md). */

/**
 * The steps of a lesson, all recorded on the server since the six-step lesson (2026-10-22):
 * Từ vựng → Nghe → Đọc → Ngữ pháp → Nói → Viết. A lesson completes when every one is done.
 */
export type Step = 'vocab' | 'listen' | 'read' | 'grammar' | 'speak' | 'write';
export const STEPS: readonly Step[] = ['vocab', 'listen', 'read', 'grammar', 'speak', 'write'];

/** The name of each step, as the lesson page and the home page show it. */
export const STEP_LABELS: Record<Step, string> = {
  vocab: 'Từ vựng',
  listen: 'Nghe',
  read: 'Đọc',
  grammar: 'Ngữ pháp',
  speak: 'Nói',
  write: 'Viết',
};

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

/**
 * Where the learner stands in the goal's roadmap. membersOnly: a guest finished the first lesson;
 * the next ones are for members.
 */
export type StudyKind = 'noGoal' | 'studying' | 'noNewLesson' | 'membersOnly';

export interface LessonRef {
  id: string;
  title: string;
  /** An upcoming lesson a guest cannot study. */
  membersOnly?: boolean;
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
  /** The learner is a guest who finished the first lesson: the next ones are for members. */
  membersOnly?: boolean;
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
  /** Place in the suggested order of the level (the list comes sorted); 0 when it has none. */
  position: number;
}

/** How a lesson shows on the learning path; membersOnly: a guest cannot study it. */
export type PathLessonState = 'done' | 'current' | 'locked' | 'membersOnly';

export interface PathLesson {
  id: string;
  title: string;
  state: PathLessonState;
  /**
   * An extra lesson of a course topic, after its main lessons: studied by default, but the next
   * topic may start once the main ones are done. Missing means false.
   */
  extra?: boolean;
}

/** A topic's place in the level's fixed course (giáo trình). */
export interface PathCourse {
  /** 1-based number of the topic in the course. */
  number: number;
  /** What the learner can do after the topic ("Sau chủ đề này bạn có thể …"). */
  canDoVi: string;
  /** The grammar points the topic teaches, in Vietnamese. */
  grammarVi: string[];
}

/** One topic's roadmap within a milestone; only the goal's (active) has a lesson to study now. */
export interface PathRoadmap {
  topicId: string;
  topicName: string;
  /** The topic's description, shown under its name in the part's banner; may be empty. */
  topicDescription?: string;
  /** The topic's place in the level's course; null (or missing) when it is not in one. */
  course?: PathCourse | null;
  active: boolean;
  completed: number;
  lessons: PathLesson[];
}

/** One level ("Phần"): every topic with lessons at that level, one roadmap after another. */
export interface Milestone {
  level: Level;
  active: boolean;
  completed: number;
  total: number;
  roadmaps: PathRoadmap[];
}

/** GET /api/lessons/path: the "Học" tab. topicId is the goal's topic, '' when there is none. */
export interface LearningPath {
  topicId: string;
  topicName: string;
  milestones: Milestone[];
  streak: number;
}
