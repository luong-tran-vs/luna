/** Types for the admin lesson API (specs/003-lesson-admin/contracts/admin-lessons-api.md). */

import { AdminPractice, PracticeStatus } from './practice';

export type Level = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export const LEVELS: readonly Level[] = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'];

export type JobStatus = 'running' | 'done' | 'failed';
/** A background job an admin can run again (POST /api/admin/lessons/{id}/retry). */
export type JobKind = 'annotate';

export interface Sentence {
  index: number;
  text: string;
}

export interface Annotation {
  text: string;
  lemma: string;
  meaningVi: string;
  sentenceIndex: number;
  editedByAdmin: boolean;
}

export interface LessonSummary {
  id: string;
  title: string;
  /** Always the level of the lesson's topic. */
  level: Level;
  topicId: string;
  topicName: string;
  annotationStatus: JobStatus;
  inRoadmap: boolean;
  createdAt: string;
}

export interface Lesson extends LessonSummary {
  content: string;
  source: string;
  license: string;
  revision: number;
  annotationError: string;
  sentences: Sentence[];
  annotations: Annotation[];
  // F15: comprehension questions, grammar note and writing prompt.
  questions: Question[];
  grammarNote: GrammarNote | null;
  writingPrompt: string;
  extrasEditedByAdmin: boolean;
  /** Bumped whenever the question set is replaced; answers belong to one version. */
  quizVersion: number;
  // F17: practice steps generated after annotation.
  practice: AdminPractice | null;
  practiceStatus: PracticeStatus;
  practiceError: string;
}

/** A multiple-choice comprehension question as admins see it. */
export interface Question {
  prompt: string;
  options: string[];
  answerIndex: number;
  explanationVi: string;
}

/** One grammar point of a lesson, with examples copied from it. */
export interface GrammarNote {
  title: string;
  bodyVi: string;
  examples: string[];
}

/** PUT /api/admin/lessons/{id}/extras. */
export interface ExtrasInput {
  questions: Question[];
  grammarNote: GrammarNote | null;
  writingPrompt: string;
}

export interface LessonInput {
  title: string;
  content: string;
  topicId: string;
  source: string;
  license: string;
  /** Add the new lesson at the end of its topic roadmap in the same request (F7). */
  appendToRoadmap?: boolean;
}

export interface AnnotationInput {
  text: string;
  lemma: string;
  meaningVi: string;
}

export interface LessonFilter {
  level?: Level | '';
  topicId?: string;
}

/** True while annotation or practice work of the lesson is still running. */
export function isRunning(
  l: Pick<LessonSummary, 'annotationStatus'> & { practiceStatus?: PracticeStatus },
): boolean {
  return l.annotationStatus === 'running' || l.practiceStatus === 'running';
}
