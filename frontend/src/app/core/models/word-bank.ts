/** F24: one word of the shared word bank, as the admin API returns it. */
export interface BankWord {
  lemma: string;
  meaningVi: string;
  ipa: string;
  /** "" when the word has no picture; changes when the picture does. */
  imageUrl: string;
  updatedAt: string;
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
}

export interface BankWordDetails {
  meaningVi?: string;
  ipa?: string;
}

/** Limits of the server (internal/wordbank). */
export const MAX_BANK_LEMMA = 100;
export const MAX_BANK_MEANING = 300;
export const MAX_BANK_IPA = 100;
