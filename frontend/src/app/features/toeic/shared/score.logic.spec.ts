import {
  accuracyOf,
  durationLabel,
  letter,
  outcomeMark,
  percentLabel,
  suggestedMinutes,
} from './score.logic';

describe('score labels', () => {
  it('formats durations', () => {
    expect(durationLabel(0)).toBe('—');
    expect(durationLabel(65_000)).toBe('1:05');
    expect(durationLabel(7_092_000)).toBe('1:58:12');
  });

  it('formats accuracy with a decimal comma', () => {
    expect(accuracyOf(129, 200)).toBe(64.5);
    expect(accuracyOf(1, 0)).toBe(0);
    expect(percentLabel(64.5)).toBe('64,5%');
    expect(percentLabel(100)).toBe('100%');
  });

  it('gives letters and signs', () => {
    expect(letter(0)).toBe('A');
    expect(letter(3)).toBe('D');
    expect(letter(-1)).toBe('');
    expect(outcomeMark('right')).toBe('✓');
    expect(outcomeMark('wrong')).toBe('✗');
    expect(outcomeMark('skipped')).toBe('–');
  });

  it('suggests ~0.6 min a question, by 5 min', () => {
    expect(suggestedMinutes(0)).toBe(0);
    expect(suggestedMinutes(4)).toBe(5);
    expect(suggestedMinutes(46)).toBe(30);
    expect(suggestedMinutes(100)).toBe(60);
  });
});
