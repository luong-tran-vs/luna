import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { AuthService } from '../services/auth.service';
import { ApiError, errorInterceptor } from './error-interceptor';

describe('errorInterceptor', () => {
  let http: HttpClient;
  let controller: HttpTestingController;
  let router: Router;
  let auth: AuthService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpClient);
    controller = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    auth = TestBed.inject(AuthService);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    vi.spyOn(router, 'url', 'get').mockReturnValue('/admin?tab=2');
  });

  afterEach(() => controller.verify());

  const failWith = async (url: string, status: number, body: object | null = null) => {
    const result = firstValueFrom(http.get(url));
    if (status === 0) {
      controller.expectOne(url).error(new ProgressEvent('error'));
    } else {
      controller.expectOne(url).flush(body, { status, statusText: 'Error' });
    }
    return result.catch((e: unknown) => e);
  };

  it('passes successful responses through', async () => {
    const result = firstValueFrom(http.get<{ ok: boolean }>('/api/x'));
    controller.expectOne('/api/x').flush({ ok: true });
    await expect(result).resolves.toEqual({ ok: true });
    expect(router.navigateByUrl).not.toHaveBeenCalled();
  });

  it('maps a network failure to a network ApiError', async () => {
    const err = await failWith('/api/x', 0);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).kind).toBe('network');
    expect((err as ApiError).status).toBe(0);
  });

  it('maps an HTTP error to an http ApiError and keeps the body', async () => {
    const body = { status: 'degraded', database: 'down' };
    const err = await failWith('/api/health', 503, body);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).kind).toBe('http');
    expect((err as ApiError).status).toBe(503);
    expect((err as ApiError).body).toEqual(body);
    expect(router.navigateByUrl).not.toHaveBeenCalled();
  });

  it('on 401 clears the user and redirects to /login with the current URL', async () => {
    const clear = vi.spyOn(auth, 'clear');
    const err = await failWith('/api/admin/ping', 401);

    expect(err).toBeInstanceOf(ApiError);
    expect(clear).toHaveBeenCalled();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/login?returnUrl=%2Fadmin%3Ftab%3D2');
  });

  it.each(['/api/auth/login', '/api/auth/register', '/api/auth/me'])(
    'does not redirect on 401 from %s',
    async (url) => {
      await failWith(url, 401);
      expect(router.navigateByUrl).not.toHaveBeenCalled();
    },
  );

  it('does not redirect on 401 while already on the login page', async () => {
    vi.spyOn(router, 'url', 'get').mockReturnValue('/login?returnUrl=%2Fadmin');
    await failWith('/api/admin/ping', 401);
    expect(router.navigateByUrl).not.toHaveBeenCalled();
  });

  it('on 403 goes to /forbidden', async () => {
    const err = await failWith('/api/admin/ping', 403);
    expect(err).toBeInstanceOf(ApiError);
    expect(router.navigateByUrl).toHaveBeenCalledWith('/forbidden');
  });
});
