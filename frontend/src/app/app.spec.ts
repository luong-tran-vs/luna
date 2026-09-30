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
import { LessonList } from './features/admin/lesson-list/lesson-list';
import { Login } from './features/auth/login/login';
import { Home } from './features/home/home';

describe('App', () => {
  const user = signal<User | null>(null);
  const loginAs = (role: User['role']) =>
    user.set({ id: '1', email: 'a@example.com', role, timezone: 'Asia/Ho_Chi_Minh' });

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

  it('renders the header and a router outlet inside main', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;

    expect(el.querySelector('lu-app-header')).not.toBeNull();
    expect(el.querySelector('main router-outlet')).not.toBeNull();
    expect(el.querySelector('footer lu-connection-status')).not.toBeNull();
  });

  it('sends a logged-out visitor from home to the login page', async () => {
    const harness = await RouterTestingHarness.create();
    const page = await harness.navigateByUrl('/', Login);
    expect(page).toBeInstanceOf(Login);
    expect(TestBed.inject(Router).url).toBe('/login?returnUrl=%2F');
  });

  it('shows home to a logged-in user', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    expect(await harness.navigateByUrl('/', Home)).toBeInstanceOf(Home);
  });

  it('keeps learners out of /admin and lets admins in', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/admin');
    expect(TestBed.inject(Router).url).toBe('/forbidden');

    loginAs('admin');
    expect(await harness.navigateByUrl('/admin', LessonList)).toBeInstanceOf(LessonList);
  });

  it('redirects unknown routes to the home page', async () => {
    loginAs('learner');
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/khong-ton-tai');
    expect(TestBed.inject(Router).url).toBe('/');
  });
});
