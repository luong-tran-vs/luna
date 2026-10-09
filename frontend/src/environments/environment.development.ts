/**
 * Development build (`ng serve`): the API base put before every API path, ending in `/api`. The
 * browser calls the backend directly, so CORS_ORIGINS of the backend must allow this app's origin.
 */
export const environment = {
  apiUrl: 'http://localhost:8080/api',
};
