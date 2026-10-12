import { DOCUMENT } from '@angular/common';
import { DestroyRef, inject, Injectable } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationStart, Router } from '@angular/router';
import { SwUpdate } from '@angular/service-worker';
import { filter, first, switchMap } from 'rxjs';

/**
 * Keeps an installed PWA on the latest build. Once the service worker has downloaded a new version,
 * the page reloads on the next route change instead of at once, so a lesson in progress is never
 * cut off. A broken cache (unrecoverable state) reloads straight away. Does nothing in `ng serve`.
 */
@Injectable({ providedIn: 'root' })
export class AppUpdateService {
  private readonly updates = inject(SwUpdate);
  private readonly router = inject(Router);
  private readonly document = inject(DOCUMENT);
  private readonly destroyRef = inject(DestroyRef);

  start(): void {
    if (!this.updates.isEnabled) return;

    this.updates.versionUpdates
      .pipe(
        filter((event) => event.type === 'VERSION_READY'),
        first(),
        switchMap(() => this.router.events.pipe(filter((e) => e instanceof NavigationStart))),
        first(),
        takeUntilDestroyed(this.destroyRef),
      )
      // Load the page the learner was heading to, now served by the new version.
      .subscribe((navigation) => this.document.location.assign(navigation.url));

    this.updates.unrecoverable
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.document.location.reload());
  }
}
