/** The learner's irregular verbs: GET /api/verbs/progress, POST /api/verbs/learned, POST /api/verbs/results. */

/** One verb the learner has learned or answered, named by its base form. */
export interface VerbProgress {
  base: string;
  /** When the learner confirmed it as learned ("Đã học", RFC 3339); null when not learned. */
  learnedAt: string | null;
  /** Review box of a learned verb, from 1; 0 when not learned. */
  level: number;
  /** When a learned verb is reviewed again (RFC 3339); null when not learned. */
  dueAt: string | null;
  right: number;
  wrong: number;
}

export interface VerbProgressList {
  verbs: VerbProgress[];
}

/** One verb of a finished review round: right when V2 and V3 were both right the first time asked. */
export interface VerbResult {
  base: string;
  right: boolean;
}
