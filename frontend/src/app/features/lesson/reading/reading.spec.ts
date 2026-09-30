import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { LookupResult, ReadingLesson } from '../../../core/models/reading';
import { Reading } from './reading';

const lesson: ReadingLesson = {
  id: 'l1',
  title: 'A day at the park',
  level: 'B1',
  topic: 'Daily',
  sentences: [
    { index: 0, text: 'We went to the park.', audioUrl: null },
    { index: 1, text: 'He gave up smoking last year in the city.', audioUrl: null },
    { index: 2, text: 'She goes home.', audioUrl: null },
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
    const req = http.expectOne((r) => r.url === '/api/lessons/l1/lookup');
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
    await TestBed.configureTestingModule({
      imports: [Reading],
      providers: [
        provideRouter([]),
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
    const lessonReq = http.expectOne(`/api/lessons/${id}`);
    const wordsReq = http.expectOne('/api/vocab/words');
    if (options.status) {
      lessonReq.flush({ error: 'lesson_locked', message: 'x' }, { status: options.status, statusText: 'Error' });
    } else {
      lessonReq.flush({ lesson });
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
    const words = http.expectOne('/api/vocab/words');
    http.expectOne('/api/lessons/l1').flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
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

      const req = http.expectOne('/api/vocab/cards');
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
      http.expectOne('/api/lessons/l1/vocabulary').flush({
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
      http.expectOne('/api/vocab/cards').flush({ card: { id: 'c1' } }, { status: 201, statusText: 'Created' });
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
      http.expectOne('/api/vocab/cards').flush({ error: 'card_exists', card: {} }, { status: 409, statusText: 'Conflict' });
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
      const req = http.expectOne('/api/vocab/cards');
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
      const req = http.expectOne('/api/vocab/cards');
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
      http.expectNone((r) => r.url === '/api/lessons/l1/lookup');
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

    it('plays the base form through one shared audio element', async () => {
      const play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue();
      word(0, 'went').click();
      await settle();
      expectLookup('went', 0).flush(wentResult);
      await settle();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      expect(play).toHaveBeenCalledTimes(2);
      expect(document.querySelectorAll('audio.word-audio')).toHaveLength(1);
      expect(document.querySelector<HTMLAudioElement>('audio.word-audio')!.src).toContain('/api/tts/word?text=go');
    });

    it('shows a message when audio cannot play', async () => {
      vi.spyOn(HTMLMediaElement.prototype, 'play').mockRejectedValue(new Error('503'));
      word(0, 'park').click();
      await settle();
      expectLookup('park', 0).flush({ source: 'dictionary', text: 'park', lemma: 'park', ipa: '', meanings: [{ pos: 'N', text: 'Công viên.' }] });
      await settle();
      popup()!.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
      await settle();
      expect(popup()?.querySelector('[role="alert"]')?.textContent).toContain('Chưa phát được âm thanh');
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
});
