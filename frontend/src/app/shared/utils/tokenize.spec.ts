import { tokenize } from './tokenize';

const words = (s: string) => tokenize(s).filter((t) => t.isWord).map((t) => t.text);

// The same samples are used by backend/internal/lesson/tokenize_test.go.
describe('tokenize', () => {
  it.each([
    ['We went to the park.', ['We', 'went', 'to', 'the', 'park']],
    ["I don't know.", ['I', "don't", 'know']],
    ['A well-known café.', ['A', 'well-known', 'café']],
    ["It’s rock'n'roll!", ['It’s', "rock'n'roll"]],
    ['U.S. troops at 9.30 a.m.', ['U', 'S', 'troops', 'at', 'a', 'm']],
    ['"Stop!" she said — twice.', ['Stop', 'she', 'said', 'twice']],
    ["'quoted' -dash- end-", ['quoted', 'dash', 'end']],
    ['123 456', []],
  ])('finds the words of %s', (sentence, expected) => {
    expect(words(sentence)).toEqual(expected);
  });

  it('keeps every character and numbers tokens in order', () => {
    const s = '"Stop!" she said.';
    const tokens = tokenize(s);
    expect(tokens.map((t) => t.text).join('')).toBe(s);
    expect(tokens.map((t) => t.index)).toEqual(tokens.map((_, i) => i));
    expect(tokens.filter((t) => !t.isWord).map((t) => t.text)).toEqual(['"', '!" ', ' ', '.']);
  });

  it('returns no tokens for an empty sentence', () => {
    expect(tokenize('')).toEqual([]);
  });
});
