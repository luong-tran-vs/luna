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

/** "Động từ to be (2 bài)" for a select option. */
export function grammarOptionLabel(p: GrammarPoint): string {
  return `${p.titleVi} (${p.lessonCount} bài)`;
}
