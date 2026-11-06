/** Types for the admin lesson API (specs/003-lesson-admin/contracts/admin-lessons-api.md). */

import { AdminPractice, PracticeStatus } from './practice';
import { ImageEngine } from './word-bank';

export type Level = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export const LEVELS: readonly Level[] = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'];

/** Vietnamese name of each level. */
export const LEVEL_NAMES: Record<Level, string> = {
  A1: 'Mới bắt đầu',
  A2: 'Sơ cấp',
  B1: 'Trung cấp',
  B2: 'Trung cấp cao',
  C1: 'Nâng cao',
  C2: 'Thành thạo',
};

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
  /** Part of speech as used in the sentence; "" or missing for lessons annotated before it was asked. */
  pos?: string;
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
  /** Hidden from learners until an admin publishes it (saved AI drafts start hidden). */
  draft?: boolean;
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
  /** Other right answers, each made of the tiles (en's words and the distractors). */
  accepted: string[];
}

export interface LessonInput {
  title: string;
  content: string;
  topicId: string;
  /** The lesson's own level; a topic is shared by every level. */
  level: Level;
  source: string;
  license: string;
  /** Curriculum grammar point; '' clears it. */
  grammarPointId?: string;
  /** Add the new lesson at the end of its topic roadmap in the same request (F7). */
  appendToRoadmap?: boolean;
  /** F18: the topic words a generated draft was asked to use; they become the lesson's whole vocabulary. */
  targetWords?: string[];
  /** Save the new lesson hidden from learners until it is published. */
  draft?: boolean;
}

/**
 * F23: how the AI draws the lesson's scene picture. Since 2026-11-06 the AI draws only the scene
 * of a lesson; word pictures are drawn in the word bank.
 */
export interface ImageSettingsInput {
  /** What the scene picture shows, in English; '' lets the AI choose from the story. */
  sceneDescription: string;
  /** How the picture should look; '' uses the server's default style. */
  style: string;
}


/** F23: the picture settings of a lesson (GET/PUT /api/admin/lessons/{id}/images). */
export interface LessonImages extends ImageSettingsInput {
  /** The server can draw with the Codex CLI; the engine choice is shown only then. */
  codexImages?: boolean;
  /** Words that have a picture of their own. */
  count: number;
  /** Words without their own picture that show the word bank's (added 2026-11-05). */
  bankCount?: number;
  /** The lesson's vocabulary words; empty until the annotation is done. */
  words: ImageWord[];
  /** Admin URL of the scene picture; '' when it has none (added 2026-11-05). */
  sceneImageUrl: string;
}

/** F23: one vocabulary word of a lesson as the admin picture list shows it. */
export interface ImageWord {
  lemma: string;
  meaningVi: string;
  /** Admin URL of the word's picture; '' when it has none. */
  imageUrl: string;
  /** The picture shown is the word bank's: the word has none of its own. */
  fromBank?: boolean;
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

/**
 * F22: the AI's corrected version of one flagged item (POST …/check/suggest), or the admin's
 * correction to apply (POST …/check/apply). Only the fields of its area are set.
 */
export interface FixSuggestion {
  area: FlagArea;
  index: number;
  /** A sentence. */
  text?: string;
  /** An annotation's meaning. */
  meaningVi?: string;
  question?: Question;
  /** A translation pair. */
  vi?: string;
  en?: string;
  /** What the AI changed and why (suggestions only). */
  noteVi?: string;
}

/** A lesson of a scene picture batch; message says why it got no picture. */
export interface SceneBatchLesson {
  id: string;
  title: string;
  message?: string;
}

/**
 * The scene picture batch of the course page (GET /api/admin/lessons/scene-batch, added
 * 2026-11-06): the lessons of a topic without a scene picture, drawn one after another.
 */
export interface SceneBatch {
  running: boolean;
  /** The admin asked to stop: the lesson being drawn finishes, no new one starts. */
  stopping: boolean;
  stopped: boolean;
  engine: ImageEngine | '';
  /** Which topic the batch is for, e.g. "A1-3", and its name. */
  key: string;
  label: string;
  total: number;
  done: number;
  drawing: SceneBatchLesson | null;
  saved: SceneBatchLesson[];
  failures: SceneBatchLesson[];
  /** Lessons not drawn at all (not annotated yet). */
  skipped: SceneBatchLesson[];
  /** Why the batch ended early by itself (no quota, AI not set up…). */
  stopMessage?: string;
  /** The server can draw with the Codex CLI. */
  codexImages: boolean;
  startedAt?: string;
  finishedAt?: string;
}

export interface SceneBatchRequest {
  lessonIds: string[];
  /** Draw the lessons that have a scene too, replacing it (the lesson page). */
  replace?: boolean;
  engine: ImageEngine;
  key: string;
  label: string;
}
