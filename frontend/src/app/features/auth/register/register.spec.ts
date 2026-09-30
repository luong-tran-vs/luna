import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { AuthService } from '../../../core/services/auth.service';
import { Register } from './register';

describe('Register', () => {
  let fixture: ComponentFixture<Register>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;

  const input = (name: string) => el.querySelector<HTMLInputElement>(`input[name="${name}"]`)!;
  const type = async (name: string, value: string) => {
    const field = input(name);
    field.value = value;
    field.dispatchEvent(new Event('input'));
    field.dispatchEvent(new Event('blur'));
    await fixture.whenStable();
  };
  const submit = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await fixture.whenStable();
  };
  const errorOf = (name: string) => {
    const id = input(name).getAttribute('aria-describedby');
    return id ? el.querySelector(`#${id}`)?.textContent?.trim() : undefined;
  };
  // submit() is async: let its promise chain finish before checking the view.
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const alertText = () => el.querySelector('[role="alert"]')?.textContent?.trim();

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Register],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    }).compileComponents();

    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture = TestBed.createComponent(Register);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('shows Vietnamese errors for invalid email and short password', async () => {
    await type('email', 'khong-hop-le');
    await type('password', '1234567');

    expect(errorOf('email')).toBe('Email không hợp lệ');
    expect(errorOf('password')).toBe('Mật khẩu cần ít nhất 8 ký tự');
    expect(input('email').getAttribute('aria-invalid')).toBe('true');
  });

  it('shows required errors when submitting an empty form and sends nothing', async () => {
    await submit();
    expect(errorOf('email')).toBe('Vui lòng nhập email');
    expect(errorOf('password')).toBe('Vui lòng nhập mật khẩu');
    http.expectNone('/api/auth/register');
  });

  it('registers, disables the button while pending, and goes home', async () => {
    await type('email', 'an@example.com');
    await type('password', 'matkhau123');
    await submit();

    const button = el.querySelector<HTMLButtonElement>('button[type="submit"]')!;
    expect(button.disabled).toBe(true);

    const req = http.expectOne('/api/auth/register');
    expect(req.request.body.timezone).toBe(Intl.DateTimeFormat().resolvedOptions().timeZone);
    req.flush(
      { user: { id: '1', email: 'an@example.com', role: 'admin', timezone: 'Asia/Ho_Chi_Minh' } },
      { status: 201, statusText: 'Created' },
    );
    await settle();

    expect(TestBed.inject(AuthService).currentUser()?.email).toBe('an@example.com');
    expect(router.navigateByUrl).toHaveBeenCalledWith('/');
  });

  it('shows the server message when the email is taken and keeps the email', async () => {
    await type('email', 'an@example.com');
    await type('password', 'matkhau123');
    await submit();
    http
      .expectOne('/api/auth/register')
      .flush(
        { error: 'email_taken', message: 'Email này đã được dùng' },
        { status: 409, statusText: 'Conflict' },
      );
    await settle();

    expect(alertText()).toBe('Email này đã được dùng');
    expect(input('email').value).toBe('an@example.com');
    expect(el.querySelector<HTMLButtonElement>('button[type="submit"]')!.disabled).toBe(false);
  });

  it('shows a connection error when the server is unreachable', async () => {
    await type('email', 'an@example.com');
    await type('password', 'matkhau123');
    await submit();
    http.expectOne('/api/auth/register').error(new ProgressEvent('error'));
    await settle();

    expect(alertText()).toBe('Không kết nối được máy chủ. Vui lòng thử lại.');
  });

  it('links to the login page', () => {
    expect(el.querySelector('a[href="/login"]')).not.toBeNull();
  });
});
