import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom, Observable } from 'rxjs';

import { WritingApiService } from './writing-api.service';

describe('WritingApiService', () => {
  let api: WritingApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(WritingApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  const call = async <T>(obs: Observable<T>, method: string, url: string, response: unknown, body?: unknown) => {
    const result = firstValueFrom(obs);
    const req = http.expectOne(url);
    expect(req.request.method).toBe(method);
    if (body !== undefined) {
      expect(req.request.body).toEqual(body);
    }
    req.flush(response as object);
    return result;
  };
  const writing = { id: 'w1' };

  it('reads, saves and submits the writing of a lesson', async () => {
    const view = { prompt: 'P', level: 'A1', canWrite: true, writing: null };
    await expect(call(api.lessonWriting('l1'), 'GET', '/api/lessons/l1/writing', view)).resolves.toEqual(view);
    await expect(
      call(api.saveDraft('l1', 'draft'), 'PUT', '/api/lessons/l1/writing', { writing }, { text: 'draft' }),
    ).resolves.toEqual(writing);
    await expect(
      call(api.submit('l1', 'text'), 'POST', '/api/lessons/l1/writing/submit', { writing }, { text: 'text' }),
    ).resolves.toEqual(writing);
  });

  it('lists, opens, marks seen and regrades writings', async () => {
    await expect(call(api.list(), 'GET', '/api/writings', { writings: [] })).resolves.toEqual([]);
    await expect(call(api.get('w1'), 'GET', '/api/writings/w1', { writing })).resolves.toEqual(writing);
    await call(api.seen('w1'), 'POST', '/api/writings/w1/seen', null);
    await expect(call(api.regrade('w1'), 'POST', '/api/writings/w1/regrade', { writing })).resolves.toEqual(writing);
    const count = { unseen: 1, pending: 0, latest: { id: 'w1', status: 'done' } };
    await expect(call(api.unseenCount(), 'GET', '/api/writings/unseen-count', count)).resolves.toEqual(count);
  });
});
