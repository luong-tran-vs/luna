import { HttpClient } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { User } from '../models/user';

interface UserEnvelope {
  user: User;
}

/**
 * Holds the logged-in user. The session itself lives in an HttpOnly cookie that the
 * browser sends with every API request (see apiUrlInterceptor).
 */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly user = signal<User | null>(null);

  readonly currentUser = this.user.asReadonly();
  readonly isLoggedIn = computed(() => this.user() !== null);
  readonly isAdmin = computed(() => this.user()?.role === 'admin');
  readonly isGuest = computed(() => this.user()?.role === 'guest');

  /** Restores the session on startup. Never rejects so the app always boots. */
  async load(): Promise<void> {
    try {
      const { user } = await firstValueFrom(this.http.get<UserEnvelope>('/auth/me'));
      this.user.set(user);
    } catch {
      this.user.set(null);
    }
  }

  async register(email: string, password: string): Promise<void> {
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const { user } = await firstValueFrom(
      this.http.post<UserEnvelope>('/auth/register', { email, password, timezone }),
    );
    this.user.set(user);
  }

  async login(email: string, password: string): Promise<void> {
    const { user } = await firstValueFrom(
      this.http.post<UserEnvelope>('/auth/login', { email, password }),
    );
    this.user.set(user);
  }

  /** Ends the session on the server; the local state is cleared even if that fails. */
  async logout(): Promise<void> {
    try {
      await firstValueFrom(this.http.post<void>('/auth/logout', null));
    } catch {
      // The cookie may already be invalid; logging out locally is what matters.
    } finally {
      this.user.set(null);
    }
  }

  clear(): void {
    this.user.set(null);
  }
}
