import { of } from 'rxjs';

import { pollWhile } from './poll-while';

describe('pollWhile', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('loads immediately, then every interval while the predicate holds', () => {
    const values = [1, 2, 3, 4];
    let calls = 0;
    const load = () => of(values[calls++]);
    const seen: number[] = [];

    const sub = pollWhile(load, (v) => v < 3, 5000).subscribe((v) => seen.push(v));
    expect(seen).toEqual([1]);

    vi.advanceTimersByTime(4999);
    expect(seen).toEqual([1]);
    vi.advanceTimersByTime(1);
    expect(seen).toEqual([1, 2]);
    vi.advanceTimersByTime(5000);
    expect(seen).toEqual([1, 2, 3]); // 3 fails the predicate: polling stops

    vi.advanceTimersByTime(20000);
    expect(seen).toEqual([1, 2, 3]);
    expect(calls).toBe(3);
    sub.unsubscribe();
  });

  it('stops when unsubscribed', () => {
    let calls = 0;
    const sub = pollWhile(() => of(calls++), () => true, 5000).subscribe();
    sub.unsubscribe();
    vi.advanceTimersByTime(20000);
    expect(calls).toBe(1);
  });
});
