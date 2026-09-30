import { compareDictation, ComparedWord, normalizeWords } from './dictation-compare';

const statuses = (words: ComparedWord[]) =>
  words.map((w) => (w.status === 'ok' ? w.word : w.status === 'missing' ? `-${w.expected}` : `!${w.word}>${w.expected ?? ''}`));

describe('normalizeWords', () => {
  it('lowercases and splits on any whitespace', () => {
    expect(normalizeWords('  We  WENT\n to\tthe park ')).toEqual(['we', 'went', 'to', 'the', 'park']);
  });

  it('drops punctuation', () => {
    expect(normalizeWords('Why? "Yes!", she said; (really)... ok.')).toEqual(['why', 'yes', 'she', 'said', 'really', 'ok']);
  });

  it('keeps apostrophes between letters and unifies curly quotes', () => {
    expect(normalizeWords("I don’t know. It's rock'n'roll")).toEqual(['i', "don't", 'know', "it's", "rock'n'roll"]);
    expect(normalizeWords("'quoted' dogs'")).toEqual(['quoted', 'dogs']);
  });

  it('splits hyphens and dashes', () => {
    expect(normalizeWords('a well-known place – really—yes')).toEqual(['a', 'well', 'known', 'place', 'really', 'yes']);
  });

  it('keeps dots and commas between digits', () => {
    expect(normalizeWords('At 9.30, about 1,000 people. Done 5.')).toEqual([
      'at',
      '9.30',
      'about',
      '1,000',
      'people',
      'done',
      '5',
    ]);
  });

  it('returns nothing for empty or punctuation-only text', () => {
    expect(normalizeWords('')).toEqual([]);
    expect(normalizeWords(' ?! ')).toEqual([]);
  });
});

describe('compareDictation', () => {
  const sentence = "I don't like green apples.";

  it('accepts an exact match ignoring case and punctuation', () => {
    const r = compareDictation(sentence, "i DON'T like green apples!!");
    expect(r.words.every((w) => w.status === 'ok')).toBe(true);
    expect(r).toMatchObject({ correctWords: 5, totalWords: 5 });
  });

  it('marks one misspelled word as wrong with the expected word', () => {
    const r = compareDictation(sentence, 'i dont like green apples');
    expect(statuses(r.words)).toEqual(['i', "!dont>don't", 'like', 'green', 'apples']);
    expect(r).toMatchObject({ correctWords: 4, totalWords: 5 });
  });

  it('marks missing words at the start, middle and end', () => {
    expect(statuses(compareDictation(sentence, "don't like green apples").words)[0]).toBe('-i');
    const middle = compareDictation(sentence, "I don't like apples");
    expect(statuses(middle.words)).toEqual(['i', "don't", 'like', '-green', 'apples']);
    expect(middle).toMatchObject({ correctWords: 4, totalWords: 5 });
    expect(statuses(compareDictation(sentence, "I don't like green").words).at(-1)).toBe('-apples');
  });

  it('marks extra words at the start, middle and end as wrong without an expected word', () => {
    expect(statuses(compareDictation(sentence, "so I don't like green apples").words)[0]).toBe('!so>');
    const middle = compareDictation(sentence, "I don't like the green apples");
    expect(statuses(middle.words)).toEqual(['i', "don't", 'like', '!the>', 'green', 'apples']);
    expect(middle).toMatchObject({ correctWords: 5, totalWords: 6 });
    expect(statuses(compareDictation(sentence, "I don't like green apples too").words).at(-1)).toBe('!too>');
  });

  it('treats empty input as all missing', () => {
    const r = compareDictation(sentence, '   ');
    expect(r.words.every((w) => w.status === 'missing')).toBe(true);
    expect(r).toMatchObject({ correctWords: 0, totalWords: 5 });
  });

  it('handles a single-word sentence', () => {
    expect(compareDictation('Why?', 'why')).toMatchObject({ correctWords: 1, totalWords: 1 });
    expect(statuses(compareDictation('Why?', 'what').words)).toEqual(['!what>why']);
  });

  it('handles totally wrong input', () => {
    const r = compareDictation('We went home.', 'cats eat fish');
    expect(r.words.map((w) => w.status)).toEqual(['wrong', 'wrong', 'wrong']);
    expect(r).toMatchObject({ correctWords: 0, totalWords: 3 });
  });

  it('handles a repeated word', () => {
    const r = compareDictation('I saw the cat.', 'I saw the the cat');
    expect(r.words.filter((w) => w.status === 'wrong')).toEqual([{ status: 'wrong', word: 'the' }]);
    expect(r).toMatchObject({ correctWords: 4, totalWords: 5 });
  });

  it('compares numbers, hyphens and contractions', () => {
    expect(compareDictation('We went to the park at 9.30.', 'we went to the park at 9.30').correctWords).toBe(7);
    expect(compareDictation('We went to the park at 9.30.', 'we went to the park at 9 30').correctWords).toBe(6);
    const r = compareDictation("It's a well-known place.", 'its a well known place');
    expect(statuses(r.words)).toEqual(["!its>it's", 'a', 'well', 'known', 'place']);
  });

  it('compares a 40-word sentence quickly', () => {
    const words = Array.from({ length: 40 }, (_, i) => `word${i}`);
    const start = performance.now();
    const r = compareDictation(words.join(' '), words.slice(5).reverse().join(' '));
    expect(performance.now() - start).toBeLessThan(100);
    expect(r.totalWords).toBeGreaterThanOrEqual(40);
  });
});
