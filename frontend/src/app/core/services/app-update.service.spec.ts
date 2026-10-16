import { DOCUMENT } from '@angular/common';
import { TestBed } from '@angular/core/testing';
import { NavigationStart, Router } from '@angular/router';
import { SwUpdate, VersionEvent } from '@angular/service-worker';
import { Subject } from 'rxjs';

import { AppUpdateService, UPDATE_CHECK_MS } from './app-update.service';

describe('AppUpdateService', () => {
  let versionUpdates: Subject<VersionEvent>;
  let unrecoverable: Subject<unknown>;
  let routerEvents: Subject<unknown>;
  let location: { assign: ReturnType<typeof vi.fn>; reload: ReturnType<typeof vi.fn> };
  let checkForUpdate: ReturnType<typeof vi.fn>;
  let doc: EventTarget & {
    location: typeof location;
    visibilityState: string;
    defaultView: EventTarget;
  };

  function setup(isEnabled: boolean): AppUpdateService {
    versionUpdates = new Subject();
    unrecoverable = new Subject();
    routerEvents = new Subject();
    location = { assign: vi.fn(), reload: vi.fn() };
    checkForUpdate = vi.fn(() => Promise.resolve(false));
    doc = Object.assign(new EventTarget(), {
      location,
      visibilityState: 'visible',
      defaultView: new EventTarget(),
    });
    TestBed.configureTestingModule({
      providers: [
        {
          provide: SwUpdate,
          useValue: { isEnabled, versionUpdates, unrecoverable, checkForUpdate },
        },
        { provide: Router, useValue: { events: routerEvents } },
        { provide: DOCUMENT, useValue: doc },
      ],
    });
    return TestBed.inject(AppUpdateService);
  }

  const versionReady = { type: 'VERSION_READY' } as VersionEvent;

  afterEach(() => vi.useRealTimers());

  it('says a new version is ready, and reloads into the next page', () => {
    const service = setup(true);
    service.start();

    routerEvents.next(new NavigationStart(1, '/before'));
    expect(location.assign).not.toHaveBeenCalled();
    expect(service.ready()).toBe(false);

    versionUpdates.next(versionReady);
    expect(service.ready()).toBe(true);
    expect(location.assign).not.toHaveBeenCalled();

    routerEvents.next(new NavigationStart(2, '/review'));
    expect(location.assign).toHaveBeenCalledExactlyOnceWith('/review');
  });

  it('says a new version is downloading until it is ready or the download fails', () => {
    const service = setup(true);
    service.start();
    expect(service.downloading()).toBe(false);

    versionUpdates.next({ type: 'VERSION_DETECTED' } as VersionEvent);
    expect(service.downloading()).toBe(true);
    versionUpdates.next({ type: 'VERSION_INSTALLATION_FAILED' } as VersionEvent);
    expect(service.downloading()).toBe(false);
    expect(service.ready()).toBe(false);

    versionUpdates.next({ type: 'VERSION_DETECTED' } as VersionEvent);
    versionUpdates.next(versionReady);
    expect(service.downloading()).toBe(false);
    expect(service.ready()).toBe(true);
  });

  it('reloads into the new version on Cập nhật', () => {
    const service = setup(true);
    service.start();
    versionUpdates.next(versionReady);
    service.update();
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('looks for a new version at start, when the app comes back to the screen or online', () => {
    setup(true).start();
    expect(checkForUpdate).toHaveBeenCalledTimes(1);

    doc.visibilityState = 'hidden';
    doc.dispatchEvent(new Event('visibilitychange'));
    expect(checkForUpdate).toHaveBeenCalledTimes(1); // going to the background

    doc.visibilityState = 'visible';
    doc.dispatchEvent(new Event('visibilitychange'));
    expect(checkForUpdate).toHaveBeenCalledTimes(2);

    doc.defaultView.dispatchEvent(new Event('online'));
    expect(checkForUpdate).toHaveBeenCalledTimes(3);

    // Not again once a version is waiting.
    versionUpdates.next(versionReady);
    doc.dispatchEvent(new Event('visibilitychange'));
    expect(checkForUpdate).toHaveBeenCalledTimes(3);
  });

  it('looks for a new version now and then while the app stays open', () => {
    vi.useFakeTimers();
    setup(true).start();
    expect(checkForUpdate).toHaveBeenCalledOnce();
    vi.advanceTimersByTime(UPDATE_CHECK_MS - 1);
    expect(checkForUpdate).toHaveBeenCalledOnce();
    vi.advanceTimersByTime(1);
    expect(checkForUpdate).toHaveBeenCalledTimes(2);
  });

  it('keeps going when a check fails (offline)', async () => {
    const service = setup(true);
    checkForUpdate.mockReturnValue(Promise.reject(new Error('offline')));
    service.start();
    doc.dispatchEvent(new Event('visibilitychange'));
    await Promise.resolve();
    expect(service.ready()).toBe(false);
  });

  it('reloads at once when the cache is unrecoverable', () => {
    setup(true).start();
    unrecoverable.next({ type: 'UNRECOVERABLE_STATE', reason: 'gone' });
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('does nothing when the service worker is off', () => {
    vi.useFakeTimers();
    const service = setup(false);
    service.start();
    versionUpdates.next(versionReady);
    routerEvents.next(new NavigationStart(1, '/review'));
    unrecoverable.next({});
    doc.dispatchEvent(new Event('visibilitychange'));
    vi.advanceTimersByTime(UPDATE_CHECK_MS);
    expect(service.ready()).toBe(false);
    expect(location.assign).not.toHaveBeenCalled();
    expect(location.reload).not.toHaveBeenCalled();
    expect(checkForUpdate).not.toHaveBeenCalled();
  });
});
