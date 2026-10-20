/** Grammar points of the curriculum (GET /api/admin/grammar). */
export interface GrammarPoint {
  id: string;
  level: string;
  titleVi: string;
  titleEn: string;
  pattern: string;
  hintVi: string;
  examples: string[];
  /** Lessons (of the asked topic, or of all topics) that already use this point. */
  lessonCount: number;
}

/** Sent as grammarPointId when generating: the AI picks the point of each lesson (added 2026-10-20). */
export const GRAMMAR_AUTO = 'auto';

/** "Động từ to be (2 bài)" for a select option. */
export function grammarOptionLabel(p: GrammarPoint): string {
  return `${p.titleVi} (${p.lessonCount} bài)`;
}
