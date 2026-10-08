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
}

/** Filter of the list: every word, or the words lacking one thing. */
export type BankMissing = '' | 'image' | 'ipa' | 'meaning';

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
