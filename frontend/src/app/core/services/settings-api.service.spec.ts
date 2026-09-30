import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { Settings } from '../models/settings';
import { SettingsApiService } from './settings-api.service';

const settings: Settings = { theme: 'dark', dailyReviewLimit: 20, timezone: 'Europe/London' };

describe('SettingsApiService', () => {
  let api: SettingsApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(SettingsApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('reads the settings', async () => {
    const result = firstValueFrom(api.get());
    const req = http.expectOne('/api/settings');
    expect(req.request.method).toBe('GET');
    req.flush(settings);
    await expect(result).resolves.toEqual(settings);
  });

  it('sends only the changed fields', async () => {
    const result = firstValueFrom(api.update({ theme: 'dark' }));
    const req = http.expectOne('/api/settings');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ theme: 'dark' });
    req.flush(settings);
    await expect(result).resolves.toEqual(settings);
  });
});
