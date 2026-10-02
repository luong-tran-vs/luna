import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { PracticeView } from '../../../core/models/practice';
import { ReadingLesson } from '../../../core/models/reading';
import { MyLessons } from '../../../core/models/study';
import { LessonVocabulary } from '../../../core/models/vocab';
import { LessonDetail } from './lesson-detail';

const lesson: ReadingLesson = {
  id: 'l1',
  title: 'Introductions',
  level: 'A1',
  topic: 'Làm quen',
  sentences: [
    { index: 0, text: 'Hello, my name is Minh.' },
    { index: 1, text: 'Nice to meet you.' },
    { index: 2, text: 'I am from Vietnam.' },
  ],
  paragraphs: [[0, 1], [2]],
  lemmas: {},
  phrases: [],
  quiz: null,
  grammarNote: {
    title: 'Giới thiệu tên',
    bodyVi: 'Dùng "My name is…".',
    examples: ['My name is Minh.'],
  },
};

const vocabulary: LessonVocabulary = {
  available: true,
  items: [
    {
      lemma: 'name',
      text: 'name',
      meaningVi: 'tên',
      ipa: '/neɪm/',
      sentenceIndex: 0,
      sentence: '',
    },
    {
      lemma: 'meet',
      text: 'meet',
      meaningVi: 'gặp',
      ipa: '/miːt/',
      sentenceIndex: 1,
      sentence: '',
    },
  ],
};

const practice: PracticeView = {
  status: 'done',
  lessonNumber: 3,
  objectiveVi: 'Bạn có thể chào hỏi và giới thiệu bản thân.',
  examples: [{ lemma: 'name', sentence: "What's your name?" }],
  dialogue: {
    speakers: ['Minh', 'Anna'],
    turns: [
      {
        speaker: 0,
        text: 'Hi, my name is Minh.',
        meaningVi: 'Chào, mình tên Minh.',
      },
      {
        speaker: 1,
        text: 'Nice to meet you.',
        meaningVi: 'Rất vui được gặp bạn.',
      },
    ],
  },
  fill: {
    turns: [
      {
        speaker: 0,
        turnIndex: 0,
        meaningVi: 'Chào, mình tên Minh.',
        parts: [{ text: 'Hi, my ' }, { blank: 0 }, { text: ' is Minh.' }],
      },
      {
        speaker: 1,
        turnIndex: 1,
        meaningVi: 'Rất vui được gặp bạn.',
        parts: [{ text: 'Nice to ' }, { blank: 1 }, { text: ' you.' }],
      },
    ],
    blanks: [{ answer: 'name' }, { answer: 'meet' }],
    wordBank: ['name', 'meet', 'from'],
  },
  grammarTipVi: 'Dùng "Nice to meet you" khi gặp lần đầu.',
  translations: [
    {
      vi: 'Tên tôi là Minh.',
      answer: ['My', 'name', 'is', 'Minh.'],
      tiles: ['My', 'name', 'is', 'Minh.', 'are'],
    },
    {
      vi: 'Rất vui được gặp bạn.',
      answer: ['Nice', 'to', 'meet', 'you.'],
      tiles: ['Nice', 'to', 'meet', 'you.', 'see'],
    },
  ],
};

const noPractice: PracticeView = {
  status: 'running',
  lessonNumber: 0,
  objectiveVi: '',
  examples: [],
  dialogue: null,
  fill: null,
  grammarTipVi: '',
  translations: [],
};

const mine = (over: Partial<MyLessons> = {}): MyLessons => ({
  today: null,
  completed: [],
  upcoming: [],
  ...over,
});

interface Options {
  status?: number;
  errorCode?: string;
  vocabulary?: LessonVocabulary;
  practice?: PracticeView | 'error';
  mine?: MyLessons;
  dashboard?: 'error';
}

describe('LessonDetail', () => {
  let speech: FakeSpeech;
  let fixture: ComponentFixture<LessonDetail>;
  let http: HttpTestingController;
  let el: HTMLElement;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
  const progress = () => el.querySelector('[role="progressbar"]')!;
  const next = async () => {
    el.querySelector<HTMLButtonElement>('.next-bar button')!.click();
    await fixture.whenStable();
  };

  const open = async (options: Options = {}) => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      imports: [LessonDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: convertToParamMap({ id: 'l1' }) } },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonDetail);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    if (options.status) {
      http
        .expectOne('/api/lessons/l1')
        .flush(
          { error: options.errorCode ?? 'x', message: 'x' },
          { status: options.status, statusText: 'Error' },
        );
      // The other requests were cancelled with the lesson's error.
      expect(http.match(() => true).every((r) => r.cancelled)).toBe(true);
      await fixture.whenStable();
      return;
    }
    http.expectOne('/api/lessons/l1/vocabulary').flush(options.vocabulary ?? vocabulary);
    const practiceReq = http.expectOne('/api/lessons/l1/practice');
    if (options.practice === 'error') {
      practiceReq.flush({ error: 'x' }, { status: 500, statusText: 'Error' });
    } else {
      practiceReq.flush(options.practice ?? practice);
    }
    http.expectOne('/api/lessons/mine').flush(options.mine ?? mine());
    const dash = http.expectOne('/api/dashboard');
    if (options.dashboard === 'error') {
      dash.flush('down', { status: 500, statusText: 'Error' });
    } else {
      dash.flush({ streak: 3 });
    }
    http.expectOne('/api/lessons/l1').flush({ lesson });
    await fixture.whenStable();
  };

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  describe('header', () => {
    it('shows the close link, progress, streak, lesson number, title, objective and level', async () => {
      await open();
      expect(el.querySelector('a[aria-label="Đóng, về danh sách bài"]')?.getAttribute('href')).toBe(
        '/lessons',
      );
      expect(progress().getAttribute('aria-valuenow')).toBe('1');
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      expect(text(el.querySelector('.streak'))).toBe('3 ngày');
      expect(text(el.querySelector('h1'))).toBe('Bài 3 Introductions');
      expect(text(el.querySelector('.objective'))).toBe(
        'Mục tiêu: Bạn có thể chào hỏi và giới thiệu bản thân.',
      );
      expect(text(el.querySelector('.level'))).toBe('Mức độ: Cơ bản');
    });

    it('hides the lesson number, objective and streak when they are missing', async () => {
      await open({ practice: noPractice, dashboard: 'error' });
      expect(text(el.querySelector('h1'))).toBe('Introductions');
      expect(el.querySelector('.objective')).toBeNull();
      expect(el.querySelector('.streak')).toBeNull();
    });

    it('explains a locked lesson and links back to the list', async () => {
      await open({ status: 403, errorCode: 'lesson_locked' });
      expect(text(el.querySelector('[role="alert"]'))).toBe('Bài này sẽ mở khi tới lượt.');
      expect(el.querySelector('a.btn[href="/lessons"]')).not.toBeNull();
      expect(el.querySelector('[role="tablist"]')).toBeNull();
    });

    it('says when the lesson does not exist', async () => {
      await open({ status: 404 });
      expect(text(el.querySelector('[role="alert"]'))).toBe('Không tìm thấy bài học.');
    });
  });

  describe('steps', () => {
    it('starts on step 1 with the words and their examples', async () => {
      await open();
      expect(text(el.querySelector('#vocab-heading'))).toContain('1. Từ vựng quan trọng');
      expect(el.querySelectorAll('.word').length).toBe(2);
      expect(text(el.querySelector('.example'))).toContain(`Ví dụ: "What's your name?"`);
    });

    it('Next moves through the 4 steps and updates the progress', async () => {
      await open();
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('2');
      expect(text(el.querySelector('.progress-text'))).toBe('2/4');
      expect(text(el.querySelector('#dialogue-heading'))).toBe('2. Hội thoại mẫu');
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('3');
      expect(el.querySelector('lu-fill-step')).not.toBeNull();
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('4');
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 1/2');
    });

    it('reads a word with the browser voice', async () => {
      await open();
      el.querySelector<HTMLButtonElement>('button[aria-label="Nghe từ name"]')!.click();
      await fixture.whenStable();
      expect(speech.last()).toMatchObject({ text: 'name', rate: 1 });
    });

    it('without practice: words without examples, then "no practice" for steps 2–4', async () => {
      await open({ practice: 'error' });
      expect(el.querySelectorAll('.word').length).toBe(2);
      expect(el.querySelector('.example')).toBeNull();
      for (const n of [2, 3, 4]) {
        await next();
        expect(progress().getAttribute('aria-valuenow')).toBe(String(n));
        expect(text(el.querySelector('.empty'))).toBe('Bài này chưa có phần luyện tập');
      }
      expect(text(el.querySelector('.next-bar button'))).toBe('Hoàn thành');
    });

    it('says when the lesson has no words', async () => {
      await open({ vocabulary: { available: false, items: [] }, practice: noPractice });
      expect(text(el.querySelector('.empty'))).toBe('Bài này chưa có từ vựng.');
    });
  });

  describe('tabs', () => {
    it('Bài đọc shows the text and grammar note; back to Bài học keeps the step', async () => {
      await open();
      await next();
      const reading = el.querySelector<HTMLButtonElement>('#tab-reading')!;
      reading.click();
      await fixture.whenStable();
      expect(reading.getAttribute('aria-selected')).toBe('true');
      expect(el.querySelector<HTMLElement>('#panel-lesson')!.hidden).toBe(true);
      const paragraphs = Array.from(el.querySelectorAll('#panel-reading .reading p')).map((p) =>
        text(p),
      );
      expect(paragraphs).toEqual([
        'Hello, my name is Minh. Nice to meet you.',
        'I am from Vietnam.',
      ]);
      expect(text(el.querySelector('#grammar-heading'))).toBe('Ngữ pháp: Giới thiệu tên');
      expect(el.querySelector('.next-bar')).toBeNull();

      el.querySelector<HTMLButtonElement>('#tab-lesson')!.click();
      await fixture.whenStable();
      expect(el.querySelector<HTMLElement>('#panel-lesson')!.hidden).toBe(false);
      expect(progress().getAttribute('aria-valuenow')).toBe('2');
      expect(el.querySelector('lu-dialogue-step')).not.toBeNull();
    });

    it('arrow keys switch tabs and move focus', async () => {
      await open();
      const lessonTab = el.querySelector<HTMLButtonElement>('#tab-lesson')!;
      lessonTab.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await fixture.whenStable();
      const readingTab = el.querySelector<HTMLButtonElement>('#tab-reading')!;
      expect(readingTab.getAttribute('aria-selected')).toBe('true');
      expect(document.activeElement).toBe(readingTab);
      readingTab.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
      await fixture.whenStable();
      expect(lessonTab.getAttribute('aria-selected')).toBe('true');
    });
  });

  describe('step 4 and summary', () => {
    const toStep4 = async () => {
      await next();
      await next();
      await next();
    };

    const fillAll = async (words: string[]) => {
      for (const w of words) {
        Array.from(el.querySelectorAll<HTMLButtonElement>('lu-fill-step .tiles .tile'))
          .find((b) => text(b) === w && !b.disabled)!
          .click();
        await fixture.whenStable();
      }
      button('Kiểm tra')!.click();
      await fixture.whenStable();
    };

    const build = async (words: string[]) => {
      for (const w of words) {
        Array.from(
          el.querySelectorAll<HTMLButtonElement>(
            'lu-translate-step ul[aria-labelledby="tiles-heading"] .tile',
          ),
        )
          .find((b) => text(b) === w && !b.disabled)!
          .click();
        await fixture.whenStable();
      }
      button('Kiểm tra')!.click();
      await fixture.whenStable();
    };

    it('Next goes through the sentences, then Hoàn thành shows the summary', async () => {
      await open();
      await next();
      await next();
      await fillAll(['name', 'from']);
      await next();
      expect(text(el.querySelector('.next-bar button'))).toBe('Tiếp theo');
      await build(['My', 'name', 'is', 'Minh.']);
      expect(text(el.querySelector('lu-translate-step [role="status"]'))).toBe('Chính xác!');
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('4');
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 2/2');
      expect(text(el.querySelector('.next-bar button'))).toBe('Hoàn thành');
      await next(); // second sentence not checked: counts wrong
      const scores = Array.from(el.querySelectorAll('.scores li')).map((li) => text(li));
      expect(scores).toEqual(['Điền đúng 1/2 ô', 'Dịch đúng 1/2 câu']);
      expect(el.querySelector('.next-bar')).toBeNull();
      // Results stay on the page: nothing was sent.
      http.expectNone(() => true);
    });

    it('offers today’s lesson after the summary', async () => {
      await open({ mine: mine({ today: { id: 'l1', title: 'Introductions' } }) });
      await toStep4();
      await next();
      await next();
      expect(text(el.querySelector('a[href="/today"]'))).toBe('Học bài này');
      expect(el.querySelector('a[href="/lessons/l1/read?review=1"]')).toBeNull();
    });

    it('offers reread, relisten and the writing for a finished lesson', async () => {
      await open({
        mine: mine({
          completed: [
            {
              id: 'l1',
              title: 'Introductions',
              topicName: 'Làm quen',
              completedAt: '2026-09-29T10:00:00Z',
            },
          ],
        }),
      });
      await toStep4();
      await next();
      await next();
      expect(text(el.querySelector('a[href="/lessons/l1/read?review=1"]'))).toBe('Đọc lại');
      expect(text(el.querySelector('a[href="/lessons/l1/listen?review=1"]'))).toBe('Nghe lại');
      expect(text(el.querySelector('a[href="/lessons/l1/write?review=1"]'))).toBe('Bài viết');
      expect(el.querySelector('a[href="/today"]')).toBeNull();
    });

    it('leaves out summary lines for parts the lesson does not have', async () => {
      await open({ practice: noPractice });
      await toStep4();
      await next();
      expect(el.querySelector('.scores')).toBeNull();
      expect(text(el.querySelector('#summary-heading'))).toBe('Hoàn thành phần luyện tập');
    });

    it('Làm lại goes back to step 1 with everything cleared and the banks shuffled again', async () => {
      const random = vi.spyOn(Math, 'random');
      random.mockReturnValue(0.1);
      await open();
      await next();
      await next();
      await fillAll(['name', 'meet']);
      await next();
      await build(['My', 'name', 'is', 'Minh.']);
      await next();
      await next();
      expect(Array.from(el.querySelectorAll('.scores li')).map((li) => text(li))).toEqual([
        'Điền đúng 2/2 ô',
        'Dịch đúng 1/2 câu',
      ]);

      random.mockReturnValue(0.9);
      button('Làm lại')!.click();
      await fixture.whenStable();
      expect(progress().getAttribute('aria-valuenow')).toBe('1');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      await next();
      await next();
      expect(el.querySelector('lu-fill-step [role="status"]')?.textContent?.trim()).toBe('');
      expect(el.querySelectorAll('lu-fill-step .blank.filled').length).toBe(0);
      await next();
      await next();
      await next();
      expect(Array.from(el.querySelectorAll('.scores li')).map((li) => text(li))).toEqual([
        'Điền đúng 0/2 ô',
        'Dịch đúng 0/2 câu',
      ]);
    });

    it('shuffles the word bank with a new seed on Làm lại', async () => {
      const random = vi.spyOn(Math, 'random');
      random.mockReturnValue(0.1);
      await open({
        practice: {
          ...practice,
          fill: { ...practice.fill!, wordBank: ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'] },
        },
      });
      const bank = () =>
        Array.from(
          el.querySelectorAll('lu-fill-step ul[aria-labelledby="fill-bank-heading"] .tile'),
        ).map((b) => text(b));
      await next();
      await next();
      const first = bank();
      expect([...first].sort()).toEqual(['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']);
      await next();
      await next();
      await next();
      random.mockReturnValue(0.7);
      button('Làm lại')!.click();
      await fixture.whenStable();
      await next();
      await next();
      expect(bank()).not.toEqual(first);
    });
  });
});
