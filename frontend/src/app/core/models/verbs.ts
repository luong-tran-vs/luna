/** The learner's practice of the irregular verbs: GET /api/verbs/progress, POST /api/verbs/results. */

/** One verb the learner has answered, named by its base form. */
export interface VerbProgress {
  base: string;
  /** 0: not known yet; 1 and up: known, each box asked again later than the one before. */
  level: number;
  /** When it is asked again (RFC 3339); null for a verb never answered right. */
  dueAt: string | null;
  right: number;
  wrong: number;
}

export interface VerbProgressList {
  verbs: VerbProgress[];
}

/** One verb of a finished round: right when V2 and V3 were both right the first time asked. */
export interface VerbResult {
  base: string;
  right: boolean;
}
