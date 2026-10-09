import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { Settings, SettingsPatch } from '../models/settings';

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
}
