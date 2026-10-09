import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Login } from './login';

describe('Login', () => {
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;
  let whenStable: () => Promise<unknown>;

  const setup = async (returnUrl?: string) => {
    await TestBed.configureTestingModule({
      imports: [Login],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { queryParamMap: convertToParamMap(returnUrl ? { returnUrl } : {}) },
          },
        },
      ],
    }).compileComponents();

    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    const fixture = TestBed.createComponent(Login);
    el = fixture.nativeElement as HTMLElement;
    // submit() is async: let its promise chain finish before checking the view.
    whenStable = async () => {
      await new Promise((resolve) => setTimeout(resolve));
      return fixture.whenStable();
    };
    await whenStable();
  };

  const type = async (name: string, value: string) => {
    const field = el.querySelector<HTMLInputElement>(`input[name="${name}"]`)!;
    field.value = value;
    field.dispatchEvent(new Event('input'));
    field.dispatchEvent(new Event('blur'));
    await whenStable();
  };
  const submitValid = async () => {
    await type('email', 'hoc@example.com');
    await type('password', 'matkhau123');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await whenStable();
  };
  const user = { id: '2', email: 'hoc@example.com', role: 'member', timezone: 'Asia/Ho_Chi_Minh' };
  const alertText = () => el.querySelector('[role="alert"]')?.textContent?.trim();

  afterEach(() => http.verify());

  it('shows required errors and sends nothing for an empty form', async () => {
    await setup();
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await whenStable();

    expect(el.textContent).toContain('Vui lòng nhập email');
    expect(el.textContent).toContain('Vui lòng nhập mật khẩu');
    http.expectNone('/auth/login');
  });

  it('shows the generic message on wrong credentials', async () => {
    await setup();
    await submitValid();
    http
      .expectOne('/auth/login')
      .flush(
        { error: 'invalid_credentials', message: 'Email hoặc mật khẩu không đúng' },
        { status: 401, statusText: 'Unauthorized' },
      );
    await whenStable();

    expect(alertText()).toBe('Email hoặc mật khẩu không đúng');
    expect(router.navigateByUrl).not.toHaveBeenCalled();
  });

  it('shows the lock message with minutes when locked', async () => {
    await setup();
    await submitValid();
    http
      .expectOne('/auth/login')
      .flush(
        { error: 'account_locked', message: 'x', retryAfterSeconds: 900 },
        { status: 429, statusText: 'Too Many Requests' },
      );
    await whenStable();

    expect(alertText()).toBe('Đăng nhập tạm khoá. Thử lại sau 15 phút.');
  });

  it('rounds the remaining lock time up to whole minutes', async () => {
    await setup();
    await submitValid();
    http
      .expectOne('/auth/login')
      .flush(
        { error: 'account_locked', retryAfterSeconds: 61 },
        { status: 429, statusText: 'Too Many Requests' },
      );
    await whenStable();

    expect(alertText()).toBe('Đăng nhập tạm khoá. Thử lại sau 2 phút.');
  });

  it('returns to the requested in-app page after login', async () => {
    await setup('/admin');
    await submitValid();
    http.expectOne('/auth/login').flush({ user });
    await whenStable();

    expect(router.navigateByUrl).toHaveBeenCalledWith('/admin');
  });

  it('ignores an external return URL', async () => {
    await setup('https://evil.example');
    await submitValid();
    http.expectOne('/auth/login').flush({ user });
    await whenStable();

    expect(router.navigateByUrl).toHaveBeenCalledWith('/');
  });

  it('disables the submit button while pending', async () => {
    await setup();
    await submitValid();
    expect(el.querySelector<HTMLButtonElement>('button[type="submit"]')!.disabled).toBe(true);
    http.expectOne('/auth/login').flush({ user });
    await whenStable();
  });

  it('links to the register page', async () => {
    await setup();
    expect(el.querySelector('a[href="/register"]')).not.toBeNull();
  });
});
