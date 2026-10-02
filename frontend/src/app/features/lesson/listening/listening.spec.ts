import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { DictationResult, DictationSummary } from '../../../core/models/dictation';
import { ReadingLesson } from '../../../core/models/reading';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { Listening } from './listening';

const lesson: ReadingLesson = {
  id: 'l1',
  quiz: null,
  grammarNote: null,
  title: 'Apples',
  level: 'A2',
  topic: '',
  sentences: [
    { index: 0, text: "I don't like green apples." },
    { index: 1, text: 'We went to the park at 9.30.' },
    { index: 2, text: 'Why?' },
  ],
  paragraphs: [[0, 1, 2]],
  lemmas: {},
  phrases: [],
};


function summaryOf(results: DictationResult[], sentenceCount = 3): DictationSummary {
  const correctWords = results.reduce((n, r) => n + r.correctWords, 0);
  const totalWords = results.reduce((n, r) => n + r.totalWords, 0);
  return {
    sentenceCount,
    checkedCount: results.length,
    correctWords,
    totalWords,
    rate: totalWords ? correctWords / totalWords : 0,
    completed: results.length === sentenceCount,
    results,
  };
}

const result = (sentenceIndex: number, typed: string, correctWords: number, totalWords: number): DictationResult => ({
  sentenceIndex,
  typed,
  correctWords,
  totalWords,
  checkedAt: '2026-09-29T08:00:00Z',
});

describe('Listening', () => {
  let fixture: ComponentFixture<Listening>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let speech: FakeSpeech;
  let completed: number;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const button = (text: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === text) as HTMLButtonElement | undefined;
  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const filled = () => Number(el.querySelector('lu-waveform clipPath rect')!.getAttribute('width'));
  const input = () => el.querySelector<HTMLInputElement>('#answer')!;
  const type = async (value: string) => {
    input().value = value;
    input().dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const check = async (value: string) => {
    await type(value);
    button('Kiểm tra')!.click();
    await settle();
  };
  const expectPost = (): TestRequest => {
    const req = http.expectOne((r) => r.url === '/api/lessons/l1/dictation' && r.method === 'POST');
    return req;
  };

  const setup = async (
    options: {
      lesson?: ReadingLesson | 404 | 403;
      summary?: DictationSummary;
      inputs?: Record<string, unknown>;
      query?: Record<string, string>;
      speech?: boolean;
    } = {},
  ) => {
    completed = 0;
    speech = new FakeSpeech();
    speech.supported = options.speech ?? true;
    await TestBed.configureTestingModule({
      imports: [Listening],
      providers: [
        provideRouter([{ path: 'forbidden', children: [] }]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ id: 'l1' }), queryParamMap: convertToParamMap(options.query ?? {}) },
          },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Listening);
    for (const [name, value] of Object.entries(options.inputs ?? {})) {
      fixture.componentRef.setInput(name, value);
    }
    fixture.componentInstance.completed.subscribe(() => completed++);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const id = (options.inputs?.['lessonId'] as string | undefined) ?? 'l1';
    const review = options.inputs?.['mode'] === 'review' || options.query?.['review'] === '1';
    const lessonReq = http.expectOne(`/api/lessons/${id}`);
    const summaryReq = review ? null : http.expectOne(`/api/lessons/${id}/dictation/summary`);
    if (options.lesson === 404 || options.lesson === 403) {
      lessonReq.flush({ error: 'x', message: 'x' }, { status: options.lesson, statusText: 'Error' });
    } else {
      lessonReq.flush({ lesson: options.lesson ?? lesson });
      summaryReq?.flush({ summary: options.summary ?? summaryOf([]) });
    }
    await settle();
  };

  beforeEach(() => {
    // jsdom has no layout: scrollIntoView does not exist.
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    http?.verify();
    vi.restoreAllMocks();
  });

  it('offers a way back to the lessons only when opened on its own', async () => {
    await setup();
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')?.getAttribute('href')).toBe('/lessons');
    fixture.destroy();
    TestBed.resetTestingModule();
    await setup({ inputs: { lessonId: 'l1', mode: 'study' } });
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')).toBeNull();
  });

  describe('listening sentence by sentence', () => {
    it('shows the first sentence with previous disabled', async () => {
      await setup();
      expect(text('.counter')).toBe('Câu 1/3');
      expect(button('Câu trước')!.disabled).toBe(true);
      expect(button('Câu sau')!.disabled).toBe(false);
    });

    it('has the browser read the hidden sentence, and read it again from the start', async () => {
      await setup();
      expect(speech.spoken).toHaveLength(0);
      button('Nghe câu')!.click();
      await settle();
      expect(speech.last()).toMatchObject({ text: "I don't like green apples.", rate: 1 });
      speech.stops = 0;
      button('Nghe câu')!.click();
      await settle();
      expect(speech.stops).toBeGreaterThan(0);
      expect(speech.spoken).toHaveLength(2);
    });

    it('fills the waveform while reading and when done', async () => {
      await setup();
      expect(el.querySelector('lu-waveform svg')!.getAttribute('aria-hidden')).toBe('true');
      expect(filled()).toBe(0);
      button('Nghe câu')!.click();
      await settle();
      speech.last().handlers.progress!(0.5, 0.6);
      await fixture.whenStable();
      expect(filled()).toBe(96);
      speech.last().handlers.progress!(1, 1.2);
      speech.last().handlers.ended!();
      await fixture.whenStable();
      expect(filled()).toBe(192); // 64 bars × 3 units
    });

    it('says so when the browser fails to read', async () => {
      await setup();
      button('Nghe câu')!.click();
      await settle();
      speech.last().handlers.failed!('synthesis-failed');
      await fixture.whenStable();
      expect(text('[role="alert"]')).toBe('Chưa đọc được câu. Hãy thử lại.');
      expect(filled()).toBe(0);
    });

    it('moves between sentences and plays the new one', async () => {
      await setup();
      button('Câu sau')!.click();
      button('Câu sau')!.click();
      await settle();
      expect(text('.counter')).toBe('Câu 3/3');
      expect(button('Câu sau')!.disabled).toBe(true);
      expect(speech.last().text).toBe('Why?');
      button('Câu trước')!.click();
      await settle();
      expect(text('.counter')).toBe('Câu 2/3');
    });

    it('keeps the chosen speed across sentences', async () => {
      await setup();
      const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
      expect(radios.map((r) => r.textContent?.trim())).toEqual(['0.5x', '0.75x', '1x', '1.25x']);
      expect(radios[2].getAttribute('aria-checked')).toBe('true');
      radios[1].click();
      await settle();
      expect(radios[1].getAttribute('aria-checked')).toBe('true');
      button('Câu sau')!.click();
      button('Câu sau')!.click();
      await settle();
      expect(speech.last().rate).toBe(0.75);
    });

    it('changes speed with arrow keys', async () => {
      await setup();
      const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
      radios[2].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await settle();
      expect(radios[3].getAttribute('aria-checked')).toBe('true');
      expect(radios[3].tabIndex).toBe(0);
      expect(radios[2].tabIndex).toBe(-1);
    });

    it('hides the transcript until asked', async () => {
      await setup();
      expect(el.querySelector('.transcript')).toBeNull();
      button('Hiện chữ')!.click();
      await settle();
      expect(text('.transcript')).toBe("I don't like green apples.");
      button('Ẩn chữ')!.click();
      await settle();
      expect(el.querySelector('.transcript')).toBeNull();
    });

    it('reads every sentence', async () => {
      await setup();
      button('Câu sau')!.click();
      await settle();
      expect(speech.last().text).toBe('We went to the park at 9.30.');
      expect(button('Nghe câu')).toBeTruthy();
    });

    it('shows a message and no dictation when the browser has no voice', async () => {
      await setup({ speech: false });
      expect(el.textContent).toContain('Trình duyệt này không có giọng đọc');
      expect(el.querySelector('#answer')).toBeNull();
    });

    it('stops reading when the page closes', async () => {
      await setup();
      speech.stops = 0;
      fixture.destroy();
      expect(speech.stops).toBeGreaterThan(0);
    });

    it('shows not found for an unknown lesson', async () => {
      await setup({ lesson: 404 });
      expect(el.textContent).toContain('Không tìm thấy bài học');
    });
  });

  describe('dictation', () => {
    it('configures the input so the keyboard does not correct words', async () => {
      await setup();
      const i = input();
      expect(i.getAttribute('autocapitalize')).toBe('off');
      expect(i.getAttribute('autocorrect')).toBe('off');
      expect(i.getAttribute('autocomplete')).toBe('off');
      expect(i.getAttribute('spellcheck')).toBe('false');
      expect(i.getAttribute('enterkeyhint')).toBe('done');
    });

    it('asks for text on an empty check and records nothing', async () => {
      await setup();
      await check('   ');
      expect(text('.answer-error')).toBe('Hãy gõ câu bạn nghe được');
      expect(el.querySelector('.result')).toBeNull();
    });

    it('checks on Enter (form submit) and shows each word state', async () => {
      await setup();
      await type('so i dont like apples');
      el.querySelector('form')!.dispatchEvent(new Event('submit'));
      await settle();
      expectPost().flush({ summary: summaryOf([result(0, 'so i dont like apples', 3, 6)]) });
      await settle();

      const words = Array.from(el.querySelectorAll<HTMLElement>('.result .word'));
      expect(words.map((w) => w.className.replace('word', '').trim())).toEqual(['extra', 'ok', 'wrong', 'ok', 'missing', 'ok']);
      expect(words[2].textContent?.replace(/\s+/g, ' ')).toContain("dont → don't");
      expect(words[2].querySelector('.visually-hidden')?.textContent).toBe("sai, đúng là don't");
      expect(words[0].querySelector('.visually-hidden')?.textContent).toBe('thừa');
      expect(words[4].querySelector('.visually-hidden')?.textContent).toBe('thiếu');
      expect(text('.score')).toBe('3/6 từ đúng');
      expect(text('.correct-sentence')).toContain("I don't like green apples.");
    });

    it('replaces the result when checking again and keeps audio available', async () => {
      await setup();
      await check('i dont like green apples');
      expectPost().flush({ summary: summaryOf([result(0, 'x', 4, 5)]) });
      await settle();
      expect(text('.score')).toBe('4/5 từ đúng');

      await check("I don't like green apples");
      expectPost().flush({ summary: summaryOf([result(0, 'x', 5, 5)]) });
      await settle();
      expect(text('.score')).toBe('5/5 từ đúng');
      expect(el.querySelectorAll('.result .word.ok')).toHaveLength(5);
      expect(button('Nghe câu')!.disabled).toBe(false);
    });
  });

  describe('saving progress', () => {
    it('restores saved results and starts at the first unchecked sentence', async () => {
      await setup({ summary: summaryOf([result(0, 'i dont like green apples', 4, 5)]) });
      expect(text('.counter')).toBe('Câu 2/3');
      expect(text('.checked-count')).toBe('Đã kiểm tra 1/3');
      const chips = Array.from(el.querySelectorAll<HTMLElement>('.chip'));
      expect(chips[0].classList.contains('checked')).toBe(true);
      expect(chips[0].textContent).toContain('✓');
      expect(chips[0].getAttribute('aria-label')).toBe('Câu 1, đã kiểm tra');
      expect(chips[1].classList.contains('checked')).toBe(false);

      chips[0].click();
      await settle();
      expect(input().value).toBe('i dont like green apples');
      expect(text('.score')).toBe('4/5 từ đúng');
    });

    it('posts each check and completes after the last sentence', async () => {
      await setup({ summary: summaryOf([result(0, 'i dont like green apples', 4, 5), result(1, 'we went to the park at 9.30', 7, 7)]) });
      expect(text('.counter')).toBe('Câu 3/3');
      await check('what');
      const req = expectPost();
      expect(req.request.body).toEqual({ sentenceIndex: 2, typed: 'what', correctWords: 0, totalWords: 1 });
      req.flush({ summary: summaryOf([result(0, 'a', 4, 5), result(1, 'b', 7, 7), result(2, 'what', 0, 1)]) });
      await settle();

      expect(text('.done')).toContain('Đã hoàn thành bước Nghe');
      expect(text('.done')).toContain('85%'); // 11/13
      expect(completed).toBe(1);

      await check('why');
      expectPost().flush({ summary: summaryOf([result(0, 'a', 4, 5), result(1, 'b', 7, 7), result(2, 'why', 1, 1)]) });
      await settle();
      expect(completed).toBe(1);
    });

    it('keeps a failed save and resends it before the next one', async () => {
      await setup();
      await check("I don't like green apples");
      expectPost().flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
      await settle();
      expect(text('.save-error')).toContain('Chưa lưu được, sẽ thử lại');
      expect(text('.score')).toBe('5/5 từ đúng');

      button('Câu sau')!.click();
      await settle();
      await check('we went');
      const retry = expectPost();
      expect(retry.request.body.sentenceIndex).toBe(0);
      retry.flush({ summary: summaryOf([result(0, 'x', 5, 5)]) });
      await settle();
      const next = expectPost();
      expect(next.request.body.sentenceIndex).toBe(1);
      next.flush({ summary: summaryOf([result(0, 'x', 5, 5), result(1, 'we went', 2, 7)]) });
      await settle();
      expect(el.querySelector('.save-error')).toBeNull();
    });

    it('retries on demand', async () => {
      await setup();
      await check('why');
      expectPost().flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
      await settle();
      button('Thử lưu lại')!.click();
      await settle();
      expectPost().flush({ summary: summaryOf([result(0, 'why', 0, 5)]) });
      await settle();
      expect(el.querySelector('.save-error')).toBeNull();
    });
  });

  describe('daily flow', () => {
    it('opens the lesson given as input at the saved sentence and reports moves', async () => {
      await setup({ inputs: { lessonId: 'l9', startSentence: 1 } });
      expect(text('.counter')).toBe('Câu 2/3');
      const positions: number[] = [];
      fixture.componentInstance.position.subscribe((p) => positions.push(p));
      button('Câu sau')!.click();
      await settle();
      expect(positions).toEqual([2]);
    });

    it('neither restores nor saves results in review mode', async () => {
      await setup({ inputs: { mode: 'review' } });
      expect(text('.checked-count')).toBe('Đã kiểm tra 0/3');
      await check("I don't like green apples");
      http.expectNone((r) => r.url === '/api/lessons/l1/dictation');
      expect(text('.score')).toBe('5/5 từ đúng');
      expect(el.textContent).toContain('Xem lại');
    });

    it('uses review mode from ?review=1', async () => {
      await setup({ query: { review: '1' } });
      await check('why');
      http.expectNone((r) => r.url === '/api/lessons/l1/dictation');
    });

    it('says a locked lesson opens later', async () => {
      await setup({ lesson: 403 });
      expect(el.textContent).toContain('Bài này sẽ mở khi tới lượt');
    });
  });
});
