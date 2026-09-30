import { DOCUMENT } from '@angular/common';
import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core';

import { AuthService } from './auth.service';
import { SettingsApiService } from './settings-api.service';

export type ThemePreference = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';
/** Saving the choice to the account: idle until a logged-in learner changes it. */
export type ThemeSaveState = 'idle' | 'saving' | 'saved' | 'error';

/** Also read by the inline script in index.html; keep both in sync. */
export const THEME_STORAGE_KEY = 'luna.theme';

const PREFERENCES: readonly ThemePreference[] = ['light', 'dark', 'system'];

function isPreference(value: unknown): value is ThemePreference {
  return PREFERENCES.includes(value as ThemePreference);
}

/**
 * Holds the light/dark/system choice, applies it to <html data-theme> and remembers it in
 * localStorage (used before login, F0) and in the account settings (F12): after login the account
 * choice wins; a change is applied at once and saved to both. Storage failures never break the page.
 */
@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT);
  private readonly auth = inject(AuthService);
  private readonly settings = inject(SettingsApiService);
  private readonly storage = this.document.defaultView?.localStorage;
  private readonly media = this.document.defaultView?.matchMedia?.('(prefers-color-scheme: dark)');

  private readonly systemDark = signal(this.media?.matches ?? false);
  private readonly _preference = signal<ThemePreference>(this.readStored());

  private readonly _saveState = signal<ThemeSaveState>('idle');

  readonly preference = this._preference.asReadonly();
  readonly saveState = this._saveState.asReadonly();
  readonly resolved = computed<ResolvedTheme>(() => {
    const pref = this._preference();
    if (pref === 'system') {
      return this.systemDark() ? 'dark' : 'light';
    }
    return pref;
  });

  constructor() {
    const onChange = (e: { matches: boolean }) => this.systemDark.set(e.matches);
    this.media?.addEventListener('change', onChange);
    inject(DestroyRef).onDestroy(() => this.media?.removeEventListener('change', onChange));

    effect(() => {
      const root = this.document.documentElement;
      const theme = this.resolved();
      root.setAttribute('data-theme', theme);
      root.style.colorScheme = theme;
    });

    effect(() => this.writeStored(this._preference()));

    // After login (or a restored session) the account choice replaces the local one.
    effect(() => {
      if (this.auth.currentUser()?.id) {
        untracked(() => this.loadFromAccount());
      }
    });
  }

  /** Applies a choice at once and saves it (to the account too when logged in). */
  set(preference: ThemePreference): void {
    this._preference.set(preference);
    if (!this.auth.isLoggedIn()) {
      return;
    }
    this._saveState.set('saving');
    this.settings.update({ theme: preference }).subscribe({
      next: () => this._saveState.set('saved'),
      // The choice stays applied and stored locally; the settings page shows the error.
      error: () => this._saveState.set('error'),
    });
  }

  private loadFromAccount(): void {
    this.settings.get().subscribe({
      next: (s) => {
        if (isPreference(s.theme)) {
          this._preference.set(s.theme);
        }
      },
      // Offline: keep the local choice.
      error: () => undefined,
    });
  }

  private readStored(): ThemePreference {
    try {
      const value = this.storage?.getItem(THEME_STORAGE_KEY);
      return isPreference(value) ? value : 'system';
    } catch {
      return 'system';
    }
  }

  private writeStored(preference: ThemePreference): void {
    try {
      this.storage?.setItem(THEME_STORAGE_KEY, preference);
    } catch {
      // Storage blocked (private mode, disabled site data): the choice still applies for this visit.
    }
  }
}
