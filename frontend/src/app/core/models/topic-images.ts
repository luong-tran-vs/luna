import { ImageEngine } from './word-bank';

/** The picture of a topic (added 2026-11-16): one banner per topic, shared by every level. */
export interface TopicImage {
  topicId: string;
  imageUrl: string;
  updatedAt: string;
}

/** GET /api/topic-images (learners) and /api/admin/topic-images. */
export interface TopicImageList {
  images: TopicImage[];
  /** Admin list only: whether this server can draw with the Codex CLI. */
  codexImages?: boolean;
}

export interface TopicImageBatchRequest {
  topicIds: string[];
  engine: ImageEngine;
}

export interface TopicImageFailure {
  topicId: string;
  message: string;
}

/** How the topic picture batch is going. */
export interface TopicImageBatch {
  running: boolean;
  stopping: boolean;
  stopped: boolean;
  engine: ImageEngine | '';
  total: number;
  /** Topics dealt with, with or without a picture. */
  done: number;
  drawing: string[];
  saved: string[];
  failures: TopicImageFailure[];
  /** Why the batch ended early by itself (no quota, Codex not logged in, …). */
  stopMessage?: string;
  codexImages: boolean;
  startedAt?: string;
  finishedAt?: string;
}
