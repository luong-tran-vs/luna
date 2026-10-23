import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { TopicSummary } from '../../core/models/study';
import { advice, skillRow, TopicSummaryPage } from './topic-summary';

const summary = (over: Partial<TopicSummary> = {}): TopicSummary => ({
  topicId: 't1',
  topicName: 'Gia đình',
  level: 'A1',
  canDoVi: 'giới thiệu các thành viên trong gia đình.',
  grammarVi: ['Động từ to be', 'have got'],
  lessons: [
    { id: 'l1', title: 'My family', done: true },
    { id: 'l2', title: 'At home', done: true },
  ],
  completed: 2,
  startedAt: '2026-10-12T02:00:00Z',
  finishedAt: '2026-10-20T02:00:00Z',
  days: 3,
  words: 14,
  remembered: 9,
  skills: [
    { skill: 'listening', percent: 92, correct: 46, total: 50 },
    { skill: 'reading', percent: 50, correct: 4, total: 8 },
    { skill: 'writing', percent: 76, correct: 2, total: 0 },
  ],
  writingAverage: 3.8,
  mistakes: [{ type: 'tense', count: 3 }],
  overall: 73,
  rating: 'fair',
  ...over,
});

describe('topic summary helpers', () => {
  it('describes each skill and marks strengths and weak points', () => {
    const s = summary();
    expect(skillRow(s.skills[0], 3.8)).toEqual({
      skill: 'listening', label: 'Nghe', percent: 92, detail: 'Chép đúng 46/50 từ', mark: 'strong',
    });
    expect(skillRow(s.skills[1], 3.8).mark).toBe('weak');
    expect(skillRow(s.skills[2], 3.8)).toMatchObject({ detail: 'Điểm trung bình 3,8/5 · 2 bài', mark: '' });
  });

  it('advises on lessons left, weak skills, words not remembered and mistakes', () => {
    const items = advice(summary({ completed: 1, lessons: [...summary().lessons, { id: 'l3', title: 'x', done: false }] }));
    expect(items.map((a) => a.text)).toEqual([
      'Còn 2 bài của chủ đề chưa học.',
      'Đọc hiểu chưa chắc: đọc lại bài và làm lại câu hỏi hiểu bài.',
      'Còn 5 từ của chủ đề chưa nhớ chắc: ôn trong Sổ từ.',
      'Lỗi viết hay gặp nhất: Thì (3 lần).',
    ]);
    expect(advice(summary({ skills: [], words: 3, remembered: 3, mistakes: [] }))).toEqual([]);
  });
});

describe('TopicSummaryPage', () => {
  let fixture: ComponentFixture<TopicSummaryPage>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ id: 't1' }), queryParamMap: convertToParamMap({ level: 'A1' }) },
          },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TopicSummaryPage);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  const flush = async (s: TopicSummary) => {
    const req = http.expectOne((r) => r.url === '/topics/t1/summary');
    expect(req.request.params.get('level')).toBe('A1');
    req.flush(s);
    await fixture.whenStable();
  };

  it('shows the rating, the figures, the skills and the way on to the next topic', async () => {
    await flush(summary());
    expect(text('.topic-name')).toBe('Gia đình');
    expect(text('.done-line')).toContain('bạn đã hoàn thành chủ đề');
    expect(text('.rating-title')).toBe('Khá');
    expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('73');
    expect(text('.can-do')).toBe('Sau chủ đề này bạn có thể giới thiệu các thành viên trong gia đình.');
    expect(Array.from(el.querySelectorAll('.fact-value')).map((n) => n.textContent?.trim())).toEqual(['2/2', '3', '14']);
    expect(text('.span')).toBe('Học từ 12/10/2026 – 20/10/2026');
    expect(Array.from(el.querySelectorAll('.skill-label')).map((n) => n.textContent)).toEqual(['Nghe', 'Đọc hiểu', 'Viết']);
    expect(el.querySelectorAll('.mark-strong').length).toBe(1);
    expect(el.querySelectorAll('.mark-weak').length).toBe(1);
    expect(text('.mistakes')).toContain('Thì');
    expect(el.querySelector('.lessons a')?.getAttribute('href')).toBe('/lessons/l1');
    expect(el.querySelector('.actions a.btn-primary')?.getAttribute('href')).toBe('/goal?completed=1');
  });

  it('says when there is nothing to rate yet', async () => {
    await flush(
      summary({
        skills: [], rating: 'none', overall: 0, completed: 0, mistakes: [], words: 0, remembered: 0,
        startedAt: null, finishedAt: null, lessons: [{ id: 'l1', title: 'My family', done: false }],
      }),
    );
    expect(text('.rating-title')).toBe('Chưa có điểm');
    expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-valuetext')).toBe('—');
    expect(el.querySelector('.done-line')).toBeNull();
    expect(el.querySelector('.span')).toBeNull();
    expect(el.querySelector('.lessons a')).toBeNull();
    expect(el.querySelector('.actions a.btn-primary')?.getAttribute('href')).toBe('/lessons');
  });

  it('offers to try again when loading fails', async () => {
    http.expectOne((r) => r.url === '/topics/t1/summary').flush(null, { status: 500, statusText: 'Error' });
    await fixture.whenStable();
    expect(text('[role="alert"]')).toContain('Không tải được tổng kết chủ đề.');
  });
});
