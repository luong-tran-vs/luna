import { DiffPart, wordDiff } from './word-diff';

const join = (parts: DiffPart[], kinds: DiffPart['kind'][]) =>
  parts
    .filter((p) => kinds.includes(p.kind))
    .map((p) => p.text)
    .join('');
const words = (s: string) => s.split(/\s+/).filter(Boolean).join(' ');

describe('wordDiff', () => {
  it('returns one unchanged part for identical text', () => {
    expect(wordDiff('I go home.', 'I go home.')).toEqual([{ kind: 'same', text: 'I go home.' }]);
  });

  it('marks an added word', () => {
    expect(wordDiff('I go home.', 'I go to home.')).toEqual([
      { kind: 'same', text: 'I go ' },
      { kind: 'add', text: 'to ' },
      { kind: 'same', text: 'home.' },
    ]);
  });

  it('marks a removed word', () => {
    expect(wordDiff('I very like it.', 'I like it.')).toEqual([
      { kind: 'same', text: 'I ' },
      { kind: 'del', text: 'very ' },
      { kind: 'same', text: 'like it.' },
    ]);
  });

  it('shows a replaced word as removed then added', () => {
    expect(wordDiff('He go to school.', 'He goes to school.')).toEqual([
      { kind: 'same', text: 'He ' },
      { kind: 'del', text: 'go ' },
      { kind: 'add', text: 'goes ' },
      { kind: 'same', text: 'to school.' },
    ]);
  });

  it('treats punctuation as part of the word', () => {
    const parts = wordDiff('I am happy', 'I am happy.');
    expect(parts.map((p) => p.kind)).toEqual(['same', 'del', 'add']);
  });

  it('handles empty texts', () => {
    expect(wordDiff('', 'New text.')).toEqual([{ kind: 'add', text: 'New text.' }]);
    expect(wordDiff('Old text.', '')).toEqual([{ kind: 'del', text: 'Old text.' }]);
    expect(wordDiff('', '')).toEqual([]);
  });

  it('rebuilds both texts from the parts', () => {
    const original = 'Yesterday I go to the park with my friend and we play football very happy.';
    const corrected = 'Yesterday I went to the park with my friends, and we played football happily.';
    const parts = wordDiff(original, corrected);
    expect(words(join(parts, ['same', 'del']))).toBe(original);
    expect(join(parts, ['same', 'add'])).toBe(corrected);
  });
});
