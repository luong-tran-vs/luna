import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom, Observable } from 'rxjs';

import { StudyApiService } from './study-api.service';

describe('StudyApiService', () => {
  let api: StudyApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(StudyApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  const call = async <T>(obs: Observable<T>, method: string, url: string, response: unknown, body?: unknown) => {
    const result = firstValueFrom(obs);
    const req = http.expectOne((r) => r.urlWithParams === url);
    expect(req.request.method).toBe(method);
    if (body !== undefined) {
      expect(req.request.body).toEqual(body);
    }
    req.flush(response as object);
    return result;
  };

  it('lists the topics of a level', async () => {
    await expect(call(api.topics('A1'), 'GET', '/api/topics?level=A1', { topics: [{ id: 't1' }] })).resolves.toEqual([{ id: 't1' }]);
  });

  it('reads and sets goals', async () => {
    const goals = { active: null, others: [] };
    await expect(call(api.goals(), 'GET', '/api/goals', goals)).resolves.toEqual(goals);
    const result = { active: {}, effectiveFrom: '2026-10-01', startsTomorrow: true };
    await expect(call(api.setGoal('t1'), 'POST', '/api/goals', result, { topicId: 't1' })).resolves.toEqual(result);
  });

  it("reads today's lesson, completes steps and saves the position", async () => {
    const today = { kind: 'noGoal' };
    await expect(call(api.today(), 'GET', '/api/today', today)).resolves.toEqual(today);
    await expect(call(api.completeStep('read'), 'POST', '/api/today/steps/read/complete', today)).resolves.toEqual(today);
    await call(api.savePosition('listen', 3), 'PUT', '/api/today/position', null, { step: 'listen', sentenceIndex: 3 });
  });

  it("lists the learner's lessons", async () => {
    const mine = { today: null, completed: [], upcoming: [] };
    await expect(call(api.myLessons(), 'GET', '/api/lessons/mine', mine)).resolves.toEqual(mine);
  });
});
