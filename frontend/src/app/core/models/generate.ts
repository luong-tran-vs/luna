import { Level } from './lesson';

/** Types for AI lesson generation (specs/012-ai-lesson-generation/contracts/generate-api.md). */

export type LessonKind = 'reading' | 'dialogue';

export interface GenerateInput {
  count: number;
  words: number;
  kind: LessonKind;
  idea: string;
}

export interface GeneratedDraft {
  title: string;
  content: string;
  words: number;
}

export interface GenerateResult {
  drafts: GeneratedDraft[];
  requested: number;
  dropped: number;
}

export const MIN_COUNT = 1;
export const MAX_COUNT = 5;
export const DEFAULT_COUNT = 3;
export const MIN_WORDS = 50;
export const MAX_WORDS = 800;
export const MAX_IDEA = 500;

/** Suggested words per lesson for each level. */
export const DEFAULT_WORDS: Record<Level, number> = {
  A1: 120,
  A2: 160,
  B1: 220,
  B2: 300,
  C1: 400,
  C2: 400,
};

/** Source and license of lessons saved from AI drafts. */
export const AI_SOURCE = 'AI sinh';
export const AI_LICENSE = 'Nội dung do AI tạo';

/** Words separated by whitespace, as the server counts them. */
export function countWords(text: string): number {
  const trimmed = text.trim();
  return trimmed ? trimmed.split(/\s+/).length : 0;
}

/** The accepted word range (±20%) for a target length. */
export function wordRange(words: number): { min: number; max: number } {
  return { min: Math.ceil(words * 0.8), max: Math.floor(words * 1.2) };
}
