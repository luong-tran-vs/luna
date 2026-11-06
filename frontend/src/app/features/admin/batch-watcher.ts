import { computed, inject, InjectionToken, signal } from '@angular/core';
import { Observable } from 'rxjs';

/** How often, in ms, the admin pages ask how a running background batch is going (tests shorten it). */
export const BATCH_POLL_MS = new InjectionToken<number>('BATCH_POLL_MS', { factory: () => 3000 });

/** How often, in ms, they look for a batch started elsewhere (another tab) while none runs. */
export const BATCH_IDLE_POLL_MS = new InjectionToken<number>('BATCH_IDLE_POLL_MS', {
  factory: () => 30000,
});

/** What every background picture batch tells about itself. */
export interface BatchStatus {
  running: boolean;
  total: number;
  startedAt?: string;
}

/**
 * Follows a picture batch the server draws in the background (added 2026-11-06), shared by the
 * admin layout, which labels it on every page, and the pages that run it. While a page watches, the
 * status is asked for every few seconds during a batch and now and then otherwise.
 */
export abstract class BatchWatcher<T extends BatchStatus> {
  private readonly pollMs = inject(BATCH_POLL_MS);
  private readonly idleMs = inject(BATCH_IDLE_POLL_MS);

  /** The running, or last, batch; null until known. */
  readonly status = signal<T | null>(null);
  readonly running = computed(() => this.status()?.running ?? false);
  /** The admin hid the result of the batch started then. */
  private readonly hiddenAt = signal<string | null>(null);
  /** A finished batch whose result is still to be shown. */
  readonly result = computed(() => {
    const b = this.status();
    return b && !b.running && b.total > 0 && b.startedAt !== this.hiddenAt() ? b : null;
  });

  private watchers = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  /** Asks the server how the batch is going. */
  protected abstract fetch(): Observable<T>;

  /** Starts following the batch; call the returned function to stop. */
  watch(): () => void {
    this.watchers++;
    if (this.watchers === 1) {
      this.refresh();
    }
    let stopped = false;
    return () => {
      if (stopped) {
        return;
      }
      stopped = true;
      this.watchers--;
      if (this.watchers === 0) {
        clearTimeout(this.timer);
      }
    };
  }

  /** Takes a status the server just answered (after starting or stopping a batch). */
  set(b: T): void {
    this.status.set(b);
    this.schedule();
  }

  hideResult(): void {
    this.hiddenAt.set(this.status()?.startedAt ?? null);
  }

  private refresh(): void {
    this.fetch().subscribe({
      next: (b) => this.set(b),
      // Not knowing the batch only hides its label; ask again later.
      error: () => this.schedule(),
    });
  }

  private schedule(): void {
    clearTimeout(this.timer);
    if (this.watchers > 0) {
      this.timer = setTimeout(() => this.refresh(), this.running() ? this.pollMs : this.idleMs);
    }
  }
}
