import {
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { Account, Role } from '../../../../core/models/user';
import { Icon } from '../../../../shared/components/icon/icon';
import { AdminApiService } from '../../admin-api.service';
import { ROLE_INFO } from '../roles';

type Field = 'email' | 'password' | 'role';

/**
 * "Thêm tài khoản" / "Sửa tài khoản" dialog. A native <dialog> opened with showModal(), so focus
 * stays inside and Escape closes it. With an account it edits it (an empty password keeps the
 * current one); without one it creates an account.
 */
@Component({
  selector: 'lu-account-dialog',
  imports: [ReactiveFormsModule, Icon],
  templateUrl: './account-dialog.html',
  styleUrl: './account-dialog.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AccountDialog {
  private readonly api = inject(AdminApiService);

  readonly open = input(false);
  /** The account to edit, null to create one. */
  readonly account = input<Account | null>(null);
  /** The account is the admin's own: its role cannot change. */
  readonly self = input(false);

  readonly saved = output<Account>();
  readonly closed = output<void>();

  protected readonly roles = ROLE_INFO;
  protected readonly form = inject(NonNullableFormBuilder).group({
    email: '',
    password: '',
    role: 'guest' as Role,
  });
  protected readonly busy = signal(false);
  protected readonly errors = signal<Partial<Record<Field, string>>>({});
  protected readonly alert = signal<string | null>(null);
  protected readonly showPassword = signal(false);

  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');

  constructor() {
    effect(() => {
      const el = this.dialog().nativeElement;
      if (this.open()) {
        untracked(() => this.reset());
        if (!el.open) {
          el.showModal();
        }
      } else if (el.open) {
        el.close();
      }
    });
  }

  private reset(): void {
    const a = this.account();
    this.form.reset({ email: a?.email ?? '', password: '', role: a?.role ?? 'guest' });
    if (this.self()) {
      this.form.controls.role.disable();
    } else {
      this.form.controls.role.enable();
    }
    this.errors.set({});
    this.alert.set(null);
    this.showPassword.set(false);
  }

  protected describedBy(field: Field, hint?: string): string | null {
    const ids = [hint, this.errors()[field] ? `account-${field}-error` : undefined].filter(Boolean);
    return ids.length ? ids.join(' ') : null;
  }

  protected onCancel(event: Event): void {
    event.preventDefault();
    this.close();
  }

  protected close(): void {
    if (!this.busy()) {
      this.closed.emit();
    }
  }

  protected async submit(): Promise<void> {
    if (this.busy()) {
      return;
    }
    const v = this.form.getRawValue();
    const local: Partial<Record<Field, string>> = {};
    if (!v.email.trim()) {
      local.email = 'Vui lòng nhập email';
    }
    if (!this.account() && !v.password) {
      local.password = 'Vui lòng nhập mật khẩu';
    }
    this.errors.set(local);
    this.alert.set(null);
    if (Object.keys(local).length > 0) {
      return;
    }
    this.busy.set(true);
    try {
      const a = this.account();
      const saved = await firstValueFrom(a ? this.api.updateAccount(a.id, v) : this.api.createAccount(v));
      this.saved.emit(saved);
    } catch (err) {
      const body =
        err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      if (body?.fields) {
        this.errors.set(body.fields);
      } else {
        this.alert.set(body?.message ?? 'Không lưu được tài khoản, vui lòng thử lại.');
      }
    } finally {
      this.busy.set(false);
    }
  }
}
