import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../core/services/auth.service';
import { AuthHero } from '../auth-hero';
import { emailError, passwordError, requestErrorMessage } from '../auth-messages';

@Component({
  selector: 'lu-register',
  imports: [AuthHero, ReactiveFormsModule, RouterLink],
  templateUrl: './register.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Register {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected readonly form = inject(NonNullableFormBuilder).group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(8), Validators.maxLength(128)]],
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
    return c.touched || this.submitted() ? passwordError(c) : null;
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
      await this.auth.register(email, password);
      await this.router.navigateByUrl('/');
    } catch (err) {
      this.serverError.set(requestErrorMessage(err));
    } finally {
      this.pending.set(false);
    }
  }
}
