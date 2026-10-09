import { environment } from '../../environments/environment';

/** The API base (e.g. `http://host:8080/api`) without a trailing slash. */
const apiBase = environment.apiUrl.replace(/\/+$/, '');

/** Where the backend lives: the API base without its `/api` part. */
const serverBase = apiBase.replace(/\/api$/, '');

/** True for a path the app sends to the API (`/auth/login`), not an absolute URL. */
export function isApiPath(url: string): boolean {
  return url.startsWith('/') && !url.startsWith('//');
}

/** Turns an API path (`/auth/login`) into the URL the browser calls; other URLs are left alone. */
export function apiUrl(path: string): string {
  return isApiPath(path) ? apiBase + path : path;
}

/**
 * Turns a URL the backend returns (`/api/lessons/{id}/images/{lemma}`, already with `/api`)
 * into one the browser can load from the backend; other URLs are left alone.
 */
export function serverUrl(path: string): string {
  return isApiPath(path) ? serverBase + path : path;
}
