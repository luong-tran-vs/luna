import { LEVELS } from './lesson';
import { CRITERIA_LABELS, formatScore, SUGGESTED_WORDS } from './writing';

describe('writing model', () => {
  it('suggests a length for every level', () => {
    for (const level of LEVELS) {
      expect(SUGGESTED_WORDS[level].min).toBeLessThan(SUGGESTED_WORDS[level].max);
    }
    expect(SUGGESTED_WORDS.A1).toEqual({ min: 30, max: 60 });
  });

  it('labels the four criteria in Vietnamese', () => {
    expect(Object.keys(CRITERIA_LABELS)).toEqual(['task', 'grammar', 'vocabulary', 'coherence']);
    expect(CRITERIA_LABELS.coherence).toBe('Mạch lạc');
  });

  it('formats scores with a decimal comma', () => {
    expect(formatScore(3.75)).toBe('3,8');
    expect(formatScore(4)).toBe('4,0');
  });
});
