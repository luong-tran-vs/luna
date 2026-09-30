import { EMPTY, expand, Observable, timer } from 'rxjs';
import { switchMap } from 'rxjs/operators';

/**
 * Emits load() now, then reloads every intervalMs while keepPolling(lastValue) is true.
 * Unsubscribing (for example with takeUntilDestroyed) stops polling.
 */
export function pollWhile<T>(
  load: () => Observable<T>,
  keepPolling: (value: T) => boolean,
  intervalMs = 5000,
): Observable<T> {
  return load().pipe(
    expand((value) => (keepPolling(value) ? timer(intervalMs).pipe(switchMap(() => load())) : EMPTY)),
  );
}
