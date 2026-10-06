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
  // F22: AI check state; absent on older responses.
  /** Flags the admin has not confirmed yet. */
  flags?: number;
  checked?: boolean;
  verified?: boolean;
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
  /** Curriculum grammar point assigned to the lesson; '' when none. */
  grammarPointId?: string;
  grammarPointTitle?: string;
  /** F22: result of the AI check; null until checked, and again after any content edit. */
  review?: LessonReview | null;
}

/** Which part of the lesson a flag points at; `index` is the position in that array. */
export type FlagArea = 'sentence' | 'annotation' | 'question' | 'translation';
export type FlagKind = 'mismatch' | 'ambiguous' | 'wrong' | 'unchecked';

export interface LessonFlag {
  area: FlagArea;
  index: number;
  kind: FlagKind;
  noteVi: string;
  confirmed: boolean;
}

export interface LessonReview {
  checkedAt: string;
  verifiedAt: string | null;
  flags: LessonFlag[];
}

/** Flags of one area. */
export function flagsOf(review: LessonReview | null | undefined, area: FlagArea): LessonFlag[] {
  return (review?.flags ?? []).filter((f) => f.area === area);
}

export function openFlagCount(review: LessonReview | null | undefined): number {
  return (review?.flags ?? []).filter((f) => !f.confirmed).length;
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

/** One translation sentence of the practice, as sent to PUT .../practice/translations. */
export interface TranslationInput {
  vi: string;
  en: string;
  distractors: string[];
}

export interface LessonInput {
  title: string;
  content: string;
  topicId: string;
  source: string;
  license: string;
  /** Curriculum grammar point; '' clears it. */
  grammarPointId?: string;
  /** Add the new lesson at the end of its topic roadmap in the same request (F7). */
  appendToRoadmap?: boolean;
  /** F23: draw a picture for each vocabulary word of the new lesson, in this style. */
  images?: ImageSettingsInput;
}

/** F23: whether the vocabulary words of a lesson get an AI-drawn picture, and in which style. */
export interface ImageSettingsInput {
  enabled: boolean;
  /** How the pictures should look; '' uses the server's default style. */
  style: string;
}

/** F23: the picture settings of a lesson (GET/PUT /api/admin/lessons/{id}/images). */
export interface LessonImages extends ImageSettingsInput {
  /** '' (off), 'running', 'done' or 'failed'. */
  status: '' | 'running' | 'done' | 'failed';
  error: string;
  /** Words that have a picture. */
  count: number;
  /** The lesson's vocabulary words; empty until the annotation is done. */
  words: ImageWord[];
}

/** F23: one vocabulary word of a lesson as the admin picture list shows it. */
export interface ImageWord {
  lemma: string;
  meaningVi: string;
  /** Admin URL of the word's picture; '' when it has none. */
  imageUrl: string;
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
