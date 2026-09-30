import { Token } from '../../core/models/reading';

/**
 * A word: letters, with apostrophes or hyphens allowed only between letters (don't,
 * well-known). The backend (internal/lesson/tokenize.go) uses the same rule.
 */
const WORD = /\p{L}+(?:['’-]\p{L}+)*/gu;

/** Splits a sentence into word tokens and the text between them, keeping every character. */
export function tokenize(sentence: string): Token[] {
  const tokens: Token[] = [];
  const push = (text: string, isWord: boolean) => {
    if (text) {
      tokens.push({ text, isWord, index: tokens.length });
    }
  };

  let last = 0;
  for (const m of sentence.matchAll(WORD)) {
    push(sentence.slice(last, m.index), false);
    push(m[0], true);
    last = m.index + m[0].length;
  }
  push(sentence.slice(last), false);
  return tokens;
}
