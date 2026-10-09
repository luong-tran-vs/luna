import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { GrammarLessonSummary, GrammarReportGroup } from '../../../core/models/grammar-admin';
import { LessonSummary } from '../../../core/models/lesson';
import { Topic } from '../../../core/models/topic';
import { Dashboard, dayLabel, greeting, todoItems } from './dashboard';

const lesson = (id: string, over: Partial<LessonSummary> = {}): LessonSummary => ({
  id,
  title: `Bài ${id}`,
  level: 'A1',
  topicId: 't1',
  topicName: 'Gia đình',
  annotationStatus: 'done',
  inRoadmap: true,
  createdAt: '2026-10-01T08:00:00Z',
  checked: true,
  verified: true,
  ...over,
});

const topic = (id: string, over: Partial<Topic> = {}): Topic => ({
  id, name: id, description: '', lessonCount: 5, createdAt: '', wordCount: 0, usedWordCount: 0,
  levels: [{ level: 'A1', lessonCount: 5, roadmapCount: 5, remaining: 5, warning: false }],
  ...over,
});
const lowA2 = { level: 'A2' as const, lessonCount: 2, roadmapCount: 2, remaining: 2, warning: true };

const grammar = (pointId: string, status: GrammarLessonSummary['status']): GrammarLessonSummary => ({
  pointId, status, edited: false, updatedAt: '', publishedAt: null, flags: 0, checked: true, verified: true,
});

const report = (count: number): GrammarReportGroup => ({
  pointId: 'p1', exerciseId: 'e1', count, reasons: {}, notes: [], latestAt: '',
});

describe('dashboard helpers', () => {
  it('greets by the time of day', () => {
    expect(greeting(8)).toBe('Chào buổi sáng!');
    expect(greeting(11)).toBe('Chào buổi sáng!');
    expect(greeting(12)).toBe('Chào buổi chiều!');
    expect(greeting(14)).toBe('Chào buổi chiều!');
    expect(greeting(21)).toBe('Chào buổi tối!');
    expect(greeting(2)).toBe('Chào buổi tối!');
  });

  it('writes the day in Vietnamese', () => {
    expect(dayLabel(new Date(2026, 9, 7))).toMatch(/^Thứ .+, 07\/10\/2026$/);
  });

  it('lists only the jobs that have something to do', () => {
    const items = todoItems({
      lessons: [
        lesson('a', { flags: 2 }),
        lesson('b', { annotationStatus: 'failed', checked: false }),
        lesson('c', { inRoadmap: false, checked: false }),
        lesson('d'),
      ],
      topics: [topic('t1', { levels: [topic('x').levels[0], lowA2] }), topic('t2')],
      grammar: [grammar('p1', 'draft'), grammar('p2', 'published')],
      reports: [report(2), report(1)],
    });
    expect(items.map((i) => [i.count, i.label])).toEqual([
      [1, 'bài có chỗ AI đánh dấu cần xem'],
      [1, 'bài chú thích bị lỗi'],
      [1, 'bài chưa kiểm tra bằng AI'],
      [1, 'bài chưa vào lộ trình'],
      [1, 'lộ trình sắp hết bài chưa học'],
      [3, 'báo lỗi bài tập ngữ pháp từ người học'],
      [1, 'bài ngữ pháp còn là bản nháp'],
    ]);
  });
});

describe('Dashboard', () => {
  let fixture: ComponentFixture<Dashboard>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const flush = async (lessons: LessonSummary[], grammarLessons: GrammarLessonSummary[] = []) => {
    http.expectOne('/admin/lessons').flush({ lessons });
    http.expectOne('/admin/topics').flush({ topics: [topic('t1'), topic('t2')] });
    http.expectOne('/admin/grammar-lessons').flush({ lessons: grammarLessons });
    http.expectOne('/admin/grammar-reports').flush({ reports: [] });
    await settle();
  };
  const text = (selector: string) => Array.from(el.querySelectorAll(selector)).map((n) => n.textContent?.replace(/\s+/g, ' ').trim());

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Dashboard],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Dashboard);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('shows the counts, the checked share and the newest lessons first', async () => {
    await flush(
      [
        lesson('a', { createdAt: '2026-10-01T00:00:00Z' }),
        lesson('b', { createdAt: '2026-10-03T00:00:00Z', verified: false }),
        lesson('c', { createdAt: '2026-10-02T00:00:00Z', inRoadmap: false }),
        lesson('d', { createdAt: '2026-09-30T00:00:00Z' }),
      ],
      [grammar('p1', 'published'), grammar('p2', 'draft')],
    );

    expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('75');
    expect(el.textContent).toContain('Đã kiểm tra 3/4 bài · 3 bài đang trong lộ trình');
    expect(text('.stat-value')).toEqual(['4', '2', '1']);
    expect(text('.recent-title')).toEqual(['Bài b', 'Bài c', 'Bài a']);
    expect(text('.todo-count')).toEqual(['1', '1']);
    expect(text('.todo-label')).toEqual(['bài chưa vào lộ trình', 'bài ngữ pháp còn là bản nháp']);
    expect(el.querySelector('a.recent-item')?.getAttribute('href')).toBe('/admin/lessons/b');
  });

  it('says when nothing is pending', async () => {
    await flush([lesson('a')]);
    expect(el.textContent).toContain('Không còn việc tồn đọng.');
    expect(el.textContent).toContain('Mọi thứ đều ổn');
  });

  it('offers to try again when loading fails', async () => {
    http.expectOne('/admin/lessons').flush(null, { status: 500, statusText: 'Error' });
    http.match(() => true); // the other requests were cancelled
    await settle();
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Không tải được trang tổng quan.');
  });
});
