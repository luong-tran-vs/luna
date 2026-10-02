import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { CardInput } from '../models/vocab';
import { VocabApiService } from './vocab-api.service';

describe('VocabApiService', () => {
  let api: VocabApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(VocabApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('saves a card and lists saved words', async () => {
    const input: CardInput = {
      text: 'went', lemma: 'go', ipa: '', meaningVi: 'đã đi', contextSentence: 'We went.', lessonId: 'l1', source: 'ai',
    };
    const saved = firstValueFrom(api.saveCard(input));
    const req = http.expectOne('/api/vocab/cards');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(input);
    req.flush({ card: { id: 'c1', ...input } });
    await expect(saved).resolves.toMatchObject({ id: 'c1' });

    const words = firstValueFrom(api.words());
    http.expectOne('/api/vocab/words').flush({ words: [{ lemma: 'go', text: 'went' }] });
    await expect(words).resolves.toEqual([{ lemma: 'go', text: 'went' }]);
  });

  it('loads due cards with a limit', async () => {
    const due = firstValueFrom(api.due(20));
    const req = http.expectOne((r) => r.url === '/api/vocab/review/due');
    expect(req.request.params.get('limit')).toBe('20');
    req.flush({ cards: [], total: 0, nextDue: null });
    await expect(due).resolves.toEqual({ cards: [], total: 0, nextDue: null });
  });

  it('sends a review and unwraps the card', async () => {
    const reviewed = firstValueFrom(api.review('c1', { rating: 3, mode: 'flip', reps: 0, context: 'free' }));
    const req = http.expectOne('/api/vocab/cards/c1/review');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'free' });
    req.flush({ card: { id: 'c1', reps: 1 } });
    await expect(reviewed).resolves.toMatchObject({ id: 'c1', reps: 1 });
  });

  it('lists notebook pages with filters', async () => {
    const page = firstValueFrom(api.list({ q: 'giv', lessonId: 'manual', page: 2 }));
    const req = http.expectOne((r) => r.url === '/api/vocab/cards');
    expect(req.request.params.get('q')).toBe('giv');
    expect(req.request.params.get('lessonId')).toBe('manual');
    expect(req.request.params.get('page')).toBe('2');
    req.flush({ cards: [], hasMore: false, today: '2026-09-30', yesterday: '2026-09-29' });
    await expect(page).resolves.toMatchObject({ hasMore: false });

    const unfiltered = firstValueFrom(api.list({}));
    const plain = http.expectOne((r) => r.url === '/api/vocab/cards');
    expect(plain.request.params.keys()).toEqual(['page']);
    plain.flush({ cards: [], hasMore: false, today: '', yesterday: '' });
    await unfiltered;
  });

  it('lists lessons, updates and deletes cards', async () => {
    const lessons = firstValueFrom(api.lessons());
    http.expectOne('/api/vocab/lessons').flush({ lessons: [], manualCount: 2 });
    await expect(lessons).resolves.toEqual({ lessons: [], manualCount: 2 });

    const updated = firstValueFrom(api.update('c1', { meaningVi: 'đi' }));
    const patch = http.expectOne('/api/vocab/cards/c1');
    expect(patch.request.method).toBe('PATCH');
    expect(patch.request.body).toEqual({ meaningVi: 'đi' });
    patch.flush({ card: { id: 'c1', meaningVi: 'đi' } });
    await expect(updated).resolves.toMatchObject({ meaningVi: 'đi' });

    const removed = firstValueFrom(api.remove('c1'));
    const del = http.expectOne('/api/vocab/cards/c1');
    expect(del.request.method).toBe('DELETE');
    del.flush(null, { status: 204, statusText: 'No Content' });
    await removed;
  });

  it('saves lesson words in bulk', async () => {
    const result = firstValueFrom(api.bulk('l1', ['go', 'give up']));
    const req = http.expectOne('/api/vocab/cards/bulk');
    expect(req.request.body).toEqual({ lessonId: 'l1', lemmas: ['go', 'give up'] });
    req.flush({ added: 2, cards: [] });
    await expect(result).resolves.toEqual({ added: 2, cards: [] });
  });
});
