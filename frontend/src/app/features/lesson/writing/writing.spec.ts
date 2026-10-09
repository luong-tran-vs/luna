import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { LessonWriting, Writing as WritingData } from '../../../core/models/writing';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { Writing } from './writing';

const words = (n: number) => Array.from({ length: n }, (_, i) => `w${i}`).join(' ');

const writing = (over: Partial<WritingData> = {}): WritingData => ({
  id: 'w1',
  lessonId: 'l1',
  lessonTitle: 'My family',
  prompt: 'Write about your family.',
  text: 'My family has four people.',
  status: 'submitted',
  submittedAt: '2026-10-01T03:00:00Z',
  grade: { status: 'pending', error: '', criteria: [], average: null, overallVi: '', correctedText: '', gradedAt: null },
  ...over,
});

const view = (over: Partial<LessonWriting> = {}): LessonWriting => ({
  prompt: 'Write about your family.',
  level: 'A1',
  canWrite: true,
  writing: null,
  ...over,
});

describe('Writing', () => {
  let fixture: ComponentFixture<Writing>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let completed: number;
  const notifier = { submitted: vi.fn() };

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const textarea = () => el.querySelector<HTMLTextAreaElement>('#writing-text');
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
  // Renders without whenStable(), which waits for pending requests and real timers.
  const render = async () => {
    await vi.advanceTimersByTimeAsync(0);
    fixture.detectChanges();
  };
  const type = async (value: string) => {
    textarea()!.value = value;
    textarea()!.dispatchEvent(new Event('input'));
    await vi.advanceTimersByTimeAsync(0);
  };

  const setup = async (v: LessonWriting, mode: 'study' | 'review' | null = 'study') => {
    vi.useFakeTimers();
    notifier.submitted.mockClear();
    await TestBed.configureTestingModule({
      imports: [Writing],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: WritingNotifier, useValue: notifier },
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({}), queryParamMap: convertToParamMap({}) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Writing);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('mode', mode);
    completed = 0;
    fixture.componentInstance.completed.subscribe(() => completed++);
    el = fixture.nativeElement;
    fixture.detectChanges();
    http.expectOne('/lessons/l1/writing').flush(v);
    await vi.advanceTimersByTimeAsync(0);
    await fixture.whenStable();
  };

  afterEach(() => {
    http.verify();
    vi.useRealTimers();
  });

  it('offers a way back to the lessons only when opened on its own', async () => {
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }), null);
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')?.getAttribute('href')).toBe('/lessons');
    expect(fixture.nativeElement.classList).toContain('standalone');
    fixture.destroy();
    TestBed.resetTestingModule();
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }));
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')).toBeNull();
  });

  it('shows the prompt, the suggested length and the word count', async () => {
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }));
    expect(text(el.querySelector('.prompt-text'))).toBe('Write about your family.');
    expect(text(el.querySelector('.step-card-head'))).toContain('Viết 30–60 từ');
    expect(textarea()!.value).toBe('My family');
    expect(text(el.querySelector('#writing-count'))).toBe('2/60 từ · Bài viết cần ít nhất 5 từ.');
    expect(button('Nộp')!.disabled).toBe(true);
  });

  it('saves the draft one second after the last keystroke', async () => {
    await setup(view());
    await type('My');
    await vi.advanceTimersByTimeAsync(500);
    await type('My family');
    await vi.advanceTimersByTimeAsync(999);
    http.expectNone('/lessons/l1/writing');
    await vi.advanceTimersByTimeAsync(1);
    const req = http.expectOne('/lessons/l1/writing');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ text: 'My family' });
    await render();
    expect(text(el.querySelector('#writing-status'))).toBe('Đang lưu…');
    req.flush({ writing: writing({ status: 'draft', text: 'My family', grade: null, submittedAt: null }) });
    await render();
    expect(text(el.querySelector('#writing-status'))).toBe('Đã lưu nháp');
  });

  it('says when the draft could not be saved', async () => {
    await setup(view());
    await type('My family');
    await vi.advanceTimersByTimeAsync(1000);
    http.expectOne('/lessons/l1/writing').error(new ProgressEvent('error'));
    await render();
    expect(text(el.querySelector('#writing-status'))).toBe('Chưa lưu được nháp');
  });

  it('allows submitting only between 5 and 400 words', async () => {
    await setup(view());
    await type(words(4));
    await fixture.whenStable();
    expect(button('Nộp')!.disabled).toBe(true);
    await type(words(401));
    await fixture.whenStable();
    expect(button('Nộp')!.disabled).toBe(true);
    expect(text(el.querySelector('#writing-count'))).toContain('Bài viết tối đa 400 từ.');
    await type(words(5));
    await fixture.whenStable();
    expect(button('Nộp')!.disabled).toBe(false);
  });

  it('submits, then completes the step and watches for the result', async () => {
    await setup(view());
    await type('My family has four people.');
    await fixture.whenStable();
    button('Nộp')!.click();
    await vi.advanceTimersByTimeAsync(0);
    const req = http.expectOne('/lessons/l1/writing/submit');
    expect(req.request.body).toEqual({ text: 'My family has four people.' });
    req.flush({ writing: writing() });
    await vi.advanceTimersByTimeAsync(0);
    await fixture.whenStable();
    expect(completed).toBe(1);
    expect(notifier.submitted).toHaveBeenCalledTimes(1);
    expect(text(el.querySelector('.result'))).toContain('Đã nộp, AI đang chấm.');
    expect(el.querySelector('.result a')?.getAttribute('href')).toBe('/writings/w1');
    expect(textarea()).toBeNull();
    expect(button('Tiếp tục')).toBeUndefined();
    // The pending draft save was cancelled.
    await vi.advanceTimersByTimeAsync(2000);
  });

  it('shows the server message when submitting fails', async () => {
    await setup(view());
    await type(words(10));
    await fixture.whenStable();
    button('Nộp')!.click();
    await vi.advanceTimersByTimeAsync(0);
    http
      .expectOne('/lessons/l1/writing/submit')
      .flush({ error: 'write_locked', message: 'Hãy học tới bước Viết của bài đang học' }, { status: 409, statusText: 'Conflict' });
    await vi.advanceTimersByTimeAsync(0);
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Hãy học tới bước Viết của bài đang học');
    expect(completed).toBe(0);
  });

  it('offers Tiếp tục for a writing submitted before this visit', async () => {
    await setup(view({ canWrite: false, writing: writing() }));
    expect(textarea()).toBeNull();
    expect(text(el.querySelector('.text'))).toBe('My family has four people.');
    button('Tiếp tục')!.click();
    expect(completed).toBe(1);
  });

  it('lets the learner skip writing in the daily flow, without sending anything', async () => {
    await setup(view({ writing: null }));
    let skipped = 0;
    fixture.componentInstance.skipped.subscribe(() => skipped++);
    expect(text(el.querySelector('.skip-hint'))).toContain('Viết là tuỳ chọn');
    button('Bỏ qua')!.click();
    expect(skipped).toBe(1);
    expect(completed).toBe(0);
  });

  it('has no skip outside the daily flow', async () => {
    await setup(view({ writing: null }), null);
    expect(button('Bỏ qua')).toBeUndefined();
    expect(el.querySelector('.skip-hint')).toBeNull();
  });

  it('shows the result summary of a graded writing in review mode', async () => {
    const graded = writing({
      grade: { status: 'done', error: '', criteria: [], average: 3.75, overallVi: 'Tốt', correctedText: 'x', gradedAt: '' },
    });
    await setup(view({ canWrite: false, writing: graded }), 'review');
    expect(text(el.querySelector('.result'))).toContain('Điểm trung bình 3,8/5.');
    expect(button('Tiếp tục')).toBeUndefined();
    expect(textarea()).toBeNull();
  });

  it('says when a past lesson has no writing', async () => {
    await setup(view({ canWrite: false }), 'review');
    expect(text(el.querySelector('.writing'))).toContain('Bài này chưa có bài viết.');
    expect(textarea()).toBeNull();
  });
});
