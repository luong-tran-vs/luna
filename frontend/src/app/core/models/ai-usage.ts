/** Requests and tokens of the AI provider over some time. */
export interface AiTotals {
  requests: number;
  errors: number;
  /** Requests refused because a limit was reached (HTTP 429). */
  quota: number;
  promptTokens: number;
  outputTokens: number;
  totalTokens: number;
}

export interface AiModelTotals extends AiTotals {
  model: string;
}

export interface AiMinute extends AiTotals {
  start: string;
}

export interface AiOpTotals extends AiTotals {
  op: string;
}

export interface AiCall {
  at: string;
  model: string;
  op: string;
  outcome: 'ok' | 'quota' | 'error';
  status: number;
  promptTokens: number;
  outputTokens: number;
  thoughtTokens: number;
  totalTokens: number;
  durationMs: number;
}

/** GET /api/admin/ai-usage. */
export interface AiUsage {
  now: string;
  /** The 60 seconds before now, per model. */
  lastMinute: AiModelTotals[];
  /** The busiest minute of the last hour. */
  peakMinute: AiTotals;
  /** The 60 minutes before now, oldest first. */
  minutes: AiMinute[];
  /** Midnight Pacific time, when Google resets the daily limits. */
  dayStart: string;
  today: AiModelTotals[];
  /** The last 7 days per operation, most requests first. */
  week: AiOpTotals[];
  /** The last requests, newest first. */
  recent: AiCall[];
}
