/** The Gemini model that writes text, chosen on the admin AI model page (GET /api/admin/ai-model). */
export interface AiModelState {
  /** The model in use. */
  model: string;
  /** The model of GEMINI_MODEL, used while the admin chose nothing. */
  default: string;
  /** The model the admin chose; empty for the default. */
  saved: string;
  updatedAt: string | null;
  /** The models the admin can choose, from the least to the most capable. */
  options: string[];
  /** The text requests a model can be chosen for (op names, as on the AI usage page). */
  ops: string[];
  /** The ops given their own model; the others use `model`. */
  byOp: Record<string, string>;
}

/** How a test request went: answered, no answer within the time limit, out of requests (429), overloaded (503). */
export type AiModelOutcome = 'ok' | 'timeout' | 'quota' | 'busy' | 'error';

export interface AiModelTest {
  outcome: AiModelOutcome;
  durationMs: number;
}
