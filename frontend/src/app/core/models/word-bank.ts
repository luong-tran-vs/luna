/** F24: one word of the shared word bank, as the admin API returns it. */
export interface BankWord {
  lemma: string;
  meaningVi: string;
  ipa: string;
  /** "" when the word has no picture; changes when the picture does. */
  imageUrl: string;
  updatedAt: string;
  /** The topics whose word list holds the word, by name. */
  topics: BankTopic[];
  /**
   * The standard IPA (General American) offered when the stored one looks wrong or writes other
   * sounds; absent otherwise.
   */
  ipaSuggestion?: string;
}

/** A topic named on a word of the bank. */
export interface BankTopic {
  id: string;
  name: string;
}

/** One page of the word bank list. */
export interface BankPage {
  words: BankWord[];
  total: number;
  hasMore: boolean;
  /** This server can draw pictures with the Codex CLI (CODEX_IMAGES=true, added 2026-11-04). */
  codexImages?: boolean;
}

/** What draws a word's picture: Gemini, or the Codex CLI of the server (added 2026-11-04). */
export type ImageEngine = 'gemini' | 'codex';

/** The name of an engine as the admin pages show it. */
export function engineLabel(engine: ImageEngine): string {
  return engine === 'codex' ? 'Codex CLI' : 'Gemini';
}

/** Where the engine choice is kept on this browser, shared by the word bank and the lesson pictures. */
const ENGINE_KEY = 'luna.admin.imageEngine';

/** The engine last chosen on this browser; Gemini by default. */
export function storedImageEngine(): ImageEngine {
  try {
    return localStorage.getItem(ENGINE_KEY) === 'codex' ? 'codex' : 'gemini';
  } catch {
    return 'gemini';
  }
}

/** Remembers the engine chosen; private mode or blocked storage keeps it only until the page closes. */
export function storeImageEngine(engine: ImageEngine): void {
  try {
    localStorage.setItem(ENGINE_KEY, engine);
  } catch {
    // The choice lasts until the page closes.
  }
}

/** Filter of the list: every word, the words lacking one thing, or those whose IPA has a suggestion. */
export type BankMissing = '' | 'image' | 'ipa' | 'meaning' | 'ipaCheck';

export interface BankWordInput {
  lemma: string;
  meaningVi: string;
  ipa: string;
  /** The topic whose word list also gets the word; "" for none. */
  topicId: string;
}

/** What adding a word did: `inBank` when it was in the bank already and only went into `topic`. */
export interface BankWordAdded extends BankWord {
  inBank: boolean;
  topic: BankTopic | null;
}

/** What filling the meanings and IPA of a topic with the AI did. */
export interface BankMeaningsFilled {
  /** Words lacking a meaning or an IPA sent to the AI (0: nothing to fill, no request). */
  asked: number;
  meanings: number;
  ipas: number;
}

export interface BankWordDetails {
  meaningVi?: string;
  ipa?: string;
}

/** Limits of the server (internal/wordbank). */
export const MAX_BANK_LEMMA = 100;
export const MAX_BANK_MEANING = 300;
export const MAX_BANK_IPA = 100;

/** How many bank words of one topic lack a picture, an IPA and a meaning (topics lacking nothing are absent). */
export interface BankTopicMissing {
  topicId: string;
  image: number;
  ipa: number;
  meaning: number;
}

/** Most words one picture batch draws (internal/wordbank MaxBatch). */
export const MAX_BANK_BATCH = 100;

/**
 * Starts a picture batch (added 2026-11-04): the ticked `lemmas`, or, without any, the words
 * lacking a picture that match `q` and `topicId`.
 */
export interface BankBatchRequest {
  lemmas?: string[];
  q?: string;
  topicId?: string;
  engine: ImageEngine;
}

/** A word of the batch that got no picture, with the server's Vietnamese reason. */
export interface BankBatchFailure {
  lemma: string;
  message: string;
}

/** How the picture batch drawing in the background is going; `total` 0 when none ran. */
export interface BankImageBatch {
  running: boolean;
  /** The admin asked to stop: the words being drawn finish, no new one starts. */
  stopping: boolean;
  /** The batch ended because the admin stopped it. */
  stopped: boolean;
  engine: ImageEngine | '';
  /** The admin ticked the words; otherwise they are the words lacking a picture of the filter. */
  ticked: boolean;
  /** The filter's topic, absent for every topic. */
  topicName?: string;
  /** The filter's search text. */
  search?: string;
  /** The topics holding the words of the batch, by name. */
  topics: string[];
  total: number;
  /** Words dealt with, with or without a picture. */
  done: number;
  drawing: string[];
  /** The words that got their picture, in order. */
  saved: BankWord[];
  failures: BankBatchFailure[];
  /** Why the batch ended early by itself (no quota, Codex not logged in, …). */
  stopMessage?: string;
  startedAt?: string;
  finishedAt?: string;
}
