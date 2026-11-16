import { formatScore, isNewBest, MAX_COMBO, nextCombo, pictureScore } from './game-score.logic';

describe('game score', () => {
  it('raises the combo up to x5 and drops it on a mistake', () => {
    expect(nextCombo(1, true)).toBe(2);
    expect(nextCombo(3, true)).toBe(4);
    expect(nextCombo(MAX_COMBO, true)).toBe(MAX_COMBO);
    expect(nextCombo(4, false)).toBe(1);
  });

  it('scores 100 × combo', () => {
    expect(pictureScore(1)).toBe(100);
    expect(pictureScore(4)).toBe(400);
  });

  it('tells a new best and formats scores', () => {
    expect(isNewBest(2100, 1800)).toBe(true);
    expect(isNewBest(1800, 1800)).toBe(false);
    expect(formatScore(2200)).toBe('2.200');
    expect(formatScore(980)).toBe('980');
    expect(formatScore(1234567)).toBe('1.234.567');
  });
});
