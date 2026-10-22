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
  grammar: CurriculumGrammar[];
  slots: CurriculumSlot[];
  /** Lessons of the topic at this level that fill no slot. */
  others: CurriculumLesson[];
}

/** The admin's view of the fixed course of a level (GET /api/admin/curriculum). */
export interface CurriculumBoard {
  level: string;
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
