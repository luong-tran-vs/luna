import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { User } from '../models/user';
import { AuthService } from './auth.service';
import { THEME_STORAGE_KEY, ThemeService } from './theme.service';

type Listener = (e: { matches: boolean }) => void;

/** Controllable stand-in for matchMedia('(prefers-color-scheme: dark)'). */
function fakeMedia(initialDark: boolean) {
  const listeners = new Set<Listener>();
  const media = {
    matches: initialDark,
    addEventListener: (_: string, l: Listener) => listeners.add(l),
    removeEventListener: (_: string, l: Listener) => listeners.delete(l),
  };
  return {
    matchMedia: vi.fn(() => media),
    setDark(dark: boolean) {
      media.matches = dark;
      listeners.forEach((l) => l({ matches: dark }));
    },
  };
}

describe('ThemeService', () => {
  const html = document.documentElement;
  let media: ReturnType<typeof fakeMedia>;
  let http: HttpTestingController;
  const user = signal<User | null>(null);
  const login = () => {
    user.set({ id: 'u1', email: 'a@example.com', role: 'learner', timezone: 'Asia/Ho_Chi_Minh' });
    TestBed.tick();
  };

  const create = () => {
    const service = TestBed.inject(ThemeService);
    TestBed.tick();
    return service;
  };

  beforeEach(() => {
    localStorage.clear();
    html.removeAttribute('data-theme');
    html.style.colorScheme = '';
    media = fakeMedia(false);
    vi.stubGlobal('matchMedia', media.matchMedia);
    user.set(null);
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: { currentUser: user, isLoggedIn: computed(() => user() !== null) } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('defaults to system and follows a light device', () => {
    const service = create();
    expect(service.preference()).toBe('system');
    expect(html.getAttribute('data-theme')).toBe('light');
  });

  it('follows a dark device when set to system', () => {
    media = fakeMedia(true);
    vi.stubGlobal('matchMedia', media.matchMedia);
    create();
    expect(html.getAttribute('data-theme')).toBe('dark');
    expect(html.style.colorScheme).toBe('dark');
  });

  it('applies a choice immediately and stores it', () => {
    const service = create();
    service.set('dark');
    TestBed.tick();
    expect(html.getAttribute('data-theme')).toBe('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('restores a stored choice', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'dark');
    const service = create();
    expect(service.preference()).toBe('dark');
    expect(html.getAttribute('data-theme')).toBe('dark');
  });

  it('falls back to system for an invalid stored value', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'purple');
    expect(create().preference()).toBe('system');
  });

  it('keeps working when storage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });

    const service = create();
    expect(service.preference()).toBe('system');

    service.set('dark');
    expect(() => TestBed.tick()).not.toThrow();
    expect(html.getAttribute('data-theme')).toBe('dark');
  });

  it('updates when the device switches while set to system', () => {
    create();
    media.setDark(true);
    TestBed.tick();
    expect(html.getAttribute('data-theme')).toBe('dark');
  });

  it('ignores device changes when an explicit choice is set', () => {
    const service = create();
    service.set('light');
    media.setDark(true);
    TestBed.tick();
    expect(html.getAttribute('data-theme')).toBe('light');
  });

  // --- F12: the choice follows the account ---

  it('does not call the server when logged out', () => {
    const service = create();
    service.set('dark');
    TestBed.tick();
    http.expectNone('/api/settings');
    expect(service.saveState()).toBe('idle');
  });

  it('applies the account choice after login without saving it back', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light');
    const service = create();
    login();
    const req = http.expectOne('/api/settings');
    expect(req.request.method).toBe('GET');
    req.flush({ theme: 'dark', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' });
    TestBed.tick();
    expect(service.preference()).toBe('dark');
    expect(html.getAttribute('data-theme')).toBe('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
    http.expectNone({ method: 'PUT', url: '/api/settings' });
  });

  it('keeps the local choice when the account cannot be read', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light');
    const service = create();
    login();
    http.expectOne('/api/settings').error(new ProgressEvent('error'));
    TestBed.tick();
    expect(service.preference()).toBe('light');
  });

  it('saves a change to the account and to the browser when logged in', () => {
    const service = create();
    login();
    http.expectOne('/api/settings').flush({ theme: 'system', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' });

    service.set('dark');
    TestBed.tick();
    expect(html.getAttribute('data-theme')).toBe('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
    expect(service.saveState()).toBe('saving');
    const req = http.expectOne({ method: 'PUT', url: '/api/settings' });
    expect(req.request.body).toEqual({ theme: 'dark' });
    req.flush({ theme: 'dark', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' });
    expect(service.saveState()).toBe('saved');
  });

  it('keeps the new choice when saving fails', () => {
    const service = create();
    login();
    http.expectOne('/api/settings').flush({ theme: 'system', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' });
    service.set('dark');
    http.expectOne({ method: 'PUT', url: '/api/settings' }).error(new ProgressEvent('error'));
    TestBed.tick();
    expect(service.saveState()).toBe('error');
    expect(html.getAttribute('data-theme')).toBe('dark');
  });

  it('keeps the browser choice after logout', () => {
    const service = create();
    login();
    http.expectOne('/api/settings').flush({ theme: 'dark', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' });
    TestBed.tick();
    user.set(null);
    TestBed.tick();
    expect(service.preference()).toBe('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });
});
