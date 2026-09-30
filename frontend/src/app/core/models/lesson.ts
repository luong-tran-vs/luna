/** Types for the admin lesson API (specs/003-lesson-admin/contracts/admin-lessons-api.md). */

export type Level = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export const LEVELS: readonly Level[] = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'];

export type JobStatus = 'running' | 'done' | 'failed';
export type JobKind = 'tts' | 'annotate';

export interface Sentence {
  index: number;
  text: string;
  audioUrl: string | null;
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
  audioStatus: JobStatus;
  annotationStatus: JobStatus;
  inRoadmap: boolean;
  createdAt: string;
}

export interface Lesson extends LessonSummary {
  content: string;
  source: string;
  license: string;
  revision: number;
  audioError: string;
  annotationError: string;
  sentences: Sentence[];
  annotations: Annotation[];
}

export interface LessonInput {
  title: string;
  content: string;
  topicId: string;
  source: string;
  license: string;
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

/** True while audio or annotation work of the lesson is still running. */
export function isRunning(l: Pick<LessonSummary, 'audioStatus' | 'annotationStatus'>): boolean {
  return l.audioStatus === 'running' || l.annotationStatus === 'running';
}
