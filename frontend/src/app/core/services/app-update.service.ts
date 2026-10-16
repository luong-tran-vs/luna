import { DOCUMENT } from '@angular/common';
import { DestroyRef, inject, Injectable, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationStart, Router } from '@angular/router';
import { SwUpdate } from '@angular/service-worker';
import { filter, first, switchMap } from 'rxjs';

/** How often an open app asks the server for a new version, in milliseconds. */
export const UPDATE_CHECK_MS = 15 * 60 * 1000;

/**
 * Keeps an installed PWA on the latest build.
 *
 * On its own the service worker only looks for a new version once the app has been idle for a few
 * seconds after a cold start (each API call pushes that back, up to half a minute), and an installed
 * app on a phone is mostly brought back from the background instead: so it is asked straight away
 * at start, when the app comes back to the screen, when the network comes back, and every
 * UPDATE_CHECK_MS while it is open.
 *
 * While a new version is being downloaded `downloading` is on (the root shows "Đang tải bản mới…",
 * design/thong-bao-cap-nhat.png); a failed download turns it off again, the next check retries.
 * Once a new version is downloaded, `ready` turns on (the root shows "Cập nhật"), and the page also
 * reloads by itself on the next route change, so a lesson in progress is never cut off. A broken
 * cache (unrecoverable state) reloads straight away. Does nothing in `ng serve`.
 */
@Injectable({ providedIn: 'root' })
export class AppUpdateService {
  // Optional: absent where the service worker is not provided (tests), the same as off.
  private readonly updates = inject(SwUpdate, { optional: true });
  private readonly router = inject(Router);
  private readonly document = inject(DOCUMENT);
  private readonly destroyRef = inject(DestroyRef);

  private readonly _ready = signal(false);
  /** A new version is downloaded and waits for a reload. */
  readonly ready = this._ready.asReadonly();

  private readonly _downloading = signal(false);
  /** A new version was found and is being downloaded. */
  readonly downloading = this._downloading.asReadonly();

  start(): void {
    const updates = this.updates;
    if (!updates?.isEnabled) return;

    updates.versionUpdates.pipe(takeUntilDestroyed(this.destroyRef)).subscribe((event) => {
      this._downloading.set(event.type === 'VERSION_DETECTED');
    });

    const ready$ = updates.versionUpdates.pipe(
      filter((event) => event.type === 'VERSION_READY'),
      first(),
    );
    ready$.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => this._ready.set(true));
    ready$
      .pipe(
        switchMap(() => this.router.events.pipe(filter((e) => e instanceof NavigationStart))),
        first(),
        takeUntilDestroyed(this.destroyRef),
      )
      // Load the page the learner was heading to, now served by the new version.
      .subscribe((navigation) => this.document.location.assign(navigation.url));

    updates.unrecoverable
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.document.location.reload());

    this.watchForUpdates(updates);
  }

  /** Cập nhật: reloads into the new version. */
  update(): void {
    this.document.location.reload();
  }

  /** Asks for a new version now, when the app comes back to the screen or online, and now and then. */
  private watchForUpdates(updates: SwUpdate): void {
    const check = () => {
      if (!this._ready() && this.document.visibilityState !== 'hidden') {
        // Offline or the server is down: the next check tries again.
        updates.checkForUpdate().catch(() => undefined);
      }
    };
    const view = this.document.defaultView;
    this.document.addEventListener('visibilitychange', check);
    view?.addEventListener('online', check);
    const timer = setInterval(check, UPDATE_CHECK_MS);
    check();
    this.destroyRef.onDestroy(() => {
      this.document.removeEventListener('visibilitychange', check);
      view?.removeEventListener('online', check);
      clearInterval(timer);
    });
  }
}
