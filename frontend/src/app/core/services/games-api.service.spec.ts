import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../interceptors/error-interceptor';
import { errorBody, errorStatus, GamesApiService, PENDING_KEY } from './games-api.service';

describe('GamesApiService', () => {
  let api: GamesApiService;
  let http: HttpTestingController;
  const body = { score: 900, completed: true, wrongWords: ['apple'], wrongVerbs: [] };

  beforeEach(() => {
    localStorage.removeItem(PENDING_KEY);
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(GamesApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
    localStorage.removeItem(PENDING_KEY);
  });

  it('reads the list and a round, and posts a result', async () => {
    const list = firstValueFrom(api.list());
    http.expectOne({ method: 'GET', url: '/games' }).flush({ games: [], daily: { day: 'd', done: false, streak: 0 } });
    await list;

    const mem = firstValueFrom(api.memoryRound('L1'));
    http.expectOne((r) => r.url === '/games/memory/round' && r.params.get('lessonId') === 'L1').flush({ pairs: [] });
    await mem;

    const done = firstValueFrom(api.finish('picture', body));
    const req = http.expectOne({ method: 'POST', url: '/games/picture/results' });
    expect(req.request.body).toEqual(body);
    req.flush({ best: 900, newBest: true, rescheduled: 1 });
    await expect(done).resolves.toEqual({ best: 900, newBest: true, rescheduled: 1 });
  });

  it('keeps a result the network lost and sends it later', async () => {
    const done = firstValueFrom(api.finish('picture', body));
    http.expectOne('/games/picture/results').error(new ProgressEvent('error'), { status: 0 });
    await expect(done).resolves.toBeNull();
    expect(api.pending()).toEqual([{ game: 'picture', body }]);

    const sent = firstValueFrom(api.flushPending());
    http.expectOne('/games/picture/results').flush({ best: 900, newBest: true, rescheduled: 1 });
    await expect(sent).resolves.toBe(1);
    expect(api.pending()).toEqual([]);
  });

  it('drops a kept result the server refuses and keeps those after a network failure', async () => {
    localStorage.setItem(
      PENDING_KEY,
      JSON.stringify([
        { game: 'grammar', body },
        { game: 'picture', body },
        { game: 'letters', body },
      ]),
    );
    const sent = firstValueFrom(api.flushPending());
    http
      .expectOne('/games/grammar/results')
      .flush({ error: 'members_only' }, { status: 403, statusText: 'Forbidden' });
    await Promise.resolve();
    http.expectOne('/games/picture/results').error(new ProgressEvent('error'), { status: 0 });
    await expect(sent).resolves.toBe(0);
    expect(api.pending().map((p) => p.game)).toEqual(['picture', 'letters']);
  });

  it('survives broken storage', () => {
    localStorage.setItem(PENDING_KEY, '{oops');
    expect(api.pending()).toEqual([]);
  });
});

describe('errorStatus', () => {
  it('reads the app error and a raw HTTP error', () => {
    expect(errorStatus(new ApiError('http', 409, { result: { score: 1 } }))).toBe(409);
    expect(errorBody(new ApiError('http', 409, { result: { score: 1 } }))).toEqual({ result: { score: 1 } });
    expect(errorStatus(new ApiError('network', 0, null))).toBe(0);
    expect(errorStatus(new Error('x'))).toBe(0);
  });
});
