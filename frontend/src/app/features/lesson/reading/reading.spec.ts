import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { LookupResult, QuizAnswer, ReadingLesson } from '../../../core/models/reading';
import { Reading } from './reading';

const lesson: ReadingLesson = {
  id: 'l1',
  quiz: null,
  grammarNote: null,
  title: 'A day at the park',
  level: 'B1',
  topic: 'Daily',
  sentences: [
    { index: 0, text: 'We went to the park.' },
    { index: 1, text: 'He gave up smoking last year in the city.' },
    { index: 2, text: 'She goes home.' },
  ],
  paragraphs: [[0, 1], [2]],
  lemmas: { we: 'we', went: 'go', park: 'park', goes: 'go', gave: 'give' },
  phrases: [{ text: 'gave up', lemma: 'give up' }],
};

const wentResult: LookupResult = { source: 'ai', text: 'went', lemma: 'go', ipa: '/ɡəʊ/', meanings: [{ pos: '', text: 'đã đi' }] };

type IOCallback = (entries: { isIntersecting: boolean }[]) => void;

describe('Reading', () => {
  let fixture: ComponentFixture<Reading>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let observers: { cb: IOCallback; disconnect: ReturnType<typeof vi.fn> }[];
  let speech: FakeSpeech;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const word = (s: number, text: string) =>
    Array.from(el.querySelectorAll<HTMLElement>(`.w[data-s="${s}"]`)).find((w) => w.textContent === text)!;
  const popup = () => document.querySelector<HTMLElement>('lu-word-popup');
  const popupButton = (text: string) =>
    Array.from(popup()?.querySelectorAll('button') ?? []).find((b) => b.textContent?.trim() === text);
  const expectLookup = (q: string, sentence: number) => {
    const req = http.expectOne((r) => r.url === '/lessons/l1/lookup');
    expect(req.request.params.get('q')).toBe(q);
    expect(req.request.params.get('sentence')).toBe(String(sentence));
    return req;
  };

  const setup = async (
    options: {
      words?: { lemma: string; text: string }[];
      io?: boolean;
      inputs?: Record<string, unknown>;
      query?: Record<string, string>;
      status?: number;
      lesson?: ReadingLesson;
    } = {},
  ) => {
    observers = [];
    if (options.io === false) {
      vi.stubGlobal('IntersectionObserver', undefined);
    } else {
      vi.stubGlobal(
        'IntersectionObserver',
        class {
          disconnect = vi.fn();
          constructor(cb: IOCallback) {
            observers.push({ cb, disconnect: this.disconnect });
          }
          observe = vi.fn();
        },
      );
    }
    speech = new FakeSpeech();
    await TestBed.configureTestingModule({
      imports: [Reading],
      providers: [
        provideFakeSpeech(speech),
        provideRouter([{ path: 'forbidden', children: [] }]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ id: 'l1' }), queryParamMap: convertToParamMap(options.query ?? {}) },
          },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Reading);
    for (const [name, value] of Object.entries(options.inputs ?? {})) {
      fixture.componentRef.setInput(name, value);
    }
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const id = (options.inputs?.['lessonId'] as string | undefined) ?? 'l1';
    const lessonReq = http.expectOne(`/lessons/${id}`);
    const wordsReq = http.expectOne('/vocab/words');
    if (options.status) {
      lessonReq.flush({ error: 'lesson_locked', message: 'x' }, { status: options.status, statusText: 'Error' });
    } else {
      lessonReq.flush({ lesson: options.lesson ?? lesson });
      wordsReq.flush({ words: options.words ?? [] });
    }
    await settle();
  };

  afterEach(() => {
    http.verify();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  // --- US1: render and look up ---

  it('offers a way back to the lessons only when opened on its own', async () => {
    await setup();
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')?.getAttribute('href')).toBe('/lessons');
    fixture.destroy();
    TestBed.resetTestingModule();
    await setup({ inputs: { lessonId: 'l1', mode: 'study' } });
    expect(el.querySelector('a[aria-label="Quay lại danh sách bài"]')).toBeNull();
  });

  describe('reading and looking up', () => {
    beforeEach(() => setup());

    it('renders paragraphs, sentences and clickable words', () => {
      expect(el.querySelector('h1')?.textContent).toContain('A day at the park');
      expect(el.querySelectorAll('.para')).toHaveLength(2);
      expect(el.querySelectorAll('.para')[0].querySelectorAll('.sentence')).toHaveLength(2);
      const w = word(0, 'went');
      expect(w.dataset['t']).toBeDefined();
      expect(el.querySelector('.para')?.textContent).toContain('We went to the park.');
      expect(Array.from(el.querySelectorAll('.w')).some((x) => x.textContent === '.')).toBe(false);
    });

    it('looks up a tapped word with its sentence and shows the popup', async () => {
      word(0, 'went').click();
      await settle();
      expect(popup()?.textContent).toContain('Đang tra');
      expectLookup('went', 0).flush(wentResult);
      await settle();
      expect(popup()?.textContent).toContain('đã đi');
      expect(popup()?.textContent).toContain('AI · theo ngữ cảnh');
    });

    it('switches to another word', async () => {
      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({ source: 'dictionary', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: 'N', text: 'Công viên.' }] });
      await settle();
      expect(popup()?.textContent).toContain('Công viên.');
    });

    it('closes with Escape and returns focus to the word', async () => {
      const w = word(0, 'went');
      w.click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      document.querySelector('.cdk-overlay-pane')!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
      await settle();
      expect(popup()).toBeNull();
      expect(document.activeElement).toBe(w);
    });

    it('opens the popup from the keyboard', async () => {
      const w = word(0, 'We');
      expect(w.tabIndex).toBe(0);
      expect(word(0, 'went').tabIndex).toBe(-1);
      w.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await settle();
      expect(document.activeElement).toBe(word(0, 'went'));
      word(0, 'went').dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
      await settle();
      expectLookup('went', 0).flush(wentResult);
    });

    it('shows the not-found state with a manual meaning field', async () => {
      word(1, 'smoking').click();
      await settle();
      expectLookup('smoking', 1).flush({ error: 'not_found', message: 'Chưa có nghĩa' }, { status: 404, statusText: 'Not Found' });
      await settle();
      expect(popup()?.textContent).toContain('Chưa có nghĩa');
      expect(popup()?.querySelector('input[name="meaning"]')).not.toBeNull();
    });
  });

  // --- F9: Hỏi AI ---

  describe('asking the AI', () => {
    const askedSmoking: LookupResult = {
      source: 'ai', text: 'smoking', lemma: 'smoking', ipa: '',
      meanings: [{ pos: '', text: 'việc hút thuốc' }], note: 'Danh động từ sau "gave up".',
    };
    const expectAsk = (text: string, sentenceIndex: number) => {
      const req = http.expectOne('/lessons/l1/ask');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ text, sentenceIndex });
      return req;
    };
    const openSmoking = async () => {
      word(1, 'smoking').click();
      await settle();
      expectLookup('smoking', 1).flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
      await settle();
    };

    beforeEach(() => setup());

    it('asks about an unknown word, shows the answer and saves it as an AI meaning', async () => {
      await openSmoking();
      popupButton('Hỏi AI')!.click();
      await settle();
      expect(popupButton('Đang hỏi AI…')?.disabled).toBe(true);
      expectAsk('smoking', 1).flush({ result: askedSmoking, cached: false });
      await settle();
      expect(popup()?.textContent).toContain('việc hút thuốc');
      expect(popup()?.textContent).toContain('Danh động từ');
      expect(popupButton('Hỏi AI')).toBeUndefined();

      popupButton('Lưu vào sổ từ')!.click();
      await settle();
      const req = http.expectOne('/vocab/cards');
      expect(req.request.body).toMatchObject({
        text: 'smoking', lemma: 'smoking', meaningVi: 'việc hút thuốc', source: 'ai',
        contextSentence: 'He gave up smoking last year in the city.',
      });
      req.flush({ card: { id: 'c3' } }, { status: 201, statusText: 'Created' });
      await settle();
    });

    it('asks about a dictionary word', async () => {
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({ source: 'dictionary', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: 'N', text: 'Công viên.' }] });
      await settle();
      popupButton('Hỏi AI')!.click();
      await settle();
      expectAsk('park', 0).flush({
        result: { source: 'ai', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: '', text: 'công viên' }], note: 'n' },
        cached: true,
      });
      await settle();
      expect(popup()?.textContent).toContain('AI · theo ngữ cảnh');
    });

    it('keeps the dictionary meaning and shows the server message when the AI fails', async () => {
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({ source: 'dictionary', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: 'N', text: 'Công viên.' }] });
      await settle();
      popupButton('Hỏi AI')!.click();
      await settle();
      expectAsk('park', 0).flush(
        { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' },
        { status: 429, statusText: 'Too Many Requests' },
      );
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('Đã hết lượt AI');
      expect(popup()?.textContent).toContain('Công viên.');
      expect(popupButton('Hỏi AI')?.disabled).toBe(false);

      popupButton('Lưu vào sổ từ')!.click();
      await settle();
      const req = http.expectOne('/vocab/cards');
      expect(req.request.body).toMatchObject({ lemma: 'park', meaningVi: 'Công viên.', source: 'dictionary' });
      req.flush({ card: { id: 'c4' } }, { status: 201, statusText: 'Created' });
      await settle();
    });

    it('keeps the manual meaning field when the AI is not configured', async () => {
      await openSmoking();
      popupButton('Hỏi AI')!.click();
      await settle();
      expectAsk('smoking', 1).flush(
        { error: 'ai_not_configured', message: 'AI chưa được cấu hình. Liên hệ người vận hành.' },
        { status: 503, statusText: 'Service Unavailable' },
      );
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('AI chưa được cấu hình');
      expect(popup()?.querySelector('input[name="meaning"]')).not.toBeNull();
    });

    it('shows a generic message on a network error', async () => {
      await openSmoking();
      popupButton('Hỏi AI')!.click();
      await settle();
      expectAsk('smoking', 1).error(new ProgressEvent('error'));
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('Không hỏi được AI');
    });

    it('drops the answer when another word was opened meanwhile', async () => {
      await openSmoking();
      popupButton('Hỏi AI')!.click();
      await settle();
      const req = expectAsk('smoking', 1);
      word(0, 'went').click();
      await settle();
      expect(req.cancelled).toBe(true);
      expectLookup('went', 0).flush(wentResult);
      await settle();
      expect(popup()?.textContent).toContain('đã đi');
      expect(popup()?.querySelector('[role="alert"]')).toBeNull();
    });
  });

  it('shows a message for a missing lesson', async () => {
    vi.stubGlobal('IntersectionObserver', undefined);
    await TestBed.configureTestingModule({
      imports: [Reading],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 'l1' }) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Reading);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const words = http.expectOne('/vocab/words');
    http.expectOne('/lessons/l1').flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
    await settle();
    expect(words.cancelled).toBe(true); // no need to wait for saved words
    expect(el.textContent).toContain('Không tìm thấy bài học');
  });

  // --- US2: saving and highlighting ---

  describe('saving', () => {
    it('highlights saved words by base form', async () => {
      await setup({ words: [{ lemma: 'go', text: 'went' }] });
      expect(word(0, 'went').classList).toContain('saved');
      expect(word(2, 'goes').classList).toContain('saved');
      expect(word(0, 'park').classList).not.toContain('saved');
    });

    it('saves a word with its sentence and highlights it', async () => {
      await setup();
      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      popupButton('Lưu vào sổ từ')!.click();
      await settle();

      const req = http.expectOne('/vocab/cards');
      expect(req.request.body).toEqual({
        text: 'went', lemma: 'go', ipa: '/ɡəʊ/', meaningVi: 'đã đi',
        contextSentence: 'We went to the park.', lessonId: 'l1', source: 'ai',
      });
      req.flush({ card: { id: 'c1' } }, { status: 201, statusText: 'Created' });
      await settle();
      expect(popup()?.textContent).toContain('✓ Đã có trong sổ');
      expect(word(2, 'goes').classList).toContain('saved');
    });

    it('marks a word saved from the popup in the vocabulary section', async () => {
      await setup();
      el.querySelector<HTMLButtonElement>('lu-lesson-vocabulary .toggle')!.click();
      await settle();
      http.expectOne('/lessons/l1/vocabulary').flush({
        available: true,
        items: [{ lemma: 'go', text: 'went', meaningVi: 'đi', ipa: '', sentenceIndex: 0, sentence: 'We went to the park.' }],
      });
      await settle();
      const item = () => el.querySelector('lu-lesson-vocabulary .vocab-item')!;
      expect(item().textContent).not.toContain('✓ Đã lưu');

      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      popupButton('Lưu vào sổ từ')!.click();
      await settle();
      http.expectOne('/vocab/cards').flush({ card: { id: 'c1' } }, { status: 201, statusText: 'Created' });
      await settle();
      expect(item().textContent).toContain('✓ Đã lưu');
    });

    it('treats a duplicate as already saved', async () => {
      await setup();
      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      popupButton('Lưu vào sổ từ')!.click();
      await settle();
      http.expectOne('/vocab/cards').flush({ error: 'card_exists', card: {} }, { status: 409, statusText: 'Conflict' });
      await settle();
      expect(popup()?.textContent).toContain('✓ Đã có trong sổ');
    });

    it('joins dictionary meanings and saves a manual meaning', async () => {
      await setup();
      word(1, 'smoking').click();
      await settle();
      expectLookup('smoking', 1).flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
      await settle();
      const input = popup()!.querySelector<HTMLInputElement>('input[name="meaning"]')!;
      input.value = 'hút thuốc';
      input.dispatchEvent(new Event('input'));
      popupButton('Lưu')!.click();
      await settle();
      const req = http.expectOne('/vocab/cards');
      expect(req.request.body).toMatchObject({ text: 'smoking', lemma: 'smoking', meaningVi: 'hút thuốc', source: 'manual', ipa: '' });
      req.flush({ card: { id: 'c2' } }, { status: 201, statusText: 'Created' });
      await settle();
    });

    it('keeps the popup open with an alert when saving fails', async () => {
      await setup();
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({
        source: 'dictionary', text: 'park', lemma: 'park', ipa: '',
        meanings: [{ pos: 'N', text: 'Công viên.' }, { pos: 'N', text: 'Bãi.' }, { pos: 'V', text: 'Đỗ.' }, { pos: 'V', text: 'Bỏ qua.' }],
      });
      await settle();
      popupButton('Lưu vào sổ từ')!.click();
      await settle();
      const req = http.expectOne('/vocab/cards');
      expect(req.request.body.meaningVi).toBe('Công viên.; Bãi.; Đỗ.');
      req.error(new ProgressEvent('error'));
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('Không lưu được');
    });
  });

  // --- US3: phrases ---

  describe('phrases', () => {
    const selectRange = async (from: HTMLElement, to: HTMLElement, fromOffset = 0, toOffset?: number) => {
      const range = document.createRange();
      range.setStart(from.firstChild!, fromOffset);
      range.setEnd(to.firstChild!, toOffset ?? to.textContent!.length);
      const sel = document.getSelection()!;
      sel.removeAllRanges();
      sel.addRange(range);
      el.querySelector('.article')!.dispatchEvent(new Event('pointerup', { bubbles: true }));
      await settle();
    };

    it('looks up a selected phrase, expanding partial words', async () => {
      await setup();
      await selectRange(word(1, 'gave'), word(1, 'up'), 1, 1); // "ave u"
      expectLookup('gave up', 1).flush({ source: 'ai', text: 'gave up', lemma: 'give up', ipa: '', meanings: [{ pos: '', text: 'từ bỏ' }] });
      await settle();
      expect(popup()?.textContent).toContain('từ bỏ');
    });

    it('keeps only the first sentence of a selection', async () => {
      await setup();
      await selectRange(word(0, 'the'), word(1, 'He'));
      expectLookup('the park', 0).flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
      await settle();
    });

    it('asks for at most 6 words', async () => {
      await setup();
      await selectRange(word(1, 'He'), word(1, 'city')); // 9 words
      expect(el.querySelector('.hint')?.textContent).toContain('Chọn tối đa 6 từ');
      http.expectNone((r) => r.url === '/lessons/l1/lookup');
    });

    it('highlights a saved phrase', async () => {
      await setup({ words: [{ lemma: 'give up', text: 'gave up' }] });
      expect(word(1, 'gave').classList).toContain('saved');
      expect(word(1, 'up').classList).toContain('saved');
      expect(word(1, 'smoking').classList).not.toContain('saved');
    });
  });

  // --- US4: pronunciation ---

  describe('pronunciation', () => {
    beforeEach(() => setup());

    it('reads the base form with the browser voice', async () => {
      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      expect(speech.texts()).toEqual(['go', 'go']);
    });

    it('shows a message when the browser cannot read', async () => {
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({ source: 'dictionary', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: 'N', text: 'Công viên.' }] });
      await settle();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      speech.last().handlers.failed!('synthesis-failed');
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('Chưa đọc được từ này');
      expect(popup()?.textContent).toContain('Công viên.');
    });
  });

  // --- US5: done button ---

  describe('done', () => {
    const doneButton = () =>
      Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Đã đọc xong')!;

    it('enables Đã đọc xong only after the end is visible, then completes', async () => {
      await setup();
      expect(doneButton().disabled).toBe(true);
      expect(el.textContent).toContain('Đọc hết bài để hoàn thành');

      observers[0].cb([{ isIntersecting: false }]);
      await settle();
      expect(doneButton().disabled).toBe(true);

      observers[0].cb([{ isIntersecting: true }]);
      await settle();
      expect(doneButton().disabled).toBe(false);
      expect(observers[0].disconnect).toHaveBeenCalled();

      const completed = vi.fn();
      fixture.componentInstance.completed.subscribe(completed);
      doneButton().click();
      await settle();
      expect(completed).toHaveBeenCalledTimes(1);
      expect(el.textContent).toContain('Đã hoàn thành bước Đọc');
    });

    it('is enabled at once without IntersectionObserver', async () => {
      await setup({ io: false });
      expect(doneButton().disabled).toBe(false);
    });
  });

  // --- L: embedded in the daily flow, review mode, locked lessons ---

  describe('daily flow', () => {
    const sentence = (i: number) => el.querySelector<HTMLElement>(`.sentence[data-s="${i}"]`)!;

    it('opens the lesson given as input at the saved sentence', async () => {
      const scroll = vi.fn();
      Element.prototype.scrollIntoView = scroll;
      await setup({ inputs: { lessonId: 'l9', startSentence: 2 } });
      expect(scroll).toHaveBeenCalled();
      expect(scroll.mock.contexts.at(-1)).toBe(sentence(2));
    });

    it('reports the first visible sentence', async () => {
      await setup();
      const positions: number[] = [];
      fixture.componentInstance.position.subscribe((p) => positions.push(p));
      const tracker = observers[1];
      tracker.cb([
        { isIntersecting: true, target: sentence(1) },
        { isIntersecting: true, target: sentence(2) },
      ] as unknown as { isIntersecting: boolean }[]);
      tracker.cb([{ isIntersecting: false, target: sentence(1) }] as unknown as { isIntersecting: boolean }[]);
      tracker.cb([{ isIntersecting: true, target: sentence(2) }] as unknown as { isIntersecting: boolean }[]);
      expect(positions).toEqual([1, 2]);
    });

    it('hides the done button in review mode', async () => {
      await setup({ inputs: { mode: 'review' } });
      expect(Array.from(el.querySelectorAll('button')).some((b) => b.textContent?.trim() === 'Đã đọc xong')).toBe(false);
      expect(el.textContent).toContain('Xem lại');
    });

    it('uses review mode from ?review=1', async () => {
      await setup({ query: { review: '1' } });
      expect(Array.from(el.querySelectorAll('button')).some((b) => b.textContent?.trim() === 'Đã đọc xong')).toBe(false);
    });

    it('says a locked lesson opens later', async () => {
      await setup({ status: 403 });
      expect(el.textContent).toContain('Bài này sẽ mở khi tới lượt');
    });
  });

  // --- F15: comprehension questions and grammar note ---

  describe('comprehension questions', () => {
    const withQuiz = (answers: QuizAnswer[] = []): ReadingLesson => ({
      ...lesson,
      quiz: {
        version: 2,
        questions: [
          { prompt: 'Where did we go?', options: ['Park', 'Home', 'School', 'Work'] },
          { prompt: 'Who goes home?', options: ['He', 'She', 'We', 'They'] },
        ],
        answers,
      },
      grammarNote: { title: 'Quá khứ đơn', bodyVi: 'Việc đã xong.', examples: ['We went to the park.'] },
    });
    const solved = (questionIndex: number, choice: number, answerIndex: number) => ({
      questionIndex,
      choice,
      correct: choice === answerIndex,
      answerIndex,
      explanationVi: 'Vì vậy.',
    });
    const button = (label: string) =>
      Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label);
    const choose = async (option: number, result: ReturnType<typeof solved>, answered: number) => {
      el.querySelectorAll<HTMLInputElement>('.question.active .option input')[option].click();
      await fixture.whenStable();
      el.querySelector<HTMLButtonElement>('lu-comprehension-quiz .step-nav .check')!.click();
      await settle();
      http.expectOne('/lessons/l1/answers').flush({ answer: result, answered, total: 2, correct: 0 });
      await settle();
      // The answered question stays on screen: move to the next one.
      button('Câu tiếp theo')?.click();
      await settle();
    };

    it('replaces Đã đọc xong with the questions and shows the grammar note', async () => {
      await setup({ lesson: withQuiz(), inputs: { mode: 'study' } });
      expect(button('Đã đọc xong')).toBeUndefined();
      expect(el.querySelector('lu-comprehension-quiz')).not.toBeNull();
      expect(el.querySelector('lu-grammar-note summary')?.textContent?.trim()).toBe('Ngữ pháp: Quá khứ đơn');
      expect(el.textContent).toContain('Trả lời hết câu hỏi để hoàn thành bước Đọc.');
    });

    it('completes the step when the last question is answered, even wrongly', async () => {
      await setup({ lesson: withQuiz(), inputs: { mode: 'study' } });
      const completed = vi.fn();
      fixture.componentInstance.completed.subscribe(completed);
      await choose(1, solved(0, 1, 0), 1);
      expect(completed).not.toHaveBeenCalled();
      await choose(0, solved(1, 0, 2), 2);
      expect(completed).toHaveBeenCalledTimes(1);
      expect(el.textContent).toContain('✓ Đã hoàn thành bước Đọc');
    });

    it('offers Tiếp tục when every question was already answered', async () => {
      await setup({ lesson: withQuiz([solved(0, 0, 0), solved(1, 2, 2)]), inputs: { mode: 'study' } });
      const completed = vi.fn();
      fixture.componentInstance.completed.subscribe(completed);
      button('Tiếp tục')!.click();
      await settle();
      expect(completed).toHaveBeenCalledTimes(1);
    });

    it('shows past answers in review mode without completing', async () => {
      await setup({ lesson: withQuiz([solved(0, 0, 0), solved(1, 1, 2)]), query: { review: '1' } });
      const completed = vi.fn();
      fixture.componentInstance.completed.subscribe(completed);
      // One question at a time: the first one with its result, the second one behind →.
      expect(el.querySelectorAll('.verdict').length).toBe(1);
      expect(el.querySelectorAll('.question.active .option').length).toBe(0);
      el.querySelectorAll<HTMLButtonElement>('lu-comprehension-quiz .step-nav .nav-btn')[1].click();
      await settle();
      expect(el.querySelector('.verdict')?.textContent).toContain('Sai');
      expect(button('Tiếp tục')).toBeUndefined();
      expect(completed).not.toHaveBeenCalled();
    });

    it('reloads the lesson when the questions changed', async () => {
      await setup({ lesson: withQuiz(), inputs: { mode: 'study' } });
      el.querySelectorAll<HTMLInputElement>('.question.active .option input')[0].click();
      await fixture.whenStable();
      el.querySelector<HTMLButtonElement>('lu-comprehension-quiz .step-nav .check')!.click();
      await settle();
      http
        .expectOne('/lessons/l1/answers')
        .flush({ error: 'quiz_changed', message: 'x' }, { status: 409, statusText: 'Conflict' });
      await settle();
      button('Tải lại')!.click();
      await settle();
      http.expectOne('/lessons/l1').flush({ lesson: { ...withQuiz(), quiz: null, grammarNote: null } });
      http.expectOne('/vocab/words').flush({ words: [] });
      await settle();
      expect(el.querySelector('lu-comprehension-quiz')).toBeNull();
      expect(button('Đã đọc xong')).toBeDefined();
    });

    it('keeps Đã đọc xong for a lesson without questions or grammar note', async () => {
      await setup({ inputs: { mode: 'study' } });
      expect(button('Đã đọc xong')).toBeDefined();
      expect(el.querySelector('lu-comprehension-quiz')).toBeNull();
      expect(el.querySelector('lu-grammar-note')).toBeNull();
    });
  });
});
