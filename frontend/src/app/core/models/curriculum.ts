import { JobStatus } from './lesson';

/** A grammar point of the fixed course; published tells whether its grammar lesson is out. */
export interface CurriculumGrammar {
  /** Grammar point id, e.g. "a1-to-be" (the admin page is /admin/grammar/:id). */
  id: string;
  titleVi: string;
  published: boolean;
}

/** The lesson that fills a slot, or one of the topic outside the course. */
export interface CurriculumLesson {
  id: string;
  title: string;
  draft: boolean;
  annotationStatus: JobStatus;
  /** Where the lesson comes from, as saved ("AI sinh", "Nhập"…); added 2026-10-26. */
  source?: string;
}

/**
 * The key of a rule checked on a generated draft: ok passed; warn only warns (the admin can still
 * publish); neither is a failure (better generate it again).
 */
export type CheckKey =
  | 'grammar'
  | 'otherPerson'
  | 'negative'
  | 'question'
  | 'targetWords'
  | 'length'
  | 'dialogue'
  | 'level';

export interface CheckItem {
  key: CheckKey;
  ok: boolean;
  warn: boolean;
  textVi: string;
}

/** A word of the lesson above its level, e.g. festival · B1. */
export interface AboveWord {
  text: string;
  level: string;
}

export interface LessonChecks {
  items: CheckItem[];
  above: AboveWord[];
}

export type SlotKind = 'main' | 'extra';

/** One planned lesson of a topic: empty (lesson null), a draft or published. */
export interface CurriculumSlot {
  key: string;
  kind: SlotKind;
  title: string;
  /** What the lesson is about, the brief given to the AI. */
  idea: string;
  grammar: CurriculumGrammar[];
  lesson: CurriculumLesson | null;
  /** Only for a draft. */
  checks: LessonChecks | null;
}

export interface CurriculumTopic {
  number: number;
  name: string;
  canDoVi: string;
  /** '' when the topic is not in the catalogue yet (the course is not applied). */
  topicId: string;
  position: number;
  /** Words of the catalogue topic a lesson of the level may be given (target words come from them). */
  words: number;
  grammar: CurriculumGrammar[];
  slots: CurriculumSlot[];
  /** Lessons of the topic at this level that fill no slot. */
  others: CurriculumLesson[];
}

/**
 * Where a level's course is: none yet, a draft the admin edits (written by the AI), or approved
 * (learners and lesson generation use it).
 */
export type CurriculumStatus = 'none' | 'draft' | 'approved';

/**
 * The admin's view of the course of a level (GET /api/admin/curriculum): its draft while there is
 * one, otherwise the approved course.
 */
export interface CurriculumBoard {
  level: string;
  status: CurriculumStatus;
  /** The level has an approved course (also while a draft edits it). */
  hasApproved: boolean;
  /** What the course lacks: too few topics or extra lessons, grammar points no topic teaches. */
  warnings: string[];
  /** Grammar points of the level, for editing a draft. */
  points: CurriculumGrammar[];
  /** Topics exist and are ordered as the course says. */
  applied: boolean;
  /** Number of main and extra slots. */
  main: number;
  extra: number;
  /** Slots filled by a published lesson, and by a draft. */
  published: number;
  drafts: number;
  topics: CurriculumTopic[];
}

/** A lesson of a draft as the admin saves it; its key and kind follow from its place. */
export interface CurriculumPlanSlot {
  title: string;
  idea: string;
  /** Grammar point ids, the first being the one the lesson is built on. */
  grammar: string[];
}

/** A topic of a draft as the admin saves it: its first 3 lessons are the main ones. */
export interface CurriculumPlanTopic {
  /** The catalogue's topic, '' for a new name (created on approval). */
  topicId: string;
  name: string;
  canDoVi: string;
  grammar: string[];
  slots: CurriculumPlanSlot[];
}

/** The draft of a level's course as the admin saves it (PUT /api/admin/curriculum/draft). */
export interface CurriculumPlan {
  topics: CurriculumPlanTopic[];
}

/** Main lessons of every topic; what a course should hold besides. */
export const MAIN_PER_TOPIC = 3;
/** Extra lessons a topic should have at most (4–5). */
export const MAX_EXTRA = 5;
export const MIN_TOPICS = 7;

/** The draft of a board, as the admin edits it. */
export function planOf(b: CurriculumBoard): CurriculumPlan {
  return {
    topics: b.topics.map((t) => ({
      topicId: t.topicId,
      name: t.name,
      canDoVi: t.canDoVi,
      grammar: t.grammar.map((g) => g.id),
      slots: t.slots.map((s) => ({
        title: s.title,
        idea: s.idea,
        grammar: s.grammar.map((g) => g.id),
      })),
    })),
  };
}

/** A catalogue topic in the approved course of a level (GET /api/admin/curriculum/topics). */
export interface CourseTopicRef {
  topicId: string;
  name: string;
  /** Its place in the course, from 1. */
  number: number;
}

/** Where the course of a level is, for the admin home (GET /api/admin/curriculum/summary). */
export interface CurriculumLevelSummary {
  level: string;
  status: CurriculumStatus;
  /** A draft edits or writes the course. */
  hasDraft: boolean;
  /** Of the approved course: the order is applied, slots without a lesson, slots with a draft. */
  applied: boolean;
  empty: number;
  drafts: number;
}

/** The source of a lesson imported instead of generated (FR-027). */
export const IMPORT_SOURCE = 'Nhập';

/** A grammar point of a slot as the writer of its lesson needs it. */
export interface SlotBriefGrammar {
  id: string;
  titleVi: string;
  titleEn: string;
  pattern: string;
  hintVi: string;
  examples: string[];
}

/** GET …/slots/{key}/brief (FR-026): what the lesson of a slot is written from. */
export interface SlotBrief {
  key: string;
  level: string;
  topicId: string;
  topicName: string;
  topicNumber: number;
  canDoVi: string;
  kind: SlotKind;
  title: string;
  idea: string;
  grammar: SlotBriefGrammar[];
  targetWords: string[];
  topicWords: string[];
  /** About how many words the lesson should have. */
  words: number;
  minGrammarUses: number;
  filled: boolean;
}

/** What the checks kept of one part of an imported lesson, and why some was dropped. */
export interface ImportPart {
  key: string;
  labelVi: string;
  given: number;
  kept: number;
  ruleVi?: string;
  reasons?: string[];
}

export interface ImportReport {
  sentences: number;
  fromDictionary: number;
  parts: ImportPart[];
}

/** POST …/slots/{key}/import (FR-027); the lesson of the slot is null on a dry run. */
export interface SlotImportResult {
  dryRun: boolean;
  slot: CurriculumSlot;
  checks: LessonChecks;
  report: ImportReport;
}
