import { ImageEngine } from './word-bank';

/** Types of the verb pictures (added 2026-11-10): one picture per irregular verb, of its verb sense. */

/** A verb that has a picture. imageUrl changes when the picture does. */
export interface VerbImage {
  base: string;
  imageUrl: string;
  updatedAt: string;
}

/** A word bank picture of a word spelt like the verb, which the admin may reuse if it shows the verb. */
export interface VerbBankPicture {
  base: string;
  meaningVi: string;
  imageUrl: string;
}

/** GET /api/admin/verb-images. */
export interface VerbImageList {
  images: VerbImage[];
  bank: VerbBankPicture[];
  codexImages: boolean;
}

/** A verb to draw: the meaning and the example sentence keep the picture on the verb's sense. */
export interface VerbDrawItem {
  base: string;
  meaningVi: string;
  sentence: string;
}

/** POST /api/admin/verb-image-batch. The engine is required: there is no default. */
export interface VerbImageBatchRequest {
  items: VerbDrawItem[];
  engine: ImageEngine;
}

export interface VerbImageFailure {
  base: string;
  message: string;
}

/** How the verb picture batch is going. */
export interface VerbImageBatch {
  running: boolean;
  stopping: boolean;
  stopped: boolean;
  engine: ImageEngine | '';
  total: number;
  /** Verbs dealt with, with or without a picture. */
  done: number;
  drawing: string[];
  saved: string[];
  failures: VerbImageFailure[];
  /** Why the batch ended early by itself (no quota, Codex not logged in, …). */
  stopMessage?: string;
  codexImages: boolean;
  startedAt?: string;
  finishedAt?: string;
}
