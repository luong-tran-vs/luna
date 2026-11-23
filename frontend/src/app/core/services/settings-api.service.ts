import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { Settings, SettingsPatch } from '../models/settings';

/** A part of the app whose progress the learner can reset from the account tab. */
export type ResetPart = 'lessons' | 'grammar' | 'vocab' | 'toeic';

/** The learner's settings (F12), stored with the account. */
@Injectable({ providedIn: 'root' })
export class SettingsApiService {
  private readonly http = inject(HttpClient);

  get(): Observable<Settings> {
    return this.http.get<Settings>('/settings');
  }

  /** Changes only the fields given; returns the full settings. */
  update(patch: SettingsPatch): Observable<Settings> {
    return this.http.put<Settings>('/settings', patch);
  }

  /**
   * Starts the ticked parts over (added 2026-11-23): their progress is removed on the server.
   * "vocab" keeps the saved words and makes their review cards new again.
   */
  resetProgress(parts: readonly ResetPart[]): Observable<void> {
    return this.http.post<void>('/me/reset', { parts });
  }
}
