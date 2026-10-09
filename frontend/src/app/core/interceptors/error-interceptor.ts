import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { AuthService } from '../services/auth.service';

/** Normalized error for every failed API call. */
export class ApiError extends Error {
  constructor(
    readonly kind: 'network' | 'http',
    readonly status: number,
    readonly body: unknown,
  ) {
    super(kind === 'network' ? 'Network error' : `HTTP ${status}`);
    this.name = 'ApiError';
  }
}

/** Auth endpoints report 401 as a normal result (wrong password, not logged in yet). */
const AUTH_ENDPOINTS = ['/auth/login', '/auth/register', '/auth/me'];

/** A lesson not open yet (L): the lesson pages say so themselves instead of "no permission". */
function isLessonLocked(body: unknown): boolean {
  return (body as { error?: unknown } | null)?.error === 'lesson_locked';
}

/**
 * Turns HttpErrorResponse into ApiError so features handle failures the same way.
 * A 401 means the session ended: clear the user and go to /login, returning here afterwards.
 * A 403 shows the "no permission" page, except for a locked lesson.
 */
export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);
  const auth = inject(AuthService);

  return next(req).pipe(
    catchError((err: unknown) => {
      if (!(err instanceof HttpErrorResponse)) {
        return throwError(() => err);
      }

      if (err.status === 401 && !AUTH_ENDPOINTS.includes(req.url)) {
        auth.clear();
        const current = router.url;
        if (!current.startsWith('/login')) {
          void router.navigateByUrl(`/login?returnUrl=${encodeURIComponent(current)}`);
        }
      } else if (err.status === 403 && !isLessonLocked(err.error)) {
        void router.navigateByUrl('/forbidden');
      }

      const kind = err.status === 0 ? 'network' : 'http';
      return throwError(() => new ApiError(kind, err.status, err.error));
    }),
  );
};
