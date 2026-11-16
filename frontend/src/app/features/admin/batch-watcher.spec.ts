import { HiddenBatch } from './batch-watcher';

describe('HiddenBatch', () => {
  const key = 'luna.admin.hiddenBatch.test';

  afterEach(() => localStorage.removeItem(key));

  it('keeps the hidden batch across a reload', () => {
    expect(new HiddenBatch(() => key).get()).toBeNull();
    new HiddenBatch(() => key).set('2026-11-16T02:00:00Z');
    // A new page reads what the last one hid.
    expect(new HiddenBatch(() => key).get()).toBe('2026-11-16T02:00:00Z');
    new HiddenBatch(() => key).set(null);
    expect(localStorage.getItem(key)).toBeNull();
  });

  it('still hides in the page when storage throws', () => {
    const spy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota');
    });
    const h = new HiddenBatch(() => key);
    h.set('2026-11-16T02:00:00Z');
    expect(h.get()).toBe('2026-11-16T02:00:00Z');
    spy.mockRestore();
  });
});
