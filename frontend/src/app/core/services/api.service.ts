import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, timeout } from 'rxjs';

import { HealthResponse } from '../models/health';

/** Upper bound for the health check: 2 s database ping plus network latency. */
const HEALTH_TIMEOUT_MS = 5000;

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);

  getHealth(): Observable<HealthResponse> {
    return this.http.get<HealthResponse>('/api/health').pipe(timeout(HEALTH_TIMEOUT_MS));
  }
}
