import { countWords, DEFAULT_WORDS, wordRange } from './generate';
import { LEVELS } from './lesson';

describe('generate model', () => {
  it('counts whitespace-separated words', () => {
    expect(countWords('')).toBe(0);
    expect(countWords('   \n ')).toBe(0);
    expect(countWords('We  went\thome.')).toBe(3);
    expect(countWords('Anna: Hi!\nBen: Hello, Anna.')).toBe(5);
  });

  it('has a default length for every level', () => {
    for (const level of LEVELS) {
      expect(DEFAULT_WORDS[level]).toBeGreaterThan(0);
    }
    expect(DEFAULT_WORDS.A1).toBe(120);
    expect(DEFAULT_WORDS.C2).toBe(400);
  });

  it('accepts ±20% of the target', () => {
    expect(wordRange(120)).toEqual({ min: 96, max: 144 });
    expect(wordRange(50)).toEqual({ min: 40, max: 60 });
  });
});
