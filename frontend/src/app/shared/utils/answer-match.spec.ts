import { isCorrectAnswer, normalizeAnswer } from './answer-match';

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
