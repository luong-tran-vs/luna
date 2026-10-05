/** Admin side of the grammar lessons (F20): one lesson per point of the curriculum. */

export type GrammarExerciseKind = 'choice' | 'fill' | 'reorder';
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
  updatedAt: string;
  publishedAt: string | null;
  /** F21: how many exercises the AI check flagged. */
  flags: number;
  /** F21: whether the AI check has run on the current content. */
  checked: boolean;
  /** F21b: the admin confirmed the lesson as checked. */
  verified: boolean;
}

export type GrammarCheckKind = 'mismatch' | 'ambiguous' | 'unchecked';

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

export interface GrammarLessonAdmin extends Omit<GrammarLessonSummary, 'flags' | 'checked' | 'verified'> {
  content: GrammarContent;
  checks: GrammarCheck[];
  checkedAt: string | null;
  verifiedAt: string | null;
  /** Only on GET of one lesson; other responses leave it out. */
  reports?: ExerciseReports[];
}

export const CHECK_LABELS: Record<GrammarCheckKind, string> = {
  mismatch: 'Cần xem: AI giải ra đáp án khác',
  ambiguous: 'Cần xem: AI thấy câu mơ hồ',
  unchecked: 'Chưa kiểm tra được: AI không trả lời câu này',
};

/** Short reason used in the list of exercises that still need a look. */
export const CHECK_SHORT: Record<GrammarCheckKind, string> = {
  mismatch: 'AI giải ra đáp án khác',
  ambiguous: 'AI thấy câu mơ hồ',
  unchecked: 'AI không trả lời câu này',
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

/** What the admin sees for a point: "Chưa có bài", "Bản nháp", "Đã đăng", plus "đã sửa tay". */
export function grammarStatusText(lesson: Pick<GrammarLessonSummary, 'status' | 'edited'> | null | undefined): string {
  if (!lesson) {
    return 'Chưa có bài';
  }
  const base = lesson.status === 'published' ? 'Đã đăng' : 'Bản nháp';
  return lesson.edited ? `${base} · đã sửa tay` : base;
}

export const EXERCISE_KIND_LABELS: Record<GrammarExerciseKind, string> = {
  choice: 'Trắc nghiệm',
  fill: 'Điền từ',
  reorder: 'Sắp xếp câu',
};
