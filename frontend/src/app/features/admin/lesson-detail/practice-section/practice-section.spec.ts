import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { Lesson } from '../../../../core/models/lesson';
import { AdminPractice } from '../../../../core/models/practice';
import { PracticeSection } from './practice-section';

const practice: AdminPractice = {
  objectiveVi: 'Bạn có thể chào hỏi.',
  examples: [{ lemma: 'meet', sentence: 'Nice to meet you.' }],
  dialogue: {
    speakers: ['Minh', 'Anna'],
    turns: [
      { speaker: 0, text: 'Hi, I am Minh.', meaningVi: 'Chào, mình là Minh.' },
      { speaker: 1, text: 'Nice to meet you.', meaningVi: 'Rất vui được gặp bạn.' },
    ],
  },
  grammarTipVi: 'Dùng "Nice to meet you" khi gặp lần đầu.',
  translations: [
    { vi: 'Rất vui được gặp bạn.', en: 'Nice to meet you.', distractors: ['see', 'glad'] },
  ],
};

const lesson = (over: Partial<Lesson> = {}): Lesson => ({
  id: 'l1',
  title: 'Park',
  level: 'A1',
  topicId: 't1',
  topicName: 'Gia đình',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '',
  content: 'We went to the park.',
  source: 's',
  license: 'l',
  revision: 1,
  annotationError: '',
  sentences: [],
  annotations: [],
  questions: [],
  grammarNote: null,
  writingPrompt: '',
  extrasEditedByAdmin: false,
  quizVersion: 0,
  practice,
  practiceStatus: 'done',
  practiceError: '',
  ...over,
});

describe('PracticeSection', () => {
  let fixture: ComponentFixture<PracticeSection>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let emitted: Lesson[];

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const regenerate = () =>
    Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Tạo lại phần luyện tập')!;
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  const render = async (l: Lesson) => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(PracticeSection);
    fixture.componentRef.setInput('lesson', l);
    emitted = [];
    fixture.componentInstance.regenerated.subscribe((x) => emitted.push(x));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('shows the status and the whole generated content', async () => {
    await render(lesson());
    expect(text(el.querySelector('lu-status-chip'))).toContain('Luyện tập: Xong');
    const content = text(el.querySelector('.content'));
    for (const part of [
      'Bạn có thể chào hỏi.',
      'meet: Nice to meet you.',
      'Minh: Hi, I am Minh.',
      'Chào, mình là Minh.',
      'Anna: Nice to meet you.',
      'Dùng "Nice to meet you" khi gặp lần đầu.',
      'Rất vui được gặp bạn. → Nice to meet you.',
      'Từ gây nhiễu: see, glad',
    ]) {
      expect(content).toContain(part);
    }
    expect(regenerate().disabled).toBe(false);
  });

  it('shows "Chưa có" without content', async () => {
    await render(lesson({ practice: null, practiceStatus: 'none' }));
    expect(text(el.querySelector('.none-chip'))).toBe('Luyện tập: Chưa có');
    expect(el.querySelector('lu-status-chip')).toBeNull();
    expect(text(el.querySelector('p.card'))).toBe('Chưa có nội dung luyện tập.');
  });

  it('shows the failure reason', async () => {
    await render(lesson({ practiceStatus: 'failed', practiceError: 'AI chưa được cấu hình' }));
    expect(text(el.querySelector('lu-status-chip'))).toContain('Luyện tập: Lỗi');
    expect(text(el.querySelector('.failure'))).toBe('Lý do: AI chưa được cấu hình');
  });

  it('disables regenerating while it runs, with the reason', async () => {
    await render(lesson({ practiceStatus: 'running' }));
    expect(regenerate().disabled).toBe(true);
    expect(regenerate().getAttribute('aria-describedby')).toBe('practice-blocked');
    expect(text(el.querySelector('#practice-blocked'))).toBe('Đang sinh…');
  });

  it('disables regenerating until annotation is done, with the reason', async () => {
    await render(lesson({ annotationStatus: 'failed', practice: null, practiceStatus: 'none' }));
    expect(regenerate().disabled).toBe(true);
    expect(text(el.querySelector('#practice-blocked'))).toBe('Cần chú thích xong trước');
  });

  it('regenerates and emits the returned lesson', async () => {
    await render(lesson());
    regenerate().click();
    await settle();
    const req = http.expectOne('/api/admin/lessons/l1/practice/regenerate');
    expect(req.request.method).toBe('POST');
    req.flush(
      { lesson: lesson({ practiceStatus: 'running' }) },
      { status: 202, statusText: 'Accepted' },
    );
    await settle();
    expect(emitted.map((l) => l.practiceStatus)).toEqual(['running']);
  });

  it('shows the API message when regenerating fails', async () => {
    await render(lesson());
    regenerate().click();
    await settle();
    http
      .expectOne('/api/admin/lessons/l1/practice/regenerate')
      .flush(
        { error: 'practice_running', message: 'Đang sinh phần luyện tập.' },
        { status: 409, statusText: 'Conflict' },
      );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Đang sinh phần luyện tập.');
    expect(emitted).toEqual([]);
  });
});
