import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { ExportApiService, filenameFrom } from './export-api.service';

describe('ExportApiService', () => {
  let api: ExportApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(ExportApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('downloads the export with the server file name', async () => {
    const result = firstValueFrom(api.download());
    const req = http.expectOne('/export');
    expect(req.request.method).toBe('GET');
    expect(req.request.responseType).toBe('blob');
    const blob = new Blob(['{}'], { type: 'application/json' });
    req.flush(blob, { headers: { 'Content-Disposition': 'attachment; filename="luna-export-20260930.json"' } });
    const file = await result;
    expect(file.filename).toBe('luna-export-20260930.json');
    expect(file.blob).toBe(blob);
  });

  it('falls back to a default name', () => {
    expect(filenameFrom(null)).toBe('luna-export.json');
    expect(filenameFrom('attachment')).toBe('luna-export.json');
    expect(filenameFrom('attachment; filename=luna-export-20261001.json')).toBe('luna-export-20261001.json');
  });
});
