import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { GrammarApiService } from './grammar-api.service';

describe('GrammarApiService', () => {
  let api: GrammarApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(GrammarApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('lists the points of a level, or of all levels without one', async () => {
    const list = firstValueFrom(api.points('A2'));
    const req = http.expectOne((r) => r.url === '/api/grammar');
    expect(req.request.method).toBe('GET');
    expect(req.request.params.get('level')).toBe('A2');
    req.flush({ points: [], next: '' });
    await expect(list).resolves.toEqual({ points: [], next: '' });

    const all = firstValueFrom(api.points());
    const req2 = http.expectOne((r) => r.url === '/api/grammar');
    expect(req2.request.params.has('level')).toBe(false);
    req2.flush({ points: [], next: '' });
    await all;
  });

  it('loads one point', async () => {
    const detail = firstValueFrom(api.get('a1-to-be'));
    http.expectOne('/api/grammar/a1-to-be').flush({ point: { id: 'a1-to-be' }, lessonCount: 2 });
    await expect(detail).resolves.toMatchObject({ lessonCount: 2 });
  });

  it('reports an exercise and remembers it for the session', async () => {
    const body = { exerciseId: 'p3', reason: 'typo' as const, note: 'sai chính tả' };
    const result = firstValueFrom(api.reportExercise('a1-to-be', body));
    const req = http.expectOne('/api/grammar/a1-to-be/reports');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush({ ok: true });
    await expect(result).resolves.toEqual({ ok: true });

    expect(api.reported().has('a1-to-be/p3')).toBe(false);
    api.markReported('a1-to-be', 'p3');
    expect(api.reported().has('a1-to-be/p3')).toBe(true);
  });

  it('posts an attempt', async () => {
    const body = { kind: 'mastery' as const, correct: 4, total: 5, wrong: ['m3'] };
    const result = firstValueFrom(api.recordAttempt('a1-to-be', body));
    const req = http.expectOne('/api/grammar/a1-to-be/attempts');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush({ progress: { status: 'mastered' }, passed: true });
    await expect(result).resolves.toMatchObject({ passed: true });
  });
});
