import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../core/services/auth.service';
import { safeReturnUrl } from '../../../shared/utils/safe-return-url';
import { AuthHero } from '../auth-hero';
import { emailError, requestErrorMessage } from '../auth-messages';

@Component({
  selector: 'lu-login',
  imports: [AuthHero, ReactiveFormsModule, RouterLink],
  templateUrl: './login.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Login {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly returnUrl = inject(ActivatedRoute).snapshot.queryParamMap.get('returnUrl');

  // Only presence is checked here; the server gives one generic answer for wrong credentials.
  protected readonly form = inject(NonNullableFormBuilder).group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', Validators.required],
  });
  protected readonly pending = signal(false);
  protected readonly submitted = signal(false);
  protected readonly serverError = signal<string | null>(null);

  protected emailError(): string | null {
    const c = this.form.controls.email;
    return c.touched || this.submitted() ? emailError(c) : null;
  }

  protected passwordError(): string | null {
    const c = this.form.controls.password;
    return (c.touched || this.submitted()) && c.hasError('required') ? 'Vui lòng nhập mật khẩu' : null;
  }

  protected async submit(): Promise<void> {
    this.submitted.set(true);
    this.serverError.set(null);
    if (this.form.invalid || this.pending()) {
      this.form.markAllAsTouched();
      return;
    }

    this.pending.set(true);
    try {
      const { email, password } = this.form.getRawValue();
      await this.auth.login(email, password);
      await this.router.navigateByUrl(safeReturnUrl(this.returnUrl));
    } catch (err) {
      this.serverError.set(requestErrorMessage(err));
    } finally {
      this.pending.set(false);
    }
  }
}
