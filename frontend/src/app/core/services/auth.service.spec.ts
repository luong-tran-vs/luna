import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../interceptors/error-interceptor';
import { User } from '../models/user';
import { AuthService } from './auth.service';

const admin: User = { id: '1', email: 'admin@example.com', role: 'admin', timezone: 'Asia/Ho_Chi_Minh' };
const learner: User = { ...admin, id: '2', email: 'hoc@example.com', role: 'learner' };

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  describe('load', () => {
    it('sets the current user from /api/auth/me', async () => {
      const done = service.load();
      http.expectOne('/api/auth/me').flush({ user: admin });
      await done;
      expect(service.currentUser()).toEqual(admin);
      expect(service.isLoggedIn()).toBe(true);
      expect(service.isAdmin()).toBe(true);
    });

    it('sets null when not logged in', async () => {
      const done = service.load();
      http
        .expectOne('/api/auth/me')
        .flush({ error: 'unauthenticated' }, { status: 401, statusText: 'Unauthorized' });
      await done;
      expect(service.currentUser()).toBeNull();
      expect(service.isLoggedIn()).toBe(false);
    });

    it('sets null and does not reject on a network error', async () => {
      const done = service.load();
      http.expectOne('/api/auth/me').error(new ProgressEvent('error'));
      await expect(done).resolves.toBeUndefined();
      expect(service.currentUser()).toBeNull();
    });
  });

  it('reports a learner as not admin', async () => {
    const done = service.load();
    http.expectOne('/api/auth/me').flush({ user: learner });
    await done;
    expect(service.isAdmin()).toBe(false);
  });

  it('register sends the browser timezone and sets the user', async () => {
    const done = service.register('an@example.com', 'matkhau123');
    const req = http.expectOne('/api/auth/register');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      email: 'an@example.com',
      password: 'matkhau123',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    });
    req.flush({ user: admin }, { status: 201, statusText: 'Created' });
    await done;
    expect(service.currentUser()).toEqual(admin);
  });

  it('login sets the user', async () => {
    const done = service.login('hoc@example.com', 'matkhau123');
    const req = http.expectOne('/api/auth/login');
    expect(req.request.body).toEqual({ email: 'hoc@example.com', password: 'matkhau123' });
    req.flush({ user: learner });
    await done;
    expect(service.currentUser()).toEqual(learner);
  });

  it('logout clears the user even when the request fails', async () => {
    const loading = service.load();
    http.expectOne('/api/auth/me').flush({ user: admin });
    await loading;

    const done = service.logout();
    http.expectOne('/api/auth/logout').error(new ProgressEvent('error'));
    await done;
    expect(service.currentUser()).toBeNull();
  });

  it('clear resets the user', async () => {
    const loading = service.load();
    http.expectOne('/api/auth/me').flush({ user: admin });
    await loading;
    service.clear();
    expect(service.currentUser()).toBeNull();
  });
});
