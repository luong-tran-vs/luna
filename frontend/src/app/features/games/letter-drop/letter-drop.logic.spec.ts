import { GameWord } from '../../../core/models/games';
import {
  currentLevel,
  LANES,
  distractorCount,
  distractorLetters,
  dropFell,
  fallDuration,
  LetterState,
  lettersFinished,
  MAX_LETTERS_SCORE,
  neededLetter,
  nextWord,
  planWord,
  startLetters,
  tapLetter,
  wordPoints,
} from './letter-drop.logic';

const w = (lemma: string): GameWord => ({ lemma, meaningVi: 'nghĩa ' + lemma, imageUrl: '' });

/** A repeatable random sequence. */
function seq(...values: number[]): () => number {
  let i = 0;
  return () => values[i++ % values.length];
}
const rand = seq(0.13, 0.71, 0.42, 0.97, 0.05, 0.58, 0.33, 0.86);

/** Taps the first drop showing letter (right or wrong). */
function tap(s: LetterState, letter: string): LetterState {
  const d = s.drops.find((x) => x.letter === letter);
  if (!d) {
    throw new Error(`no drop "${letter}"`);
  }
  return tapLetter(s, d.id);
}

function spell(s: LetterState): LetterState {
  let t = s;
  while (neededLetter(t) !== null && !t.solved) {
    t = tap(t, neededLetter(t)!);
  }
  return t;
}

describe('letter drop', () => {
  it('fills "house" letter by letter and scores it', () => {
    let s = startLetters([w('house'), w('cat')], rand);
    for (const c of 'hous') {
      s = tap(s, c);
      expect(s.last).toBe('right');
    }
    expect(s.filled).toBe(4);
    s = tap(s, 'e');
    expect(s.solved).toBe(true);
    expect(s.spelt).toBe(1);
    expect(s.score).toBe(100);
    expect(s.drops).toEqual([]);
  });

  it('counts the repeated letters of "apple" in order', () => {
    let s = startLetters([w('apple')], rand);
    expect(s.drops.filter((d) => d.letter === 'p').length).toBe(2);
    s = tap(s, 'a');
    s = tap(s, 'p');
    expect(s.filled).toBe(2);
    expect(neededLetter(s)).toBe('p');
    s = tap(s, 'p');
    expect(s.filled).toBe(3);
    expect(neededLetter(s)).toBe('l');
    expect(s.lives).toBe(3);
  });

  it('loses a life and fills nothing on a wrong letter', () => {
    let s = startLetters([w('house')], rand);
    s = tap(s, 'h');
    const wrong = s.drops.find((d) => d.letter !== 'o')!;
    s = tapLetter(s, wrong.id);
    expect(s.lives).toBe(2);
    expect(s.filled).toBe(1);
    expect(s.wrongId).toBe(wrong.id);
    expect(s.drops).toContain(wrong);
    expect(s.missed.map((m) => m.lemma)).toEqual(['house']);
  });

  it('loses a life when the needed letter falls off, and it falls again', () => {
    let s = startLetters([w('cat')], rand);
    const c = s.drops.find((d) => d.letter === 'c')!;
    s = dropFell(s, c.id, rand);
    expect(s.lives).toBe(2);
    expect(s.last).toBe('fell');
    expect(s.drops.some((d) => d.id === c.id)).toBe(false);
    const again = s.drops.find((d) => d.letter === 'c')!;
    expect(again.id).not.toBe(c.id);
    expect(s.drops.length).toBe(startLetters([w('cat')], rand).drops.length);
    // A letter not needed now falls again for free.
    const t = s.drops.find((d) => d.letter === 't')!;
    s = dropFell(s, t.id, rand);
    expect(s.lives).toBe(2);
    expect(s.drops.some((d) => d.letter === 't')).toBe(true);
    expect(s.missed.length).toBe(1);
  });

  it('ends with no life left', () => {
    let s = startLetters([w('dog')], rand);
    for (let i = 0; i < 3; i++) {
      s = tapLetter(s, s.drops.find((d) => d.letter !== 'd')!.id);
    }
    expect(s.lives).toBe(0);
    expect(lettersFinished(s)).toBe(true);
    expect(tap(s, 'd')).toBe(s);
  });

  it('makes 5 words a level, faster and with more distractors after', () => {
    const words = ['cat', 'dog', 'sun', 'pen', 'box', 'cup'].map(w);
    let s = startLetters(words, rand);
    const first = s.drops;
    expect(first.length).toBe(3 + distractorCount(1));
    let score = 0;
    for (let i = 0; i < 5; i++) {
      expect(currentLevel(s)).toBe(1);
      s = spell(s);
      score += s.gained;
      s = nextWord(s, rand);
    }
    expect(score).toBe(4 * wordPoints(1, false) + wordPoints(1, true));
    expect(currentLevel(s)).toBe(2);
    expect(s.drops.length).toBe(3 + distractorCount(2));
    expect(distractorCount(2)).toBeGreaterThan(distractorCount(1));
    expect(s.drops[0].duration).toBeLessThan(first[0].duration);
    expect(fallDuration(2)).toBeLessThan(fallDuration(1));
    s = nextWord(spell(s), rand);
    expect(lettersFinished(s)).toBe(true);
  });

  it('never takes distractors from the letters still missing', () => {
    expect(distractorLetters('house', 0)).not.toContain('o');
    expect(distractorLetters('house', 2)).toContain('h');
    expect(distractorLetters('house', 2)).not.toContain('u');
    for (let level = 1; level <= 6; level++) {
      const drops = planWord('house', level, Math.random, 1);
      const extra = drops.map((d) => d.letter).filter((c) => !'house'.includes(c));
      expect(extra.length).toBe(distractorCount(level));
      expect(drops.length).toBe(5 + distractorCount(level));
    }
  });

  it('keeps every drop inside the area and staggers them', () => {
    const drops = planWord('banana', 3, Math.random, 10);
    for (const d of drops) {
      expect(d.x).toBeGreaterThanOrEqual(0);
      expect(d.x).toBeLessThanOrEqual(100);
      expect(d.duration).toBe(fallDuration(3));
    }
    expect(new Set(drops.map((d) => d.id)).size).toBe(drops.length);
    const delays = drops.map((d) => d.delay);
    expect(Math.max(...delays)).toBeGreaterThan(Math.min(...delays));
  });

  it('caps the score at the server maximum', () => {
    let s = startLetters([w('cat')], rand);
    s = { ...s, score: MAX_LETTERS_SCORE - 10 };
    s = spell(s);
    expect(s.score).toBe(MAX_LETTERS_SCORE);
  });
});

describe('letter lanes', () => {
  it('never puts two letters in a row in the same or a next lane', () => {
    for (let seed = 1; seed < 30; seed++) {
      let s = seed;
      const rand = () => ((s = (s * 16807) % 2147483647) / 2147483647);
      const drops = planWord('speaker', 3, rand, 0);
      drops.forEach((d, i) => {
        expect(LANES).toContain(d.x);
        if (i > 0) {
          expect(Math.abs(d.x - drops[i - 1].x)).toBeGreaterThan(25);
        }
      });
    }
  });
});
