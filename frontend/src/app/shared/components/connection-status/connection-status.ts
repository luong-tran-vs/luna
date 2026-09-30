import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { catchError, filter, map, merge, of, switchMap } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { ConnectionStatus, HealthResponse } from '../../../core/models/health';
import { ApiService } from '../../../core/services/api.service';

const ERRORS: Partial<Record<ConnectionStatus, string>> = {
  'database-down': 'Mất kết nối cơ sở dữ liệu',
  'server-unreachable': 'Không kết nối được máy chủ',
};

function isHealthResponse(body: unknown): body is HealthResponse {
  if (typeof body !== 'object' || body === null) {
    return false;
  }
  const { status, database } = body as Record<string, unknown>;
  return (status === 'ok' || status === 'degraded') && (database === 'up' || database === 'down');
}

/** Maps a health response to a status (F0 research R7). */
function fromResponse(body: unknown): ConnectionStatus {
  return isHealthResponse(body) && body.database === 'up' ? 'connected' : 'server-unreachable';
}

/** Maps a failed health call to a status: only the backend's own 503 body means "database down". */
function fromError(err: unknown): ConnectionStatus {
  if (err instanceof ApiError && err.status === 503 && isHealthResponse(err.body)) {
    return err.body.database === 'down' ? 'database-down' : 'server-unreachable';
  }
  return 'server-unreachable';
}

/**
 * The F0 connection check, moved to the footer (F6): it checks after every navigation and shows
 * a message only when the server or the database cannot be reached.
 */
@Component({
  selector: 'lu-connection-status',
  template: `
    <div class="region" role="status" aria-live="polite">
      @if (message(); as text) {
        <p class="alert" [attr.data-status]="status()">
          <span class="dot" aria-hidden="true"></span>
          {{ text }}
        </p>
      }
    </div>
  `,
  styles: `
    .alert {
      --status-color: var(--color-bad);

      display: flex;
      align-items: center;
      gap: var(--space-2);
      margin: 0;
      padding: var(--space-3) var(--space-4);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text);
      background: var(--color-surface);
      border-top: var(--space-1) solid var(--status-color);
    }
    .alert[data-status='database-down'] {
      --status-color: var(--color-warn);
    }
    .dot {
      flex: none;
      width: var(--space-3);
      height: var(--space-3);
      background: var(--status-color);
      border-radius: var(--radius-full);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConnectionStatusBar {
  protected readonly status = signal<ConnectionStatus>('checking');
  protected readonly message = computed(() => ERRORS[this.status()] ?? '');

  constructor() {
    const api = inject(ApiService);
    const router = inject(Router);
    const navigations = router.events.pipe(filter((e) => e instanceof NavigationEnd));
    merge(router.navigated ? of(null) : of(), navigations)
      .pipe(
        switchMap(() =>
          api.getHealth().pipe(
            map(fromResponse),
            catchError((err: unknown) => of(fromError(err))),
          ),
        ),
        takeUntilDestroyed(inject(DestroyRef)),
      )
      .subscribe((s) => this.status.set(s));
  }
}
