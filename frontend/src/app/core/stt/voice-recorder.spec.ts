import { EndOfSpeech, rms } from './voice-recorder';

describe('rms', () => {
  it('is 0 for silence and the amplitude for a steady signal', () => {
    expect(rms(new Float32Array(4))).toBe(0);
    expect(rms(new Float32Array([0.5, -0.5, 0.5, -0.5]))).toBeCloseTo(0.5);
  });
});

describe('EndOfSpeech', () => {
  it('stops one second after the speaker goes quiet', () => {
    const end = new EndOfSpeech();
    expect(end.update(0, 0.5)).toBe(false);
    expect(end.update(0.1, 1)).toBe(false);
    expect(end.update(0.1, 2)).toBe(false);
    expect(end.update(0, 2.5)).toBe(false);
    expect(end.update(0, 3)).toBe(true);
  });

  it('gives up after six seconds without speech', () => {
    const end = new EndOfSpeech();
    expect(end.update(0.001, 5.9)).toBe(false);
    expect(end.update(0.001, 6)).toBe(true);
  });
});
