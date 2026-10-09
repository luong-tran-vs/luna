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
    await expect(call(api.topics('A1'), 'GET', '/topics?level=A1', { topics: [{ id: 't1' }] })).resolves.toEqual([{ id: 't1' }]);
  });

  it('reads and sets goals', async () => {
    const goals = { active: null, others: [] };
    await expect(call(api.goals(), 'GET', '/goals', goals)).resolves.toEqual(goals);
    const result = { active: {} };
    await expect(call(api.setGoal('t1', 'A2'), 'POST', '/goals', result, { topicId: 't1', level: 'A2' })).resolves.toEqual(result);
  });

  it("reads a lesson's steps, completes them and saves the position", async () => {
    const study = { status: 'studying' };
    await expect(call(api.lessonStudy('l1'), 'GET', '/lessons/l1/study', study)).resolves.toEqual(study);
    await expect(call(api.completeStep('l1', 'read'), 'POST', '/lessons/l1/steps/read/complete', study)).resolves.toEqual(study);
    await expect(call(api.skipWrite('l1'), 'POST', '/lessons/l1/steps/write/skip', study)).resolves.toEqual(study);
    await call(api.savePosition('l1', 'listen', 3), 'PUT', '/lessons/l1/position', null, { step: 'listen', sentenceIndex: 3 });
  });

  it("lists the learner's lessons", async () => {
    const mine = { current: null, completed: [], upcoming: [] };
    await expect(call(api.myLessons(), 'GET', '/lessons/mine', mine)).resolves.toEqual(mine);
  });
});
