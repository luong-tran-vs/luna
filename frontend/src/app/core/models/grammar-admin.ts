/** Admin side of the grammar lessons (F20): one lesson per point of the curriculum. */
import { ImageEngine } from './word-bank';

export type GrammarExerciseKind = 'choice' | 'fill' | 'reorder' | 'fix' | 'picture';
export type GrammarLessonStatus = 'draft' | 'published';

export interface GrammarExercise {
  id: string;
  kind: GrammarExerciseKind;
  promptVi?: string;
  text?: string;
  options?: string[];
  answerIndex?: number;
  answers?: string[];
  sentence?: string;
  words?: string[];
  /** A picture exercise (added 2026-11-06): what its picture shows, in English, and the picture once drawn. */
  picture?: string;
  imageUrl?: string;
  explanationVi: string;
}

export interface GrammarStructure {
  label: string;
  pattern: string;
  example: string;
}

export interface GrammarExample {
  en: string;
  vi: string;
  /** 1-based number of the structure the sentence shows; absent on lessons written before 2026-10-30. */
  structure?: number;
  /** `en` with the pieces of that pattern marked "{n:words}" ("{1:She} {2:is} {3:reading} a book."). */
  marked?: string;
}

export interface GrammarMistake {
  wrong: string;
  right: string;
  noteVi: string;
}

export interface GrammarContent {
  objective: string;
  explanation: string[];
  usage: string[];
  structures: GrammarStructure[];
  examples: GrammarExample[];
  mistakes: GrammarMistake[];
  practice: GrammarExercise[];
  mastery: GrammarExercise[];
}

/** A row of GET /api/admin/grammar-lessons (only points that have a lesson). */
export interface GrammarLessonSummary {
  pointId: string;
  status: GrammarLessonStatus;
  edited: boolean;
  /** Imported instead of generated (FR-028); absent from older responses. */
  imported?: boolean;
  updatedAt: string;
  publishedAt: string | null;
  /** F21: how many exercises the AI check flagged. */
  flags: number;
  /** F21: whether the AI check has run on the current content. */
  checked: boolean;
  /** F21b: the admin confirmed the lesson as checked. */
  verified: boolean;
  /** The "Nhìn ảnh chọn từ" exercises of a point that can be drawn, and how many have their picture (added 2026-11-06). */
  pictures?: number;
  picturesDrawn?: number;
  /** Practice and mastery both have their picture exercises. */
  picturesComplete?: boolean;
}

/** "meaning" (added 2026-10-20): the Vietnamese does not say what the English answer says. */
export type GrammarCheckKind = 'mismatch' | 'ambiguous' | 'unchecked' | 'meaning';

/** F21: an exercise the independent AI solver flagged. */
export interface GrammarCheck {
  exerciseId: string;
  kind: GrammarCheckKind;
  noteVi: string;
  /** The admin said this exercise is right as it is. */
  confirmed: boolean;
}

export type GrammarReportReason = 'wrong_answer' | 'ambiguous' | 'typo' | 'other';

/** F21: open learner reports of one exercise (inside a lesson). */
export interface ExerciseReports {
  exerciseId: string;
  count: number;
  reasons: Partial<Record<GrammarReportReason, number>>;
  notes: string[];
}

/** F21: a row of GET /api/admin/grammar-reports, grouped by (point, exercise). */
export interface GrammarReportGroup extends ExerciseReports {
  pointId: string;
  latestAt: string;
}

export interface GrammarLessonAdmin extends Omit<
  GrammarLessonSummary,
  'flags' | 'checked' | 'verified'
> {
  content: GrammarContent;
  checks: GrammarCheck[];
  checkedAt: string | null;
  verifiedAt: string | null;
  /** The picture of each example of the content, in order; '' for none (added 2026-11-06). */
  exampleImages?: string[];
  /** The server can draw with the Codex CLI. */
  codexImages?: boolean;
  /** Only on GET of one lesson; other responses leave it out. */
  reports?: ExerciseReports[];
}

export const CHECK_LABELS: Record<GrammarCheckKind, string> = {
  mismatch: 'Cần xem: AI giải ra đáp án khác',
  ambiguous: 'Cần xem: AI thấy câu mơ hồ',
  unchecked: 'Chưa kiểm tra được',
  meaning: 'Cần xem: tiếng Việt không khớp nghĩa đáp án',
};

/** Short reason used in the list of exercises that still need a look. */
export const CHECK_SHORT: Record<GrammarCheckKind, string> = {
  mismatch: 'AI giải ra đáp án khác',
  ambiguous: 'AI thấy câu mơ hồ',
  unchecked: 'AI chưa kiểm tra được câu này',
  meaning: 'Tiếng Việt lệch nghĩa đáp án',
};

/** The flags nobody confirmed yet. */
export function openChecks(lesson: Pick<GrammarLessonAdmin, 'checks'>): GrammarCheck[] {
  return lesson.checks.filter((c) => !c.confirmed);
}

/** The body of PUT …/exercises/{id}: an exercise without its id. */
export type GrammarExerciseInput = Omit<GrammarExercise, 'id'>;

export const REPORT_REASON_LABELS: Record<GrammarReportReason, string> = {
  wrong_answer: 'Đáp án sai',
  ambiguous: 'Câu mơ hồ',
  typo: 'Lỗi chính tả',
  other: 'Khác',
};

/** "Đáp án sai: 2 · Lỗi chính tả: 1" for the reasons of a report group. */
export function reasonsText(reasons: Partial<Record<GrammarReportReason, number>>): string {
  return (Object.keys(REPORT_REASON_LABELS) as GrammarReportReason[])
    .filter((r) => (reasons[r] ?? 0) > 0)
    .map((r) => `${REPORT_REASON_LABELS[r]}: ${reasons[r]}`)
    .join(' · ');
}

/** The strip above the preview: not checked / N flagged / checked clean. */
export function checkStatusText(lesson: Pick<GrammarLessonAdmin, 'checks' | 'checkedAt'>): string {
  if (!lesson.checkedAt) {
    return 'Chưa kiểm tra';
  }
  const open = openChecks(lesson).length;
  if (open > 0) {
    return `${open} câu cần xem`;
  }
  return lesson.checks.length > 0
    ? 'Đã kiểm tra, các câu bị gắn cờ đã được xác nhận'
    : 'Đã kiểm tra, không có câu nào bị gắn cờ';
}

/**
 * What the admin sees for a point: "Chưa có bài", "Bản nháp", "Đã đăng", plus "nhập" for an imported
 * lesson or "đã sửa tay".
 */
export function grammarStatusText(
  lesson: Pick<GrammarLessonSummary, 'status' | 'edited' | 'imported'> | null | undefined,
): string {
  if (!lesson) {
    return 'Chưa có bài';
  }
  const base = lesson.status === 'published' ? 'Đã đăng' : 'Bản nháp';
  if (lesson.imported) {
    return `${base} · nhập`;
  }
  return lesson.edited ? `${base} · đã sửa tay` : base;
}

/** POST …/grammar-lessons/{pointId}/import (FR-028); replaced tells whether a lesson was there. */
export interface GrammarImportResult {
  dryRun: boolean;
  replaced: boolean;
  lesson: GrammarLessonAdmin;
}

export const EXERCISE_KIND_LABELS: Record<GrammarExerciseKind, string> = {
  choice: 'Trắc nghiệm',
  fill: 'Điền từ',
  reorder: 'Sắp xếp câu',
  fix: 'Tìm và sửa lỗi',
  picture: 'Nhìn ảnh chọn từ',
};

/** An example of a grammar picture batch; message says why it got no picture. */
export interface GrammarImageBatchItem {
  pointId: string;
  key: string;
  en: string;
  message?: string;
}

/**
 * The batch drawing the pictures of grammar examples in the background (GET
 * /api/admin/grammar-image-batch, added 2026-11-06): of one lesson, or of every lesson of a level.
 */
export interface GrammarImageBatch {
  running: boolean;
  stopping: boolean;
  stopped: boolean;
  engine: ImageEngine | '';
  /** What the batch is for: "point:<id>" or "level:A1", and its name. */
  key: string;
  label: string;
  /** Pictures to draw, and the lessons they come from. */
  total: number;
  points: number;
  done: number;
  /** The examples being drawn now: several at once with Codex (a list since 2026-11-11). */
  drawing: GrammarImageBatchItem[];
  saved: GrammarImageBatchItem[];
  failures: GrammarImageBatchItem[];
  stopMessage?: string;
  codexImages: boolean;
  startedAt?: string;
  finishedAt?: string;
}

export interface GrammarImageBatchRequest {
  pointIds: string[];
  /** Only these examples (one lesson): "Vẽ lại" of one example. */
  keys?: string[];
  /** Draw the examples that have a picture too, replacing it. */
  replace?: boolean;
  engine: ImageEngine;
  key: string;
  label: string;
}

/** A lesson of a picture exercise batch: how many it got, or why it got none (message). */
export interface GrammarPictureBatchItem {
  pointId: string;
  title: string;
  added: number;
  message?: string;
}

/**
 * The batch adding "Nhìn ảnh chọn từ" exercises to the lessons of a level in the background (GET
 * /api/admin/grammar-picture-batch, added 2026-11-06). total and done count lessons.
 */
export interface GrammarPictureBatch {
  running: boolean;
  stopping: boolean;
  stopped: boolean;
  /** What the batch is for ("level:A1") and its name. */
  key: string;
  label: string;
  total: number;
  done: number;
  current: GrammarPictureBatchItem | null;
  saved: GrammarPictureBatchItem[];
  failures: GrammarPictureBatchItem[];
  stopMessage?: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface GrammarPictureBatchRequest {
  pointIds: string[];
  key: string;
  label: string;
}

/** How many picture exercises a lesson of a point that can be drawn has in practice and mastery. */
export const PICTURE_PRACTICE = 3;
export const PICTURE_MASTERY = 1;

/**
 * What removing the pictures of grammar lessons takes away, or would with dryRun (added
 * 2026-11-12): every picture, of how many lessons, and how many each lesson has.
 */
export interface GrammarImagesRemoved {
  pictures: number;
  points: number;
  byPoint: { pointId: string; pictures: number }[];
}
