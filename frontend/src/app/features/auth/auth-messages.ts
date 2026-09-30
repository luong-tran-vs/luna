import { AbstractControl } from '@angular/forms';

import { ApiError } from '../../core/interceptors/error-interceptor';

const NETWORK_ERROR = 'Không kết nối được máy chủ. Vui lòng thử lại.';
const GENERIC_ERROR = 'Có lỗi xảy ra, vui lòng thử lại.';

export function emailError(control: AbstractControl): string | null {
  if (control.hasError('required')) {
    return 'Vui lòng nhập email';
  }
  if (control.hasError('email')) {
    return 'Email không hợp lệ';
  }
  return null;
}

export function passwordError(control: AbstractControl): string | null {
  if (control.hasError('required')) {
    return 'Vui lòng nhập mật khẩu';
  }
  if (control.hasError('minlength')) {
    return 'Mật khẩu cần ít nhất 8 ký tự';
  }
  if (control.hasError('maxlength')) {
    return 'Mật khẩu tối đa 128 ký tự';
  }
  return null;
}

interface ErrorBody {
  message?: unknown;
  fields?: Record<string, string>;
  retryAfterSeconds?: unknown;
}

/** Turns a failed login/register request into one Vietnamese sentence for the form. */
export function requestErrorMessage(err: unknown): string {
  if (!(err instanceof ApiError)) {
    return GENERIC_ERROR;
  }
  if (err.kind === 'network') {
    return NETWORK_ERROR;
  }

  const body = (typeof err.body === 'object' && err.body !== null ? err.body : {}) as ErrorBody;
  if (err.status === 429) {
    const seconds = typeof body.retryAfterSeconds === 'number' ? body.retryAfterSeconds : 900;
    return `Đăng nhập tạm khoá. Thử lại sau ${Math.max(1, Math.ceil(seconds / 60))} phút.`;
  }
  if (err.status === 400 && body.fields) {
    return Object.values(body.fields).join('. ');
  }
  if (err.status < 500 && typeof body.message === 'string') {
    return body.message;
  }
  return err.status >= 502 ? NETWORK_ERROR : GENERIC_ERROR;
}
