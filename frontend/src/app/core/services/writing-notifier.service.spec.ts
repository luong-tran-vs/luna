import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { UnseenCount } from '../models/writing';
import { AuthService } from './auth.service';
import { POLL_INTERVAL, WritingNotifier } from './writing-notifier.service';

describe('WritingNotifier', () => {
  let notifier: WritingNotifier;
  let http: HttpTestingController;
  const loggedIn = signal(true);
  const admin = signal(false);

  const count = (unseen: number, pending: number, latest: UnseenCount['latest'] = null): UnseenCount => ({
    unseen,
    pending,
    latest,
  });
  const flushCount = async (c: UnseenCount) => {
    http.expectOne('/writings/unseen-count').flush(c);
    await vi.advanceTimersByTimeAsync(0);
  };

  beforeEach(async () => {
    vi.useFakeTimers();
    loggedIn.set(true);
    admin.set(false);
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: { isLoggedIn: loggedIn, isAdmin: admin } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    notifier = TestBed.inject(WritingNotifier);
    TestBed.tick();
    await flushCount(count(0, 0));
  });

  afterEach(() => {
    http.verify();
    vi.useRealTimers();
  });

  it('loads the count on login and does not poll without gradings', async () => {
    expect(notifier.unseen()).toBe(0);
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL * 2);
    http.expectNone('/writings/unseen-count');
  });

  it('polls while a writing is graded and shows a toast for the new result', async () => {
    notifier.submitted();
    await flushCount(count(0, 1));
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL);
    await flushCount(count(0, 1));
    expect(notifier.toast()).toBeNull();

    await vi.advanceTimersByTimeAsync(POLL_INTERVAL);
    await flushCount(count(1, 0, { id: 'w1', status: 'done' }));
    expect(notifier.unseen()).toBe(1);
    expect(notifier.toast()).toEqual({ writingId: 'w1', status: 'done', message: 'Bài viết đã có kết quả' });

    // Nothing pending any more: polling stops.
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL * 2);
    http.expectNone('/writings/unseen-count');
  });

  it('says when grading failed', async () => {
    notifier.submitted();
    await flushCount(count(1, 0, { id: 'w2', status: 'failed' }));
    expect(notifier.toast()?.message).toBe('Chấm bài viết bị lỗi');
  });

  it('marks a result seen and refreshes the count', async () => {
    notifier.submitted();
    await flushCount(count(1, 0, { id: 'w1', status: 'done' }));
    const done = notifier.markSeen('w1');
    http.expectOne('/writings/w1/seen').flush(null);
    await vi.advanceTimersByTimeAsync(0);
    await flushCount(count(0, 0));
    await done;
    expect(notifier.unseen()).toBe(0);
    expect(notifier.toast()).toBeNull();
  });

  it('clears everything on logout', async () => {
    notifier.submitted();
    await flushCount(count(1, 1, { id: 'w1', status: 'done' }));
    loggedIn.set(false);
    TestBed.tick();
    expect(notifier.unseen()).toBe(0);
    expect(notifier.toast()).toBeNull();
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL * 2);
    http.expectNone('/writings/unseen-count');
  });

  it('does not ask for writings for an admin', async () => {
    loggedIn.set(false);
    TestBed.tick();
    admin.set(true);
    loggedIn.set(true);
    TestBed.tick();
    await vi.advanceTimersByTimeAsync(0);
    http.expectNone('/writings/unseen-count');
    expect(notifier.unseen()).toBe(0);
  });
});
