import { editDistance, isCorrectAnswer, letterHint, matchAnswer, normalizeAnswer } from './answer-match';

describe('answer match', () => {
  it('ignores case and extra spaces', () => {
    expect(normalizeAnswer('  Give   UP ')).toBe('give up');
    expect(isCorrectAnswer('WENT', 'went')).toBe(true);
    expect(isCorrectAnswer(' went ', 'Went')).toBe(true);
    expect(isCorrectAnswer('give  up', 'give up')).toBe(true);
  });

  it('treats curly and straight apostrophes alike', () => {
    expect(isCorrectAnswer('don’t', "don't")).toBe(true);
  });

  it('rejects a different word or an empty answer', () => {
    expect(isCorrectAnswer('want', 'went')).toBe(false);
    expect(isCorrectAnswer('give', 'give up')).toBe(false);
    expect(isCorrectAnswer('   ', 'went')).toBe(false);
  });
});

describe('matchAnswer', () => {
  it('tells an exact answer, a close one and a wrong one apart', () => {
    expect(matchAnswer(' Loud ', 'loud')).toBe('exact');
    expect(matchAnswer('musik', 'music')).toBe('close'); // a slip in a word of 5 letters
    expect(matchAnswer('beautifull', 'beautiful')).toBe('close');
    expect(matchAnswer('dance too', 'dance to')).toBe('close');
    expect(matchAnswer('alot', 'a lot')).toBe('close'); // a space left out
    expect(matchAnswer('want', 'went')).toBe('wrong'); // a short word one letter off is another word
    expect(matchAnswer('lout', 'loud')).toBe('wrong');
    expect(matchAnswer('', 'loud')).toBe('wrong');
  });

  it('counts the letters to change', () => {
    expect(editDistance('kitten', 'sitting')).toBe(3);
    expect(editDistance('', 'abc')).toBe(3);
    expect(editDistance('same', 'same')).toBe(0);
  });
});

describe('isSpellingSlip', () => {
  it('does not take a missing or added -s or -ed for a slip', () => {
    expect(matchAnswer('like', 'likes')).toBe('wrong');
    expect(matchAnswer('played', 'playe')).toBe('wrong');
    expect(matchAnswer('musik', 'music')).toBe('close');
  });
});

describe('letterHint', () => {
  it('shows the first letter of each word and a blank for each other letter', () => {
    expect(letterHint('loud')).toBe('l _ _ _');
    expect(letterHint('take a break')).toBe('t _ _ _ / a / b _ _ _ _');
    expect(letterHint('  ')).toBe('');
  });
});
