const AUTH_PAGES = ['/login', '/register'];

/**
 * Returns url when it is a path inside the app, otherwise '/'. Prevents open redirects
 * (https://…, //host, /\host) and loops back to the login or register pages.
 */
export function safeReturnUrl(url: unknown): string {
  if (typeof url !== 'string' || !url.startsWith('/') || url.startsWith('//') || url.startsWith('/\\')) {
    return '/';
  }
  const path = url.split(/[?#]/)[0];
  return AUTH_PAGES.includes(path) ? '/' : url;
}
