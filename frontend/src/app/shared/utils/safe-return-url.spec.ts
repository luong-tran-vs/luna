import { safeReturnUrl } from './safe-return-url';

describe('safeReturnUrl', () => {
  it.each(['/admin', '/?a=1', '/bai-hoc/3#doan-2'])('keeps the in-app path %s', (url) => {
    expect(safeReturnUrl(url)).toBe(url);
  });

  it.each([
    null,
    undefined,
    '',
    42,
    'admin',
    'https://evil.example',
    '//evil.example',
    '/\\evil.example',
    'javascript:alert(1)',
    '/login',
    '/login?returnUrl=/admin',
    '/register?x=1',
  ])('falls back to / for %s', (url) => {
    expect(safeReturnUrl(url)).toBe('/');
  });
});
