import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Grade, UnseenCount, Writing } from '../../../core/models/writing';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { WritingDetail } from './writing-detail';

const done: Grade = {
  status: 'done',
  error: '',
  criteria: [
    { name: 'task', score: 4, commentVi: 'Đúng đề.' },
    { name: 'grammar', score: 3, commentVi: 'Sai thì.' },
    { name: 'vocabulary', score: 4, commentVi: 'Đủ từ.' },
    { name: 'coherence', score: 4, commentVi: 'Mạch lạc.' },
  ],
  average: 3.8,
  overallVi: 'Khá tốt.',
  correctedText: 'My family has four members.',
  gradedAt: '2026-10-01T03:01:00Z',
};

const writing = (grade: Grade | null, used = 1): Writing => ({
  id: 'w1',
  lessonId: 'l1',
  lessonTitle: 'My family',
  prompt: 'Write about your family.',
  text: 'My family have four people.',
  status: 'submitted',
  submittedAt: '2026-10-01T03:00:00Z',
  grade,
  gradings: { used, max: 2 },
});

const pending: Grade = { ...done, status: 'pending', criteria: [], average: null, overallVi: '', correctedText: '' };

describe('WritingDetail', () => {
  let fixture: ComponentFixture<WritingDetail>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const latest = signal<UnseenCount['latest']>(null);
  const notifier = { latest, markSeen: vi.fn(), submitted: vi.fn() };

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);

  const setup = async (response: Writing | 'missing') => {
    latest.set(null);
    notifier.markSeen.mockClear();
    notifier.submitted.mockClear();
    await TestBed.configureTestingModule({
      imports: [WritingDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: WritingNotifier, useValue: notifier },
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 'w1' }) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(WritingDetail);
    el = fixture.nativeElement;
    const req = http.expectOne('/api/writings/w1');
    if (response === 'missing') {
      req.flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
    } else {
      req.flush({ writing: response });
    }
    await settle();
  };

  afterEach(() => http.verify());

  it('shows a graded writing with the four criteria, the correction and the diff', async () => {
    await setup(writing(done));
    expect(text(el.querySelector('.title'))).toBe('My family');
    expect(text(el.querySelector('.average'))).toBe('Điểm trung bình 3,8/5');
    const criteria = Array.from(el.querySelectorAll('.criterion')).map((c) => [
      text(c.querySelector('.criterion-head')),
      text(c.querySelector('.criterion-head + p')),
    ]);
    expect(criteria).toEqual([
      ['Hoàn thành yêu cầu: 4/5 điểm', 'Đúng đề.'],
      ['Ngữ pháp: 3/5 điểm', 'Sai thì.'],
      ['Từ vựng: 4/5 điểm', 'Đủ từ.'],
      ['Mạch lạc: 4/5 điểm', 'Mạch lạc.'],
    ]);
    expect(text(el)).toContain('Khá tốt.');
    expect(text(el)).toContain('My family has four members.');

    const ins = Array.from(el.querySelectorAll('.diff ins')).map((i) => text(i));
    const del = Array.from(el.querySelectorAll('.diff del')).map((d) => text(d));
    expect(ins).toEqual(['thêm: has', 'thêm: members.']);
    expect(del).toEqual(['bớt: have', 'bớt: people.']);
    expect(el.querySelector('.diff ins .visually-hidden')).not.toBeNull();
    expect(text(el.querySelector('.legend'))).toBe('Gạch chân: thêm · Gạch ngang: bớt');
    expect(notifier.markSeen).toHaveBeenCalledWith('w1');
  });

  it('shows the grading in progress and reloads when the result arrives', async () => {
    await setup(writing(pending));
    expect(text(el.querySelector('.status'))).toContain('Đang chấm');
    expect(el.querySelector('.criteria')).toBeNull();
    expect(notifier.markSeen).not.toHaveBeenCalled();

    latest.set({ id: 'w1', status: 'done' });
    await settle();
    http.expectOne('/api/writings/w1').flush({ writing: writing(done) });
    await settle();
    expect(el.querySelectorAll('.criterion').length).toBe(4);
  });

  it('offers Chấm lại for a failed grading', async () => {
    await setup(writing({ ...pending, status: 'failed', error: 'AI hết lượt, vui lòng chấm lại sau' }));
    expect(text(el.querySelector('.failed p'))).toBe('Chấm lỗi: AI hết lượt, vui lòng chấm lại sau');
    expect(notifier.markSeen).toHaveBeenCalledWith('w1');
    button('Chấm lại (còn 1 lượt)')!.click();
    await settle();
    const req = http.expectOne('/api/writings/w1/regrade');
    expect(req.request.method).toBe('POST');
    req.flush({ writing: writing(pending) });
    await settle();
    expect(text(el.querySelector('.status'))).toContain('Đang chấm');
    expect(notifier.submitted).toHaveBeenCalled();
  });

  it('shows the message when Chấm lại is refused', async () => {
    await setup(writing({ ...pending, status: 'failed', error: 'x' }));
    button('Chấm lại (còn 1 lượt)')!.click();
    await settle();
    http
      .expectOne('/api/writings/w1/regrade')
      .flush({ error: 'not_failed', message: 'Chỉ chấm lại được bài chấm lỗi' }, { status: 409, statusText: 'Conflict' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Chỉ chấm lại được bài chấm lỗi');
  });

  it('hides Chấm lại once both gradings are used', async () => {
    await setup(writing({ ...pending, status: 'failed', error: 'x' }, 2));
    expect(button('Chấm lại (còn 1 lượt)')).toBeUndefined();
    expect(text(el.querySelector('.failed'))).toContain('Bài viết này đã dùng hết lượt chấm.');
    expect(text(el.querySelector('.gradings'))).toBe('Lượt chấm: 2/2');
  });

  it('lets the learner edit and resubmit a graded writing once', async () => {
    await setup(writing(done));
    expect(text(el.querySelector('.gradings'))).toBe('Lượt chấm: 1/2');
    button('Sửa bài')!.click();
    await settle();
    const area = el.querySelector<HTMLTextAreaElement>('#resubmit-text')!;
    expect(area.value).toBe('My family have four people.');

    area.value = 'Too short';
    area.dispatchEvent(new Event('input'));
    await settle();
    expect(text(el.querySelector('#resubmit-count'))).toBe('2 từ (từ 5 đến 400)');
    button('Nộp lại để chấm')!.click();
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Bài viết cần từ 5 đến 400 từ.');

    area.value = 'My family has four people.';
    area.dispatchEvent(new Event('input'));
    button('Nộp lại để chấm')!.click();
    await settle();
    const req = http.expectOne('/api/writings/w1/resubmit');
    expect(req.request.body).toEqual({ text: 'My family has four people.' });
    req.flush({ writing: { ...writing(pending, 2), text: 'My family has four people.' } });
    await settle();
    expect(text(el.querySelector('.status'))).toContain('Đang chấm');
    expect(el.querySelector('.resubmit')).toBeNull();
    expect(notifier.submitted).toHaveBeenCalled();
  });

  it('offers no resubmit once both gradings are used', async () => {
    await setup(writing(done, 2));
    expect(button('Sửa bài')).toBeUndefined();
    expect(text(el)).toContain('Bài viết này đã dùng hết 2 lượt chấm.');
  });

  it('says when the writing does not exist', async () => {
    await setup('missing');
    expect(text(el)).toContain('Không tìm thấy bài viết.');
  });
});
