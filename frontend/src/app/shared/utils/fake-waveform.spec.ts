import { estimateSeconds, fakeWaveform } from './fake-waveform';

describe('fakeWaveform', () => {
  it('gives the same heights for the same sentence, in [0.05, 1]', () => {
    const a = fakeWaveform('Nice to meet you, Anna.', 48);
    expect(a).toEqual(fakeWaveform('Nice to meet you, Anna.', 48));
    expect(a.length).toBe(48);
    expect(Math.max(...a)).toBe(1);
    expect(a.every((h) => h >= 0.05 && h <= 1)).toBe(true);
    expect(fakeWaveform('We went home.', 48)).not.toEqual(a);
  });

  it('dips at a full stop between two sentences', () => {
    const bars = fakeWaveform('Hello there friend. Welcome to our beautiful home', 80);
    // The pause sits around the middle of the timeline.
    const middle = bars.slice(25, 45);
    expect(Math.min(...middle)).toBeLessThan(0.35);
  });

  it('handles empty text and zero bars', () => {
    expect(fakeWaveform('', 5)).toEqual([0.05, 0.05, 0.05, 0.05, 0.05]);
    expect(fakeWaveform('Hi.', 0)).toEqual([]);
  });

  it('estimates a longer time for a longer sentence', () => {
    const short = estimateSeconds('Hi.');
    const long = estimateSeconds('My grandmother lives in a beautiful village, far from the city.');
    expect(short).toBeGreaterThanOrEqual(0.5);
    expect(long).toBeGreaterThan(short * 3);
  });
});
