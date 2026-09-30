import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { LessonSummary } from '../../../core/models/lesson';
import { Topic, TopicRoadmap } from '../../../core/models/topic';
import { LessonList } from './lesson-list';

const summary = (over: Partial<LessonSummary>): LessonSummary => ({
  id: 'l1',
  title: 'A day at the park',
  level: 'B1',
  topicId: 't3',
  topicName: 'Công việc',
  audioStatus: 'done',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '2026-09-29T08:00:00Z',
  ...over,
});

const topic = (id: string, name: string, level: Topic['level'], over: Partial<Topic> = {}): Topic => ({
  id, name, level, description: '', lessonCount: 1, roadmapCount: 5, remaining: 5, warning: false, createdAt: '', ...over,
});

const topics = [topic('t1', 'Gia đình', 'A1'), topic('t2', 'Mua sắm', 'A1'), topic('t3', 'Công việc', 'B1')];

describe('LessonList', () => {
  let fixture: ComponentFixture<LessonList>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const flushLoad = async (lessons: LessonSummary[], topicList: Topic[] = topics) => {
    http.expectOne((r) => r.url === '/api/admin/lessons').flush({ lessons });
    http.expectOne('/api/admin/topics').flush({ topics: topicList });
    await settle();
  };
  const button = (text: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === text);
  const select = (name: string) => el.querySelector<HTMLSelectElement>(`select[name="${name}"]`)!;
  const choose = async (name: string, value: string) => {
    select(name).value = value;
    select(name).dispatchEvent(new Event('change'));
    await settle();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LessonList],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonList);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('renders lessons with level, topic, status chips and links', async () => {
    await flushLoad([summary({ audioStatus: 'failed' })]);

    const link = el.querySelector<HTMLAnchorElement>('a[href="/admin/lessons/l1"]');
    expect(link?.textContent?.trim()).toBe('A day at the park');
    expect(el.textContent).toContain('B1');
    expect(el.textContent).toContain('Công việc');
    expect(el.textContent).toContain('Audio: Lỗi');
    expect(el.textContent).toContain('Chú thích: Xong');
    expect(el.querySelector('a[href="/admin/lessons/new"]')).not.toBeNull();
    expect(el.querySelector('a[href="/admin/roadmap"]')).not.toBeNull();
    expect(el.querySelector('a[href="/admin/topics"]')?.textContent?.trim()).toBe('Chủ đề');
  });

  it('shows an empty state', async () => {
    await flushLoad([]);
    expect(el.textContent).toContain('Chưa có bài học');
  });

  it('filters by level and by topic of that level', async () => {
    await flushLoad([summary({})]);
    expect(Array.from(select('topicId').options).map((o) => o.textContent?.trim())).toEqual([
      'Tất cả',
      'A1 · Gia đình',
      'A1 · Mua sắm',
      'B1 · Công việc',
    ]);

    await choose('topicId', 't3');
    const byTopic: TestRequest = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(byTopic.request.params.get('topicId')).toBe('t3');
    await flushLoadRest(byTopic);

    // Choosing A1 keeps only A1 topics and clears the B1 topic.
    await choose('level', 'A1');
    const byLevel = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(byLevel.request.params.get('level')).toBe('A1');
    expect(byLevel.request.params.has('topicId')).toBe(false);
    await flushLoadRest(byLevel);
    expect(Array.from(select('topicId').options).map((o) => o.value)).toEqual(['', 't1', 't2']);

    await choose('topicId', 't1');
    const both = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(both.request.params.get('level')).toBe('A1');
    expect(both.request.params.get('topicId')).toBe('t1');
    await flushLoadRest(both);
  });

  async function flushLoadRest(req: TestRequest): Promise<void> {
    req.flush({ lessons: [] });
    http.expectOne('/api/admin/topics').flush({ topics });
    await settle();
  }

  it('offers retry only for failed work and reloads after it', async () => {
    await flushLoad([summary({ audioStatus: 'failed', annotationStatus: 'done' })]);
    expect(button('Chạy lại chú thích')).toBeUndefined();

    button('Chạy lại audio')!.click();
    await settle();
    const req = http.expectOne('/api/admin/lessons/l1/retry?job=tts');
    expect(req.request.method).toBe('POST');
    req.flush({ lesson: summary({ audioStatus: 'running' }) });
    await settle();
    await flushLoad([summary({ audioStatus: 'done' })]);
    expect(button('Chạy lại audio')).toBeUndefined();
  });

  it("adds a lesson to the end of its topic's roadmap", async () => {
    await flushLoad([summary({}), summary({ id: 'l0', title: 'Old', inRoadmap: true })]);
    expect(el.textContent).toContain('Trong lộ trình');

    button('Thêm vào lộ trình')!.click();
    await settle();
    const roadmap: TopicRoadmap = {
      topic: topics[2], lessons: [summary({ id: 'l0', inRoadmap: true })], remaining: 1, warning: true,
    };
    http.expectOne('/api/admin/topics/t3/roadmap').flush(roadmap);
    await settle();
    const put = http.expectOne((r) => r.url === '/api/admin/topics/t3/roadmap' && r.method === 'PUT');
    expect(put.request.body).toEqual({ lessonIds: ['l0', 'l1'] });
    put.flush(roadmap);
    await settle();
    await flushLoad([summary({ inRoadmap: true }), summary({ id: 'l0', title: 'Old', inRoadmap: true })]);
  });

  it('points to the roadmap page when topics run low', async () => {
    await flushLoad([], [topic('t1', 'Gia đình', 'A1', { remaining: 2, warning: true }), topic('t2', 'Mua sắm', 'A1', { remaining: 0, warning: true }), topics[2]]);
    const banner = el.querySelector('.banner-warning')!;
    expect(banner.textContent).toContain('2 chủ đề sắp hết bài chưa học');
    expect(banner.querySelector('a[href="/admin/roadmap"]')).not.toBeNull();
  });

  it('shows no warning when every topic has enough lessons', async () => {
    await flushLoad([], [topics[2]]);
    expect(el.querySelector('.banner-warning')).toBeNull();
  });

  it('polls every 5 seconds while work is running', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
    try {
      await flushLoad([summary({ audioStatus: 'running' })]);
      vi.advanceTimersByTime(5000);
      await flushLoad([summary({ audioStatus: 'done' })]);
      vi.advanceTimersByTime(10000);
      http.expectNone((r) => r.url === '/api/admin/lessons');
    } finally {
      vi.useRealTimers();
    }
  });
});
