import { shareText, squares } from './share.logic';

describe('daily share', () => {
  it('writes the day, the right answers, the streak and the squares only', () => {
    const answers = [true, true, true, false, true, true, true, true, true, true];
    expect(shareText('2026-11-13', answers, 13)).toBe('Luna 13/11 · 9/10 🔥13\n🟩🟩🟩🟥🟩🟩🟩🟩🟩🟩');
  });

  it('pads the day and month', () => {
    expect(shareText('2026-03-05', [false], 1)).toBe('Luna 05/03 · 0/1 🔥1\n🟥');
    expect(squares([])).toBe('');
  });
});
