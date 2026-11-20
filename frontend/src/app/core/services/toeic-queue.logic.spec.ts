import { isEmptyProgress, mergeProgress } from './toeic-queue.logic';

describe('mergeProgress', () => {
  it('keeps the first progress as it is', () => {
    expect(mergeProgress(null, { answers: { a: 1 }, remainingMs: 500 })).toEqual({
      answers: { a: 1 },
      remainingMs: 500,
    });
  });

  it('merges the answers, a later -1 included', () => {
    const out = mergeProgress({ answers: { a: 1, b: 2 } }, { answers: { b: -1, c: 0 } });
    expect(out.answers).toEqual({ a: 1, b: -1, c: 0 });
  });

  it('keeps the smaller time left and the larger elapsed time', () => {
    const out = mergeProgress(
      { remainingMs: 1000, elapsedMs: 9000 },
      { remainingMs: 2000, elapsedMs: 8000 },
    );
    expect(out.remainingMs).toBe(1000);
    expect(out.elapsedMs).toBe(9000);
  });

  it('never goes back to the listening', () => {
    expect(mergeProgress({ phase: 'reading' }, { phase: 'listening' }).phase).toBe('reading');
    expect(mergeProgress({ phase: 'listening' }, { phase: 'reading' }).phase).toBe('reading');
  });

  it('takes the newer position and flags', () => {
    const out = mergeProgress({ position: 'a', flags: ['x'] }, { position: 'b', flags: [] });
    expect(out.position).toBe('b');
    expect(out.flags).toEqual([]);
  });
});

describe('isEmptyProgress', () => {
  it('tells a progress with nothing in it', () => {
    expect(isEmptyProgress({})).toBe(true);
    expect(isEmptyProgress({ answers: {} })).toBe(true);
    expect(isEmptyProgress({ answers: { a: 1 } })).toBe(false);
    expect(isEmptyProgress({ flags: [] })).toBe(false);
    expect(isEmptyProgress({ remainingMs: 0 })).toBe(false);
  });
});
