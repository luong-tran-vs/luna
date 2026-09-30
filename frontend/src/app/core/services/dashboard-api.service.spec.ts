import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { DashboardApiService } from './dashboard-api.service';

describe('DashboardApiService', () => {
  let api: DashboardApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(DashboardApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('reads the dashboard', async () => {
    const result = firstValueFrom(api.dashboard());
    const req = http.expectOne('/api/dashboard');
    expect(req.request.method).toBe('GET');
    req.flush({ kind: 'noGoal', streak: 0 });
    await expect(result).resolves.toEqual({ kind: 'noGoal', streak: 0 });
  });

  it('reads the stats', async () => {
    const result = firstValueFrom(api.stats());
    const req = http.expectOne('/api/stats');
    expect(req.request.method).toBe('GET');
    req.flush({ cards: 25 });
    await expect(result).resolves.toEqual({ cards: 25 });
  });

  it('asks the server again on every call', () => {
    api.dashboard().subscribe();
    api.dashboard().subscribe();
    expect(http.match('/api/dashboard').length).toBe(2);
    http.match('/api/dashboard');
  });
});
