import { DOCUMENT } from '@angular/common';
import { TestBed } from '@angular/core/testing';
import { NavigationStart, Router } from '@angular/router';
import { SwUpdate, VersionEvent } from '@angular/service-worker';
import { Subject } from 'rxjs';

import { AppUpdateService } from './app-update.service';

describe('AppUpdateService', () => {
  let versionUpdates: Subject<VersionEvent>;
  let unrecoverable: Subject<unknown>;
  let routerEvents: Subject<unknown>;
  let location: { assign: ReturnType<typeof vi.fn>; reload: ReturnType<typeof vi.fn> };

  function setup(isEnabled: boolean): AppUpdateService {
    versionUpdates = new Subject();
    unrecoverable = new Subject();
    routerEvents = new Subject();
    location = { assign: vi.fn(), reload: vi.fn() };
    TestBed.configureTestingModule({
      providers: [
        { provide: SwUpdate, useValue: { isEnabled, versionUpdates, unrecoverable } },
        { provide: Router, useValue: { events: routerEvents } },
        { provide: DOCUMENT, useValue: { location } },
      ],
    });
    return TestBed.inject(AppUpdateService);
  }

  const versionReady = { type: 'VERSION_READY' } as VersionEvent;

  it('reloads into the next page once a new version is ready', () => {
    setup(true).start();

    routerEvents.next(new NavigationStart(1, '/before'));
    expect(location.assign).not.toHaveBeenCalled();

    versionUpdates.next(versionReady);
    expect(location.assign).not.toHaveBeenCalled();

    routerEvents.next(new NavigationStart(2, '/review'));
    expect(location.assign).toHaveBeenCalledExactlyOnceWith('/review');
  });

  it('reloads at once when the cache is unrecoverable', () => {
    setup(true).start();
    unrecoverable.next({ type: 'UNRECOVERABLE_STATE', reason: 'gone' });
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('does nothing when the service worker is off', () => {
    setup(false).start();
    versionUpdates.next(versionReady);
    routerEvents.next(new NavigationStart(1, '/review'));
    unrecoverable.next({});
    expect(location.assign).not.toHaveBeenCalled();
    expect(location.reload).not.toHaveBeenCalled();
  });
});
