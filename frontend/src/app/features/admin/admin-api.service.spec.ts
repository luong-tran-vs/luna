import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom, Observable } from 'rxjs';

import { AdminApiService } from './admin-api.service';

describe('AdminApiService', () => {
  let api: AdminApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    api = TestBed.inject(AdminApiService);
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

  const lesson = { id: 'l1', title: 'T' };
  const input = { title: 'T', content: 'C.', topicId: 't1', source: 's', license: 'l' };

  it('lists lessons with filters, skipping empty ones', async () => {
    await expect(
      call(api.list({ level: 'B1', topicId: 't1' }), 'GET', '/api/admin/lessons?level=B1&topicId=t1', { lessons: [lesson] }),
    ).resolves.toEqual([lesson]);
    await call(api.list({ level: '', topicId: '' }), 'GET', '/api/admin/lessons', { lessons: [] });
  });

  it('gets, creates, updates and deletes a lesson', async () => {
    await expect(call(api.get('l1'), 'GET', '/api/admin/lessons/l1', { lesson })).resolves.toEqual(lesson);
    await expect(call(api.create(input), 'POST', '/api/admin/lessons', { lesson }, input)).resolves.toEqual(lesson);
    await expect(call(api.update('l1', input), 'PUT', '/api/admin/lessons/l1', { lesson }, input)).resolves.toEqual(lesson);
    await call(api.remove('l1'), 'DELETE', '/api/admin/lessons/l1', null);
  });

  it('retries a job and saves annotations', async () => {
    await expect(call(api.retry('l1', 'annotate'), 'POST', '/api/admin/lessons/l1/retry?job=annotate', { lesson })).resolves.toEqual(lesson);
    const items = [{ text: 'went', lemma: 'go', meaningVi: 'đi' }];
    await expect(
      call(api.saveAnnotations('l1', items), 'PUT', '/api/admin/lessons/l1/annotations', { lesson }, { annotations: items }),
    ).resolves.toEqual(lesson);
  });

  it('regenerates the practice', async () => {
    await expect(
      call(api.regeneratePractice('l1'), 'POST', '/api/admin/lessons/l1/practice/regenerate', { lesson }),
    ).resolves.toEqual(lesson);
  });

  it('saves the questions, grammar note and writing prompt', async () => {
    const extras = { questions: [], grammarNote: null, writingPrompt: 'Write.' };
    await expect(
      call(api.updateExtras('l1', extras), 'PUT', '/api/admin/lessons/l1/extras', { lesson }, extras),
    ).resolves.toEqual(lesson);
  });

  it('lists, creates, updates and deletes topics', async () => {
    const topic = { id: 't1', name: 'Gia đình' };
    const topicInput = { name: 'Gia đình', level: 'A1' as const, description: '' };
    await expect(call(api.topics(), 'GET', '/api/admin/topics', { topics: [topic] })).resolves.toEqual([topic]);
    await call(api.topics('A1'), 'GET', '/api/admin/topics?level=A1', { topics: [] });
    await expect(call(api.createTopic(topicInput), 'POST', '/api/admin/topics', { topic }, topicInput)).resolves.toEqual(topic);
    await expect(call(api.updateTopic('t1', topicInput), 'PUT', '/api/admin/topics/t1', { topic }, topicInput)).resolves.toEqual(topic);
    await call(api.deleteTopic('t1'), 'DELETE', '/api/admin/topics/t1', null);
  });

  it('reads and saves a topic roadmap', async () => {
    const roadmap = { topic: {}, lessons: [], remaining: 0, warning: true };
    await expect(call(api.topicRoadmap('t1'), 'GET', '/api/admin/topics/t1/roadmap', roadmap)).resolves.toEqual(roadmap);
    await expect(
      call(api.setTopicRoadmap('t1', ['a', 'b']), 'PUT', '/api/admin/topics/t1/roadmap', roadmap, { lessonIds: ['a', 'b'] }),
    ).resolves.toEqual(roadmap);
  });

  it('generates lesson drafts for a topic with target words', async () => {
    const result = {
      drafts: [{ title: 'T', content: 'C.', words: 1, targetWords: ['Family'], missingWords: [] }],
      requested: 1,
      dropped: 0,
    };
    const body = { count: 1, words: 120, kind: 'reading' as const, idea: '', targetWords: [['Family']] };
    await expect(
      call(api.generateLessons('t1', body), 'POST', '/api/admin/topics/t1/generate', result, body),
    ).resolves.toEqual(result);
  });

  it('reads and replaces the vocabulary of a topic (F18)', async () => {
    const words = [{ text: 'Family', used: true, lessonCount: 2 }];
    await expect(call(api.topicWords('t1'), 'GET', '/api/admin/topics/t1/words', { words })).resolves.toEqual(words);
    await expect(
      call(api.setTopicWords('t1', ['Family', 'cousin']), 'PUT', '/api/admin/topics/t1/words', { words }, {
        words: ['Family', 'cousin'],
      }),
    ).resolves.toEqual(words);
  });

  it('asks for a split of target words (F18)', async () => {
    const groups = [['Family', 'cousin'], []];
    await expect(
      call(api.wordPlan('t1', 2, 8), 'GET', '/api/admin/topics/t1/word-plan?count=2&perLesson=8', { groups }),
    ).resolves.toEqual(groups);
  });
});
