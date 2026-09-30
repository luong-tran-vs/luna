import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { Dashboard, Stats } from '../models/dashboard';

/** The home dashboard and the stats page (F6). Never cached: every call reads fresh numbers. */
@Injectable({ providedIn: 'root' })
export class DashboardApiService {
  private readonly http = inject(HttpClient);

  dashboard(): Observable<Dashboard> {
    return this.http.get<Dashboard>('/api/dashboard');
  }

  stats(): Observable<Stats> {
    return this.http.get<Stats>('/api/stats');
  }
}
