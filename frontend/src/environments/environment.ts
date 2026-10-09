/**
 * Production (`npm run build`): the API base put before every API path (`/auth/login`), ending in
 * `/api`. Use `/api` alone when a reverse proxy serves the app and the backend on one origin.
 */
export const environment = {
  apiUrl: '/api',
};
