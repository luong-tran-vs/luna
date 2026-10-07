import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { LessonSummary } from '../../../core/models/lesson';
import { Topic, TopicLevel, TopicRoadmap } from '../../../core/models/topic';
import { LessonList } from './lesson-list';

const summary = (over: Partial<LessonSummary>): LessonSummary => ({
  id: 'l1',
  title: 'A day at the park',
  level: 'B1',
  topicId: 't3',
  topicName: 'Công việc',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '2026-09-29T08:00:00Z',
  ...over,
});

/** A topic with enough lessons in the roadmap of each given level. */
const topic = (id: string, name: string, levels: TopicLevel[] = [], over: Partial<Topic> = {}): Topic => ({
  id, name, description: '', lessonCount: 5, levels, createdAt: '', wordCount: 0, usedWordCount: 0, ...over,
});
const lv = (level: TopicLevel['level'], remaining = 5): TopicLevel => ({
  level, lessonCount: 5, roadmapCount: remaining, remaining, warning: remaining < 3,
});

const topics = [topic('t1', 'Gia đình', [lv('A1')]), topic('t2', 'Mua sắm', [lv('A1')]), topic('t3', 'Công việc', [lv('B1')])];

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

  it('shows the AI check state in words (F22)', async () => {
    await flushLoad([
      summary({ id: 'a', title: 'Cờ', flags: 3, checked: true }),
      summary({ id: 'b', title: 'Xong', flags: 0, checked: true, verified: true }),
      summary({ id: 'c', title: 'Mới', flags: 0, checked: false, verified: false }),
      summary({ id: 'd', title: 'Đã chạy', flags: 0, checked: true, verified: false }),
    ]);
    const items = Array.from(el.querySelectorAll('li.item')).map((li) => li.textContent ?? '');
    expect(items[0]).toContain('3 chỗ cần xem');
    expect(items[1]).toContain('Đã kiểm tra');
    expect(items[2]).toContain('Chưa kiểm tra');
    expect(items[3]).not.toContain('Chưa kiểm tra');
    expect(items[3]).not.toContain('chỗ cần xem');
  });

  it('renders lessons with level, topic, status chips and links', async () => {
    await flushLoad([summary({ annotationStatus: 'failed' })]);

    const link = el.querySelector<HTMLAnchorElement>('a[href="/admin/lessons/l1"]');
    expect(link?.textContent?.trim()).toBe('A day at the park');
    expect(el.textContent).toContain('B1');
    expect(el.textContent).toContain('Công việc');
    expect(el.textContent).not.toContain('Audio');
    expect(el.textContent).toContain('Chú thích: Lỗi');
    expect(el.querySelector('a[href="/admin/lessons/new"]')).not.toBeNull();
  });

  it('shows an empty state', async () => {
    await flushLoad([]);
    expect(el.textContent).toContain('Chưa có bài học');
  });

  it('filters by level and by topic, independently', async () => {
    await flushLoad([summary({})]);
    expect(Array.from(select('topicId').options).map((o) => o.textContent?.trim())).toEqual([
      'Tất cả',
      'Công việc',
      'Gia đình',
      'Mua sắm',
    ]);

    await choose('topicId', 't3');
    const byTopic: TestRequest = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(byTopic.request.params.get('topicId')).toBe('t3');
    await flushLoadRest(byTopic);

    // A topic is shared by every level: choosing a level keeps the topic and every topic choice.
    await choose('level', 'A1');
    const both = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(both.request.params.get('level')).toBe('A1');
    expect(both.request.params.get('topicId')).toBe('t3');
    await flushLoadRest(both);
    expect(Array.from(select('topicId').options).map((o) => o.value)).toEqual(['', 't3', 't1', 't2']);
  });

  async function flushLoadRest(req: TestRequest): Promise<void> {
    req.flush({ lessons: [] });
    http.expectOne('/api/admin/topics').flush({ topics });
    await settle();
  }

  it('offers retry only for failed work and reloads after it', async () => {
    await flushLoad([summary({ annotationStatus: 'failed' })]);

    button('Chạy lại chú thích')!.click();
    await settle();
    const req = http.expectOne('/api/admin/lessons/l1/retry?job=annotate');
    expect(req.request.method).toBe('POST');
    req.flush({ lesson: summary({ annotationStatus: 'running' }) });
    await settle();
    await flushLoad([summary({ annotationStatus: 'done' })]);
    expect(button('Chạy lại chú thích')).toBeUndefined();
  });

  it('adds a lesson to the end of the roadmap of its topic and level', async () => {
    await flushLoad([summary({}), summary({ id: 'l0', title: 'Old', inRoadmap: true })]);
    expect(el.textContent).toContain('Trong lộ trình');

    button('Thêm vào lộ trình')!.click();
    await settle();
    const roadmap: TopicRoadmap = {
      topic: topics[2], level: 'B1', lessons: [summary({ id: 'l0', inRoadmap: true })], remaining: 1, warning: true,
    };
    http.expectOne('/api/admin/topics/t3/roadmap?level=B1').flush(roadmap);
    await settle();
    const put = http.expectOne((r) => r.url === '/api/admin/topics/t3/roadmap' && r.method === 'PUT');
    expect(put.request.params.get('level')).toBe('B1');
    expect(put.request.body).toEqual({ lessonIds: ['l0', 'l1'] });
    put.flush(roadmap);
    await settle();
    await flushLoad([summary({ inRoadmap: true }), summary({ id: 'l0', title: 'Old', inRoadmap: true })]);
  });

  it('points to the roadmap page when roadmaps run low', async () => {
    await flushLoad([], [topic('t1', 'Gia đình', [lv('A1', 2), lv('B1', 0)]), topic('t2', 'Mua sắm', [lv('A1')]), topics[2]]);
    const banner = el.querySelector('.banner-warning')!;
    expect(banner.textContent).toContain('2 lộ trình sắp hết bài chưa học');
    expect(banner.querySelector('a[href="/admin/roadmap"]')).not.toBeNull();
  });

  it('shows no warning when every topic has enough lessons', async () => {
    await flushLoad([], [topics[2]]);
    expect(el.querySelector('.banner-warning')).toBeNull();
  });

  it('polls every 5 seconds while work is running', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
    try {
      await flushLoad([summary({ annotationStatus: 'running' })]);
      vi.advanceTimersByTime(5000);
      await flushLoad([summary({ annotationStatus: 'done' })]);
      vi.advanceTimersByTime(10000);
      http.expectNone((r) => r.url === '/api/admin/lessons');
    } finally {
      vi.useRealTimers();
    }
  });
});
