import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { VerbsApiService } from './verbs-api.service';

describe('VerbsApiService', () => {
  let api: VerbsApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    api = TestBed.inject(VerbsApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('reads the progress and posts a round', async () => {
    const list = firstValueFrom(api.progress());
    http.expectOne({ method: 'GET', url: '/verbs/progress' }).flush({ verbs: [] });
    await expect(list).resolves.toEqual({ verbs: [] });

    const saved = firstValueFrom(api.record([{ base: 'go', right: true }]));
    const req = http.expectOne({ method: 'POST', url: '/verbs/results' });
    expect(req.request.body).toEqual({ results: [{ base: 'go', right: true }] });
    req.flush({ verbs: [{ base: 'go', level: 1, dueAt: null, right: 1, wrong: 0 }] });
    await expect(saved).resolves.toMatchObject({ verbs: [{ base: 'go', level: 1 }] });
  });
});
