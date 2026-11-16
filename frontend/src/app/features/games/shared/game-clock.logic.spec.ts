import { clockLabel, pauseClock, remaining, resumeClock, startClock, timeUp } from './game-clock.logic';

describe('game clock', () => {
  it('counts down from the time it started', () => {
    const c = startClock(60_000, 1000);
    expect(remaining(c, 1000)).toBe(60_000);
    expect(remaining(c, 17_000)).toBe(44_000);
    expect(timeUp(c, 61_000)).toBe(true);
    expect(remaining(c, 99_000)).toBe(0);
  });

  it('does not count while paused', () => {
    let c = startClock(60_000, 0);
    c = pauseClock(c, 10_000);
    expect(remaining(c, 50_000)).toBe(50_000);
    c = resumeClock(c, 50_000);
    expect(remaining(c, 55_000)).toBe(45_000);
    // Pausing or resuming twice changes nothing.
    expect(pauseClock(pauseClock(c, 56_000), 70_000)).toEqual(pauseClock(c, 56_000));
    expect(resumeClock(c, 99_000)).toBe(c);
  });

  it('shows minutes and seconds', () => {
    expect(clockLabel(44_000)).toBe('0:44');
    expect(clockLabel(60_000)).toBe('1:00');
    expect(clockLabel(200)).toBe('0:01');
    expect(clockLabel(0)).toBe('0:00');
  });
});
