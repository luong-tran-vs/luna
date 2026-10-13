import { selectWords } from './selection';

describe('selectWords', () => {
  it('returns null when no word is touched', () => {
    expect(selectWords([])).toBeNull();
  });

  it('selects one word', () => {
    expect(selectWords([{ s: 2, t: 4 }])).toEqual({ sentence: 2, first: 4, last: 4, count: 1, crossed: false });
  });

  it('sorts and spans consecutive words', () => {
    expect(selectWords([{ s: 1, t: 6 }, { s: 1, t: 2 }, { s: 1, t: 4 }])).toEqual({
      sentence: 1, first: 2, last: 6, count: 3, crossed: false,
    });
  });

  it('keeps only the first sentence of a selection across sentences', () => {
    expect(selectWords([{ s: 1, t: 8 }, { s: 2, t: 0 }, { s: 1, t: 10 }, { s: 2, t: 2 }])).toEqual({
      sentence: 1, first: 8, last: 10, count: 2, crossed: true,
    });
  });

  it('takes any number of words of one sentence', () => {
    const touched = Array.from({ length: 15 }, (_, i) => ({ s: 0, t: i * 2 }));
    expect(selectWords(touched)).toEqual({ sentence: 0, first: 0, last: 28, count: 15, crossed: false });
  });
});
