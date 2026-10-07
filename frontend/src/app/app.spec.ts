import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';

import { App } from './app';
import { routes } from './app.routes';
import { User } from './core/models/user';
import { AuthService } from './core/services/auth.service';

describe('App', () => {
  const user = signal<User | null>(null);
  const loginAs = (role: User['role']) =>
    user.set({ id: '1', email: 'a@example.com', role, timezone: 'Asia/Ho_Chi_Minh' });

  const page = (harness: RouterTestingHarness, selector: string) =>
    (harness.fixture.nativeElement as HTMLElement).querySelector(selector);

  beforeEach(async () => {
    user.set(null);
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideRouter(routes),
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: AuthService,
          useValue: {
            currentUser: user,
            isLoggedIn: computed(() => user() !== null),
            isAdmin: computed(() => user()?.role === 'admin'),
          },
        },
      ],
    }).compileComponents();
  });

  it('sends a logged-out visitor from home to the login page', async () => {
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/');
    expect(page(harness, 'lu-public-layout lu-login')).not.toBeNull();
    expect(TestBed.inject(Router).url).toBe('/login?returnUrl=%2F');
  });

  it('shows home to a logged-in user', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/');
    expect(page(harness, 'lu-learner-layout lu-home')).not.toBeNull();
  });

  it('keeps learners out of /admin and lets admins in', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/admin');
    expect(TestBed.inject(Router).url).toBe('/forbidden');

    loginAs('admin');
    await harness.navigateByUrl('/admin');
    expect(page(harness, 'lu-admin-layout lu-admin-dashboard')).not.toBeNull();
  });

  it('redirects unknown routes to the home page', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/khong-ton-tai');
    expect(TestBed.inject(Router).url).toBe('/');
  });

  // --- separate areas ---

  const navLinks = (harness: RouterTestingHarness) =>
    Array.from((harness.fixture.nativeElement as HTMLElement).querySelectorAll('lu-app-header nav a')).map((a) =>
      a.getAttribute('href'),
    );

  it('shows learners only the five learning tabs, inside main with the connection footer', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/');
    const el = harness.fixture.nativeElement as HTMLElement;
    expect(el.querySelector('lu-learner-layout main lu-home')).not.toBeNull();
    expect(el.querySelector('lu-learner-layout footer lu-connection-status')).not.toBeNull();
    expect(el.querySelector('lu-app-header')).toBeNull();
    const tabs = () => Array.from(el.querySelectorAll<HTMLAnchorElement>('lu-learner-layout .tabs a'));
    expect(tabs().map((a) => a.getAttribute('href'))).toEqual(['/', '/goal', '/grammar', '/vocabulary/review', '/account']);
    const current = () => tabs().filter((a) => a.getAttribute('aria-current') === 'page').map((a) => a.textContent?.trim());
    expect(current()).toEqual(['Trang chủ']);

    for (const [url, tab] of [
      ['/goal', 'Khóa học'],
      ['/lessons', 'Khóa học'],
      ['/grammar', 'Ngữ pháp'],
      ['/grammar/a1-to-be', 'Ngữ pháp'],
      ['/vocabulary', 'Ôn tập'],
      ['/settings', 'Tài khoản'],
      ['/writings', 'Tài khoản'],
    ]) {
      await harness.navigateByUrl(url);
      expect(current(), url).toEqual([tab]);
    }
  });

  it('hides the tab bar while studying a lesson', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/lessons/l1/listen');
    const layout = (harness.fixture.nativeElement as HTMLElement).querySelector('lu-learner-layout')!;
    expect(layout.classList).toContain('focus');
    await harness.navigateByUrl('/lessons');
    expect(layout.classList).not.toContain('focus');
  });

  it('shows admins only the admin menu', async () => {
    loginAs('admin');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/admin');
    const el = harness.fixture.nativeElement as HTMLElement;
    expect(el.querySelector('lu-admin-layout main lu-admin-dashboard')).not.toBeNull();
    expect(el.querySelector('lu-learner-layout')).toBeNull();
    expect(el.querySelector('.area')?.textContent?.trim()).toBe('Quản trị');
    // The admin menu is the sidebar, not the header.
    expect(navLinks(harness)).toEqual([]);
    const sidebar = () => Array.from(el.querySelectorAll<HTMLAnchorElement>('lu-admin-layout .sidebar .menu a'));
    expect(sidebar().map((a) => a.getAttribute('href'))).toEqual(['/admin', '/admin/lessons', '/admin/topics', '/admin/grammar', '/admin/roadmap', '/admin/appearance', '/admin/tts-lab', '/admin/stt-lab']);
    const current = () => sidebar().filter((a) => a.getAttribute('aria-current') === 'page').map((a) => a.textContent?.trim());
    expect(current()).toEqual(['Trang chủ']);

    await harness.navigateByUrl('/admin/lessons');
    expect(current()).toEqual(['Bài học']);
    await harness.navigateByUrl('/admin/topics');
    expect(current()).toEqual(['Chủ đề']);
    await harness.navigateByUrl('/admin/lessons/new');
    expect(current()).toEqual(['Bài học']);
  });

  it('sends admins from learning pages to the admin area', async () => {
    loginAs('admin');
    const harness = await RouterTestingHarness.create();
    for (const url of ['/', '/today', '/vocabulary', '/writings', '/settings', '/lessons/l1/read']) {
      await harness.navigateByUrl(url);
      expect(TestBed.inject(Router).url, url).toBe('/admin');
    }
  });

  it('shows login without any menu', async () => {
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/login');
    expect(page(harness, 'lu-public-layout lu-login')).not.toBeNull();
    expect(navLinks(harness)).toEqual([]);
  });
});
