import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { DictationSummary } from '../../core/models/dictation';
import { ListeningApiService } from './listening-api.service';

const summary: DictationSummary = {
  sentenceCount: 2, checkedCount: 1, correctWords: 4, totalWords: 5, rate: 0.8, completed: false, results: [],
};

describe('ListeningApiService', () => {
  let api: ListeningApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(ListeningApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('loads the dictation summary', async () => {
    const result = firstValueFrom(api.summary('l1'));
    http.expectOne('/lessons/l1/dictation/summary').flush({ summary });
    await expect(result).resolves.toEqual(summary);
  });

  it('records a checked sentence and returns the new summary', async () => {
    const input = { sentenceIndex: 0, typed: 'we went', correctWords: 2, totalWords: 2 };
    const result = firstValueFrom(api.record('l1', input));
    const req = http.expectOne('/lessons/l1/dictation');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(input);
    req.flush({ summary });
    await expect(result).resolves.toEqual(summary);
  });

  it('forgets the dictation results of a lesson', async () => {
    const result = firstValueFrom(api.reset('l1'), { defaultValue: undefined });
    const req = http.expectOne('/lessons/l1/dictation');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await expect(result).resolves.toBeNull();
  });
});
