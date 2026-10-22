import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { WritingHelp } from '../../../core/models/practice';
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
  const notifier = { submitted: vi.fn(), latest: signal<{ id: string; status: 'pending' | 'done' | 'failed' } | null>(null) };

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

  const setup = async (
    v: LessonWriting,
    mode: 'study' | 'review' | null = 'study',
    extra: { help?: WritingHelp | null; targetWords?: string[] } = {},
  ) => {
    vi.useFakeTimers();
    notifier.submitted.mockClear();
    notifier.latest.set(null);
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
    fixture.componentRef.setInput('help', extra.help ?? null);
    fixture.componentRef.setInput('targetWords', extra.targetWords ?? []);
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

  it("offers a way back to the lesson's part only when opened on its own", async () => {
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }), null);
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')?.getAttribute('href')).toBe('/lessons/levels/A1');
    expect(fixture.nativeElement.classList).toContain('standalone');
    fixture.destroy();
    TestBed.resetTestingModule();
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }));
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')).toBeNull();
  });

  it('shows the prompt, the suggested length and the word count', async () => {
    await setup(view({ writing: writing({ status: 'draft', text: 'My family', submittedAt: null, grade: null }) }));
    expect(text(el.querySelector('.prompt-text'))).toBe('Write about your family.');
    expect(text(el.querySelector('.step-card-head'))).toContain('Nên viết 30–60 từ');
    expect(textarea()!.value).toBe('My family');
    expect(text(el.querySelector('#writing-count'))).toBe('2/60 từ · Bài viết cần ít nhất 5 từ.');
    expect(button('Nộp')!.disabled).toBe(true);
  });

  it('says a text shorter than suggested can still be submitted', async () => {
    await setup(
      view({
        writing: writing({
          status: 'draft',
          text: 'My family is big and happy.',
          submittedAt: null,
          grade: null,
        }),
      }),
    );
    expect(text(el.querySelector('#writing-count'))).toBe(
      '6/60 từ · Còn ngắn so với gợi ý 30–60 từ, vẫn nộp được.',
    );
    expect(button('Nộp')!.disabled).toBe(false);
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
    expect(text(el.querySelector('.result'))).toBe('Điểm trung bình 3,8/5');
    expect(button('Tiếp tục')).toBeUndefined();
    expect(textarea()).toBeNull();
  });

  it('marks the corrections in the text and shows each criterion', async () => {
    const graded = writing({
      text: 'I eats rice at six oclock.',
      grade: {
        status: 'done',
        error: '',
        criteria: [
          { name: 'task', score: 5, commentVi: '' },
          { name: 'grammar', score: 2, commentVi: '' },
        ],
        average: 3.5,
        overallVi: 'Với chủ ngữ I, động từ giữ nguyên.',
        correctedText: "I eat rice at six o'clock.",
        gradedAt: '',
      },
    });
    await setup(view({ canWrite: false, writing: graded }), 'review');
    expect(text(el.querySelector('.submitted .prompt-label'))).toBe('Bài viết của bạn · 2 chỗ cần sửa');
    expect(Array.from(el.querySelectorAll('.diff del')).map((d) => text(d))).toEqual(['bỏ: eats', 'bỏ: oclock.']);
    expect(Array.from(el.querySelectorAll('.diff ins')).map((d) => text(d))).toEqual(['sửa thành: eat', "sửa thành: o'clock."]);
    expect(text(el.querySelector('.criteria .low'))).toBe('Ngữ pháp2/5');
    expect(text(el.querySelector('.tip'))).toBe('Với chủ ngữ I, động từ giữ nguyên.');
    expect(el.querySelector('.detail-link')?.getAttribute('href')).toBe('/writings/w1');
  });

  it('shows the result once the grading finishes', async () => {
    await setup(view({ canWrite: false, writing: writing() }), 'review');
    expect(text(el.querySelector('.result'))).toContain('AI đang chấm');
    notifier.latest.set({ id: 'w1', status: 'done' });
    await render();
    const done = { status: 'done' as const, error: '', criteria: [], average: 4, overallVi: '', correctedText: 'My family has four people.', gradedAt: '' };
    http.expectOne('/lessons/l1/writing').flush(view({ canWrite: false, writing: writing({ grade: done }) }));
    await render();
    expect(text(el.querySelector('.submitted .prompt-label'))).toBe('Bài viết của bạn · không có lỗi cần sửa');
  });

  describe('with the lesson words and the writing help', () => {
    const help: WritingHelp = {
      grammarLabelVi: '1 câu nói về người khác (he / she …)',
      anyOf: ['he', 'she'],
      outlineVi: ['Câu 1: nhạc bạn thích.', 'Câu 2: một người thích nhạc.', 'Câu 3: cảm nhận.'],
      modelAnswer: 'I like pop music. My sister plays loud songs and she sings.',
    };
    const targetWords = ['sounds', 'sound', 'loud', 'dance', 'turn on'];
    const item = (cls: string) => el.querySelector(`.needs-list .${cls}`);

    it('ticks what the text has while typing, without blocking Nộp', async () => {
      await setup(view(), 'study', { help, targetWords });
      expect(text(el.querySelector('.needs'))).toContain('Cần có');
      expect(text(item('need-words'))).toBe('☐Chưa đạt: 3 từ của bài: sound · loud · dance');
      expect(text(item('need-grammar'))).toBe('☐Chưa đạt: 1 câu nói về người khác (he / she …)');
      expect(text(item('need-range'))).toBe('☐Chưa đạt: 30–60 từ (đang có 0)');

      await type('The sounds are LOUD and my mother likes them.');
      fixture.detectChanges();
      const used = Array.from(el.querySelectorAll('.need-word.used')).map((w) => text(w));
      expect(used).toEqual(['sound (đã dùng)', 'loud (đã dùng)']);
      expect(item('need-words')!.classList).not.toContain('done');
      expect(item('need-grammar')!.classList).not.toContain('done');
      expect(button('Nộp')!.disabled).toBe(false);

      await type('She dances to the loud sound. ' + words(25));
      fixture.detectChanges();
      expect(item('need-words')!.classList).toContain('done');
      expect(item('need-grammar')!.classList).toContain('done');
      expect(item('need-range')!.classList).toContain('done');
      expect(text(item('need-range'))).toBe('☑Đã đạt: 30–60 từ (đang có 31)');
      await vi.advanceTimersByTimeAsync(1000);
      http.expectOne('/lessons/l1/writing').flush({ writing: writing({ status: 'draft', grade: null, submittedAt: null }) });
    });

    it('underlines the lesson words behind the text box', async () => {
      await setup(view(), 'study', { help, targetWords });
      await type('Turn on the loud music');
      fixture.detectChanges();
      const backdrop = el.querySelector('.backdrop')!;
      expect(backdrop.getAttribute('aria-hidden')).toBe('true');
      expect(Array.from(backdrop.querySelectorAll('mark')).map((m) => text(m))).toEqual(['Turn on', 'loud']);
      expect(textarea()!.classList).toContain('over-backdrop');
      await vi.advanceTimersByTimeAsync(1000);
      http.expectOne('/lessons/l1/writing').flush({ writing: writing({ status: 'draft', grade: null, submittedAt: null }) });
    });

    it('shows the 3-sentence outline on demand', async () => {
      await setup(view(), 'study', { help, targetWords });
      expect(el.querySelector('#writing-outline')).toBeNull();
      const show = button('Xem dàn ý 3 câu')!;
      expect(show.getAttribute('aria-expanded')).toBe('false');
      show.click();
      fixture.detectChanges();
      expect(Array.from(el.querySelectorAll('#writing-outline li')).map((li) => text(li))).toEqual(help.outlineVi);
      expect(button('Ẩn dàn ý')!.getAttribute('aria-expanded')).toBe('true');
    });

    it('keeps the word range only, without help or lesson words', async () => {
      await setup(view());
      expect(item('need-words')).toBeNull();
      expect(item('need-grammar')).toBeNull();
      expect(item('need-range')).not.toBeNull();
      expect(el.querySelector('.backdrop')).toBeNull();
      expect(el.querySelector('.outline')).toBeNull();
    });

    it('numbers the tagged mistakes, then shows the model answer and the frequent mistakes', async () => {
      const graded = writing({
        text: 'My mother like old songs.',
        grade: {
          status: 'done',
          error: '',
          criteria: [],
          average: 3.5,
          overallVi: 'Bạn dùng được loud.',
          correctedText: 'My mother likes old songs.',
          gradedAt: '',
          mistakes: [{ original: 'like', corrected: 'likes', type: 'verb-s', noteVi: 'my mother = she → likes' }],
        },
      });
      await setup(view({ canWrite: false, writing: graded }), 'review', { help, targetWords });
      expect(text(el.querySelector('.diff sup'))).toBe('(lỗi 1)');
      expect(text(el.querySelector('.mistakes li'))).toBe('Động từ thêm -s, lỗi 1 my mother = she → likes');
      expect(text(el.querySelector('.tip'))).toBe('Bạn dùng được loud.');
      expect(el.querySelector('.model')).toBeNull();

      button('Xem bài mẫu')!.click();
      fixture.detectChanges();
      http.expectOne('/writings/mistakes').flush({
        mistakes: [
          { type: 'verb-s', count: 4, examples: [{ original: 'she like', corrected: 'she likes', type: 'verb-s', noteVi: '' }] },
          {
            type: 'preposition',
            count: 2,
            examples: [{ original: 'listen music', corrected: 'listen to', type: 'preposition', noteVi: '' }],
          },
        ],
      });
      await render();
      expect(text(el.querySelector('#model-heading'))).toBe('Bài mẫu A1');
      expect(Array.from(el.querySelectorAll('.model-text .lesson-word')).map((w) => text(w))).toEqual(['loud']);
      expect(Array.from(el.querySelectorAll('.model-text .grammar-word')).map((w) => text(w))).toEqual(['she']);
      expect(text(el.querySelector('.model .legend'))).toBe('Gạch chấm: từ của bài · Tô vàng: ngữ pháp của bài');
      const rows = Array.from(el.querySelectorAll('.frequent li')).map((li) => text(li));
      expect(rows).toEqual(['Động từ thêm -s: she like → she likes, 4 lần', 'Giới từ: listen music → listen to, 2 lần']);
    });

    it('hides the frequent mistakes when they fail to load, and keeps older grades unnumbered', async () => {
      const graded = writing({
        grade: { status: 'done', error: '', criteria: [], average: 4, overallVi: '', correctedText: 'My family has four people.', gradedAt: '' },
      });
      await setup(view({ canWrite: false, writing: graded }), 'review', { help, targetWords });
      expect(el.querySelector('.mistakes')).toBeNull();
      button('Xem bài mẫu')!.click();
      fixture.detectChanges();
      http.expectOne('/writings/mistakes').error(new ProgressEvent('error'));
      await render();
      expect(el.querySelector('.model')).not.toBeNull();
      expect(el.querySelector('.frequent')).toBeNull();
    });
  });

  it('says when a past lesson has no writing', async () => {
    await setup(view({ canWrite: false }), 'review');
    expect(text(el.querySelector('.writing'))).toContain('Bài này chưa có bài viết.');
    expect(textarea()).toBeNull();
  });
});
