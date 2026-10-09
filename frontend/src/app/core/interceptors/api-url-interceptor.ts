import { HttpInterceptorFn } from '@angular/common/http';

import { apiUrl, isApiPath } from '../api-url';

/**
 * Sends API paths (`/auth/login`) to the API base of the environment (`apiUrl`, which ends in
 * `/api`), with the session cookie (needed when the backend is on another origin). Listed after
 * errorInterceptor, which matches the original paths.
 */
export const apiUrlInterceptor: HttpInterceptorFn = (req, next) =>
  next(isApiPath(req.url) ? req.clone({ url: apiUrl(req.url), withCredentials: true }) : req);
