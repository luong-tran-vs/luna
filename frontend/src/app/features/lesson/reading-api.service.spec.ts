import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { ReadingApiService } from './reading-api.service';

describe('ReadingApiService', () => {
  let api: ReadingApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(ReadingApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('loads a lesson for reading', async () => {
    const result = firstValueFrom(api.getLesson('l1'));
    http.expectOne('/api/lessons/l1').flush({ lesson: { id: 'l1' } });
    await expect(result).resolves.toEqual({ id: 'l1' });
  });

  it('looks up a word with its sentence', async () => {
    const result = firstValueFrom(api.lookup('l1', 'gave up', 2));
    const req = http.expectOne((r) => r.url === '/api/lessons/l1/lookup');
    expect(req.request.params.get('q')).toBe('gave up');
    expect(req.request.params.get('sentence')).toBe('2');
    req.flush({ source: 'ai', text: 'gave up', lemma: 'give up', ipa: '', meanings: [] });
    await expect(result).resolves.toMatchObject({ lemma: 'give up' });
  });

  it('asks the AI about a word of a sentence', async () => {
    const result = firstValueFrom(api.ask('l1', 'gave up', 1));
    const req = http.expectOne('/api/lessons/l1/ask');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ text: 'gave up', sentenceIndex: 1 });
    req.flush({
      result: { source: 'ai', text: 'gave up', lemma: 'give up', ipa: '', meanings: [{ pos: '', text: 'bỏ' }], note: 'n' },
      cached: true,
    });
    await expect(result).resolves.toMatchObject({ cached: true, result: { lemma: 'give up', note: 'n' } });
  });

  it('sends a comprehension answer', async () => {
    const result = firstValueFrom(api.answer('l1', { version: 2, questionIndex: 1, choice: 3 }));
    const req = http.expectOne('/api/lessons/l1/answers');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ version: 2, questionIndex: 1, choice: 3 });
    req.flush({ answer: { questionIndex: 1 }, answered: 1, total: 3, correct: 0 });
    await expect(result).resolves.toMatchObject({ total: 3 });
  });

  it('loads the lesson vocabulary', async () => {
    const result = firstValueFrom(api.vocabulary('l1'));
    http.expectOne('/api/lessons/l1/vocabulary').flush({ available: false, items: [] });
    await expect(result).resolves.toEqual({ available: false, items: [] });
  });

  it('loads the lesson practice', async () => {
    const result = firstValueFrom(api.practice('l1'));
    const req = http.expectOne('/api/lessons/l1/practice');
    expect(req.request.method).toBe('GET');
    req.flush({ status: 'none', lessonNumber: 0 });
    await expect(result).resolves.toMatchObject({ status: 'none' });
  });
});
