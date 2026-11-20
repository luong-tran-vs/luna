import { splitText } from './doc-text.logic';

describe('splitText', () => {
  it('keeps plain text whole', () => {
    expect(splitText('Hello there.')).toEqual([{ kind: 'text', text: 'Hello there.' }]);
  });

  it('finds bold words and Part 6 blanks', () => {
    expect(splitText('Price **$189** and [[2]] more.')).toEqual([
      { kind: 'text', text: 'Price ' },
      { kind: 'bold', text: '$189' },
      { kind: 'text', text: ' and ' },
      { kind: 'blank', text: '', no: 2 },
      { kind: 'text', text: ' more.' },
    ]);
  });

  it('marks the evidence inside plain and bold text', () => {
    const out = splitText('The garden opens at **9 a.m.** daily.', [
      { text: 'opens at', kind: 'evidence' },
      { text: '9 a.m.', kind: 'user' },
    ]);
    expect(out).toEqual([
      { kind: 'text', text: 'The garden ' },
      { kind: 'text', text: 'opens at', mark: 'evidence' },
      { kind: 'text', text: ' ' },
      { kind: 'bold', text: '9 a.m.', mark: 'user' },
      { kind: 'text', text: ' daily.' },
    ]);
  });

  it('marks every place the words appear and ignores empty marks', () => {
    const out = splitText('ab ab', [
      { text: 'ab', kind: 'user' },
      { text: ' ', kind: 'user' },
    ]);
    // A blank-only mark is ignored.
    expect(out.filter((s) => s.mark).length).toBe(2);
    expect(splitText('x', [{ text: '', kind: 'user' }])).toEqual([{ kind: 'text', text: 'x' }]);
  });
});
