import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component, input, output, Type } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import {
  ActivatedRoute,
  convertToParamMap,
  ParamMap,
  provideRouter,
  Router,
} from '@angular/router';
import { BehaviorSubject } from 'rxjs';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { PracticeView } from '../../../core/models/practice';
import { ReadingLesson } from '../../../core/models/reading';
import { LessonStudy } from '../../../core/models/study';
import { LessonVocabulary } from '../../../core/models/vocab';
import { Listening } from '../listening/listening';
import { Reading } from '../reading/reading';
import { Writing } from '../writing/writing';
import { DialogueStep } from './dialogue-step/dialogue-step';
import { FillStep } from './fill-step/fill-step';
import { LessonDetail, SPEAKING_PRACTICE } from './lesson-detail';
import { SpeakStep } from './speak-step/speak-step';
import { TranslateStep } from './translate-step/translate-step';
import { VocabStep } from './vocab-step/vocab-step';

@Component({ selector: 'lu-reading', template: '' })
class ReadingStub {
  readonly lessonId = input('');
  readonly startSentence = input<number | null>(null);
  readonly mode = input<string | null>(null);
  readonly number = input(0);
  readonly canBack = input(false);
  readonly laterLabel = input('');
  readonly doneLabel = input('');
  readonly nextLabel = input('');
  readonly completed = output<void>();
  readonly position = output<number>();
  readonly back = output<void>();
  readonly leave = output<void>();
}

@Component({ selector: 'lu-listening', template: '' })
class ListeningStub {
  readonly lessonId = input('');
  readonly startSentence = input<number | null>(null);
  readonly mode = input<string | null>(null);
  readonly laterLabel = input('');
  readonly nextLabel = input('');
  readonly completed = output<void>();
  readonly position = output<number>();
  readonly leave = output<void>();
}

@Component({ selector: 'lu-writing', template: '' })
class WritingStub {
  readonly lessonId = input('');
  readonly mode = input<string | null>(null);
  readonly completed = output<void>();
  readonly skipped = output<void>();
}

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

/** A lesson the learner is not studying (an admin's view, or another topic): the practice only. */
const other = (over: Partial<LessonStudy> = {}): LessonStudy => ({
  status: 'other',
  steps: {},
  currentStep: '',
  sentenceIndex: 0,
  next: null,
  goal: null,
  goalCompleted: false,
  streak: 3,
  ...over,
});

/** The lesson being studied, before any of its steps. */
const studying = (over: Partial<LessonStudy> = {}): LessonStudy =>
  other({
    status: 'studying',
    steps: { read: 'current', listen: 'locked', write: 'locked' },
    currentStep: 'read',
    ...over,
  });

interface Options {
  status?: number;
  errorCode?: string;
  vocabulary?: LessonVocabulary;
  practice?: PracticeView | 'error';
  study?: LessonStudy | 'error';
  /** With the Speaking step (off by default: the other specs count the steps without it). */
  speaking?: boolean;
  /** Stay on the overview a lesson being learnt opens on (by default the tests start the lesson). */
  overview?: boolean;
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
  const nextButton = () => el.querySelector<HTMLButtonElement>('.next-bar .next');
  /**
   * On to the next step the way the learner does: a practice step's own finishing button, else the
   * bar of a step done (read only), else "Để sau" on a lesson step still to do.
   */
  const next = async () => {
    const finish =
      child(VocabStep)?.done ??
      child(DialogueStep)?.finished ??
      child(FillStep)?.done ??
      child(TranslateStep)?.done ??
      child(SpeakStep)?.done ??
      child(ReadingStub)?.leave ??
      child(ListeningStub)?.leave;
    if (finish) {
      finish.emit();
    } else {
      (nextButton() ?? el.querySelector<HTMLButtonElement>('.leave'))!.click();
    }
    await fixture.whenStable();
  };
  /** Opens the lesson's overview with "i". */
  const openInfo = async () => {
    if (!el.querySelector('#lesson-overview')) {
      el.querySelector<HTMLButtonElement>('.info-btn')!.click();
      await fixture.whenStable();
    }
  };
  /** A step in the overview, by its name. */
  const stepButton = (name: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.info-step')).find((b) =>
      text(b).includes(name),
    )!;
  /** Opens a Luyện thêm practice from the overview. */
  const openExtra = async (name: string) => {
    await openInfo();
    Array.from(el.querySelectorAll<HTMLButtonElement>('.info-extras button'))
      .find((b) => text(b) === name)!
      .click();
    await fixture.whenStable();
  };
  const child = <T>(type: Type<T>) =>
    fixture.debugElement.query(By.directive(type))?.componentInstance as T | undefined;
  let params: BehaviorSubject<ParamMap>;
  let router: Router;

  /** Answers the requests that load lesson `id`. */
  const load = async (id: string, options: Options = {}) => {
    if (options.status) {
      http
        .expectOne(`/lessons/${id}`)
        .flush(
          { error: options.errorCode ?? 'x', message: 'x' },
          { status: options.status, statusText: 'Error' },
        );
      // The other requests were cancelled with the lesson's error.
      expect(http.match(() => true).every((r) => r.cancelled)).toBe(true);
      await fixture.whenStable();
      return;
    }
    http.expectOne(`/lessons/${id}/vocabulary`).flush(options.vocabulary ?? vocabulary);
    const practiceReq = http.expectOne(`/lessons/${id}/practice`);
    if (options.practice === 'error') {
      practiceReq.flush({ error: 'x' }, { status: 500, statusText: 'Error' });
    } else {
      practiceReq.flush(options.practice ?? practice);
    }
    const studyReq = http.expectOne(`/lessons/${id}/study`);
    if (options.study === 'error') {
      studyReq.flush('down', { status: 500, statusText: 'Error' });
    } else {
      studyReq.flush(options.study ?? other());
    }
    http.expectOne(`/lessons/${id}`).flush({ lesson: { ...lesson, id } });
    await fixture.whenStable();
    if (!options.overview && el.querySelector('#lesson-overview')) {
      nextButton()!.click();
      await fixture.whenStable();
    }
  };

  const open = async (options: Options = {}) => {
    speech = new FakeSpeech();
    params = new BehaviorSubject(convertToParamMap({ id: 'l1' }));
    TestBed.overrideComponent(LessonDetail, {
      remove: { imports: [Reading, Listening, Writing] },
      add: { imports: [ReadingStub, ListeningStub, WritingStub] },
    });
    TestBed.configureTestingModule({
      imports: [LessonDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        { provide: ActivatedRoute, useValue: { paramMap: params } },
        { provide: SPEAKING_PRACTICE, useValue: options.speaking ?? false },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(LessonDetail);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    await load('l1', options);
  };

  beforeEach(() => localStorage.clear());

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
    localStorage.clear();
  });

  describe('header', () => {
    it('shows the close link, progress, streak, lesson number, title, objective and level', async () => {
      await open();
      expect(el.querySelector('a[aria-label="Đóng, về danh sách bài"]')?.getAttribute('href')).toBe(
        '/lessons/levels/A1',
      );
      expect(progress().getAttribute('aria-valuenow')).toBe('1');
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      expect(text(el.querySelector('.streak'))).toBe('3 ngày');
      expect(text(el.querySelector('h1'))).toBe('Bài 3 Introductions');
      // The details stay behind "i" so the steps get the screen.
      expect(el.querySelector('.objective')).toBeNull();
      const info = el.querySelector<HTMLButtonElement>('.info-btn')!;
      expect(info.getAttribute('aria-pressed')).toBe('false');
      await openInfo();
      expect(info.getAttribute('aria-pressed')).toBe('true');
      expect(text(el.querySelector('.estimate'))).toBe('4 bước · khoảng 11 phút');
      expect(text(el.querySelector('.objective'))).toBe(
        'Mục tiêu: Bạn có thể chào hỏi và giới thiệu bản thân.',
      );
      expect(text(el.querySelector('.level'))).toBe('Mức độ: Cơ bản');
    });

    it('hides the lesson number, objective and streak when they are missing', async () => {
      await open({ practice: noPractice, study: 'error' });
      expect(text(el.querySelector('h1'))).toBe('Introductions');
      expect(el.querySelector('.objective')).toBeNull();
      expect(el.querySelector('.streak')).toBeNull();
    });

    it('explains a locked lesson and links back to the list', async () => {
      await open({ status: 403, errorCode: 'lesson_locked' });
      expect(text(el.querySelector('[role="alert"]'))).toBe('Bài này sẽ mở khi tới lượt.');
      expect(el.querySelector('a.btn[href="/lessons"]')).not.toBeNull();
      expect(el.querySelector('.reading-toggle')).toBeNull();
    });

    it('says when the lesson does not exist', async () => {
      await open({ status: 404 });
      expect(text(el.querySelector('[role="alert"]'))).toBe('Không tìm thấy bài học.');
    });
  });

  describe('learning a completed lesson again', () => {
    const completed = () =>
      other({
        status: 'completed',
        steps: { read: 'done', listen: 'done', write: 'done' },
        currentStep: 'done',
      });
    const confirm = async () => {
      await openInfo();
      button('Học lại bài')!.click();
      await fixture.whenStable();
      expect(text(el.querySelector('lu-confirm-dialog'))).toContain('Học lại bài này?');
      el.querySelector<HTMLButtonElement>('lu-confirm-dialog .btn-primary')!.click();
      await fixture.whenStable();
    };
    const deleted = (url: string) => http.expectOne((r) => r.method === 'DELETE' && r.url === url);

    it('forgets the answers and dictation, then goes through every step without recording progress', async () => {
      await open({ study: completed() });
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      await confirm();
      deleted('/lessons/l1/answers').flush(null, { status: 204, statusText: 'No Content' });
      deleted('/lessons/l1/dictation').flush(null, { status: 204, statusText: 'No Content' });
      await fixture.whenStable();
      await new Promise((r) => setTimeout(r));
      await fixture.whenStable();

      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      expect(text(el.querySelector('.relearn-note'))).toContain('Đang học lại bài');
      expect(button('Học lại bài')).toBeUndefined();
      // Learning again starts from the overview.
      expect(el.querySelector('#lesson-overview')).not.toBeNull();
      expect(text(nextButton())).toBe('Bắt đầu: Từ vựng');
      await next(); // the overview → words
      await next(); // words → Đọc, done again
      expect(child(ReadingStub)!.mode()).toBe('study');
      child(ReadingStub)!.completed.emit();
      await fixture.whenStable();
      http.expectNone((r) => r.url.includes('/steps/'));
      expect(child(ListeningStub)!.mode()).toBe('study');
      child(ListeningStub)!.completed.emit();
      await fixture.whenStable();
      http.expectNone((r) => r.url.includes('/steps/'));
      // The practice is Luyện thêm: Viết comes right after, its writing only shown again.
      expect(child(WritingStub)!.mode()).toBe('review');
    });

    it('stays as it was when the results cannot be cleared', async () => {
      await open({ study: completed() });
      await confirm();
      deleted('/lessons/l1/answers').flush('down', { status: 500, statusText: 'Error' });
      http.match((r) => r.method === 'DELETE');
      await fixture.whenStable();
      await new Promise((r) => setTimeout(r));
      await fixture.whenStable();
      expect(text(el.querySelector('[role="alert"]'))).toBe('Chưa học lại được, vui lòng thử lại.');
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
    });

    it('is not offered on a lesson that is not completed', async () => {
      await open();
      expect(button('Học lại bài')).toBeUndefined();
    });
  });

  describe('speaking step', () => {
    it('comes right after Nghe in the lesson being studied, with the dialogue turns', async () => {
      localStorage.setItem('luna.lesson-step.l1', 'speak');
      await open({ study: studying(), speaking: true });
      expect(text(el.querySelector('#speak-heading'))).toBe('4. Luyện nói');
      expect(text(el.querySelector('.progress-text'))).toBe('4/5');
      expect(text(el.querySelector('lu-speak-step .sentence'))).toBe('Hi, my name is Minh.');
    });

    it('comes last in another lesson', async () => {
      await open({ speaking: true });
      expect(text(el.querySelector('.progress-text'))).toBe('1/5');
    });
  });

  describe('steps', () => {
    it('starts on step 1 with the words and their examples', async () => {
      await open();
      expect(text(el.querySelector('#vocab-heading'))).toBe('1. Từ vựng – Chủ đề: Làm quen');
      expect(text(el.querySelector('.word .lemma'))).toBe('name');
      expect(text(el.querySelector('.example .sentence'))).toBe("What's your name?");
      // The card has its own Tiếp theo: no bar at the bottom.
      expect(el.querySelector('.next-bar')).toBeNull();
    });

    it('goes through the words on the card, then to step 2', async () => {
      await open();
      const cardNext = () => el.querySelector<HTMLButtonElement>('lu-vocab-step .next-word')!;
      cardNext().click();
      await fixture.whenStable();
      expect(text(el.querySelector('.word .lemma'))).toBe('meet');
      cardNext().click();
      await fixture.whenStable();
      expect(progress().getAttribute('aria-valuenow')).toBe('2');
      expect(el.querySelector('lu-dialogue-step')).not.toBeNull();
    });

    it('Next moves through the 4 steps and updates the progress', async () => {
      await open();
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('2');
      expect(text(el.querySelector('.progress-text'))).toBe('2/4');
      expect(text(el.querySelector('#dialogue-heading'))).toBe('2. Hội thoại');
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('3');
      expect(el.querySelector('lu-fill-step')).not.toBeNull();
      await next();
      expect(progress().getAttribute('aria-valuenow')).toBe('4');
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 1/2');
    });

    it('puts ← and the step\'s main button in one bar; the overview lists every step', async () => {
      await open();
      await next();
      // The practice step's own buttons are the bar: no second one.
      expect(el.querySelector('.next-bar')).toBeNull();
      const bar = el.querySelector('lu-dialogue-step .step-nav')!;
      expect(bar.querySelector('[aria-label="Bước trước"]')).not.toBeNull();
      expect(text(bar.querySelector('.finish'))).toBe('Sang bước Điền ô trống');
      (bar.querySelector('[aria-label="Bước trước"]') as HTMLButtonElement).click();
      await fixture.whenStable();
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      await next();

      await openInfo();
      expect(Array.from(el.querySelectorAll('.info-step')).map((b) => text(b))).toEqual([
        'Từ vựng (đã qua)',
        '2 Hội thoại',
        '3 Điền ô trống',
        '4 Dịch câu',
      ]);
      expect(stepButton('Hội thoại').getAttribute('aria-current')).toBe('step');
      stepButton('Từ vựng').click();
      await fixture.whenStable();
      expect(el.querySelector('#lesson-overview')).toBeNull();
      expect(progress().getAttribute('aria-valuenow')).toBe('1');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
    });

    it('names each step\'s last button after the next step', async () => {
      await open();
      await next();
      expect(text(el.querySelector('lu-dialogue-step .finish'))).toBe('Sang bước Điền ô trống');
    });

    it('the translation card moves between its sentences', async () => {
      await open();
      await next();
      await next();
      await next();
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 1/2');
      button('Bỏ qua câu này')!.click();
      await fixture.whenStable();
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 2/2');
    });

    it('reads a word with the browser voice', async () => {
      await open();
      el.querySelector<HTMLButtonElement>('button[aria-label="Nghe từ name"]')!.click();
      await fixture.whenStable();
      expect(speech.last()).toMatchObject({ text: 'name', rate: 1 });
    });

    it('without practice: only the words step, numbered 1/1, and no empty steps', async () => {
      await open({ practice: 'error' });
      expect(el.querySelector('.example')).toBeNull();
      expect(progress().getAttribute('aria-valuemax')).toBe('1');
      expect(text(el.querySelector('.progress-text'))).toBe('1/1');
      const cardNext = () => el.querySelector<HTMLButtonElement>('lu-vocab-step .next-word')!;
      expect(text(cardNext())).toBe('Tiếp theo');
      cardNext().click();
      await fixture.whenStable();
      expect(text(cardNext())).toBe('Hoàn thành');
      expect(el.textContent).not.toContain('chưa có phần luyện tập');
    });

    it('only shows the steps that have content, numbered in order', async () => {
      await open({ practice: { ...noPractice, translations: practice.translations } });
      expect(text(el.querySelector('.progress-text'))).toBe('1/2');
      await next();
      expect(el.querySelector('lu-dialogue-step')).toBeNull();
      expect(el.querySelector('lu-fill-step')).toBeNull();
      expect(text(el.querySelector('lu-translate-step .step-title'))).toBe(
        '2. Dịch câu sang tiếng Anh',
      );
      expect(text(el.querySelector('.progress-text'))).toBe('2/2');
    });

    it('without words or practice: no Bài học tab, only the text', async () => {
      await open({ vocabulary: { available: false, items: [] }, practice: noPractice });
      expect(el.querySelector('.reading-toggle')).toBeNull();
      expect(el.querySelector('#panel-lesson')).toBeNull();
      expect(el.querySelector('[role="progressbar"]')).toBeNull();
      expect(el.querySelector('.next-bar')).toBeNull();
      expect(el.querySelector<HTMLElement>('#panel-reading')!.hidden).toBe(false);
      expect(el.querySelectorAll('#panel-reading .reading p').length).toBe(2);
    });
  });

  describe('tabs', () => {
    it('Bài đọc shows the text and grammar note; back to Bài học keeps the step', async () => {
      await open();
      await next();
      const reading = el.querySelector<HTMLButtonElement>('.reading-toggle')!;
      expect(text(reading)).toBe('Bài đọc');
      reading.click();
      await fixture.whenStable();
      expect(reading.getAttribute('aria-pressed')).toBe('true');
      expect(text(reading)).toBe('Bài học');
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

      // Tapping a sentence reads it aloud and highlights it; tapping it again while read stops it.
      const sentences = Array.from(el.querySelectorAll<HTMLButtonElement>('#panel-reading .read-sentence'));
      expect(sentences.map((b) => text(b))).toEqual([
        'Hello, my name is Minh.',
        'Nice to meet you.',
        'I am from Vietnam.',
      ]);
      sentences[1].click();
      await fixture.whenStable();
      expect(speech.last()).toMatchObject({ text: 'Nice to meet you.' });
      speech.playing.set('Nice to meet you.');
      await fixture.whenStable();
      expect(sentences[1].classList).toContain('reading-now');
      expect(sentences[1].getAttribute('aria-pressed')).toBe('true');
      expect(sentences[0].classList).not.toContain('reading-now');
      const stops = speech.stops;
      sentences[1].click();
      await fixture.whenStable();
      expect(speech.stops).toBe(stops + 1);
      expect(sentences[1].classList).not.toContain('reading-now');

      reading.click();
      await fixture.whenStable();
      expect(el.querySelector<HTMLElement>('#panel-lesson')!.hidden).toBe(false);
      expect(progress().getAttribute('aria-valuenow')).toBe('2');
      expect(el.querySelector('lu-dialogue-step')).not.toBeNull();
    });
  });

  describe('step 4 and summary', () => {
    const toStep4 = async () => {
      await next();
      await next();
      await next();
    };

    /** Fills and checks the fill-in lines one by one (one blank per line here). */
    const fillAll = async (words: string[]) => {
      for (const [i, w] of words.entries()) {
        // The blanks are lines filled from the word bank.
        Array.from(el.querySelectorAll<HTMLButtonElement>('lu-fill-step .tiles .tile'))
          .find((b) => text(b).startsWith(w) && !b.disabled)!
          .click();
        await fixture.whenStable();
        button('Kiểm tra')!.click();
        await fixture.whenStable();
        if (i < words.length - 1) {
          button('Câu tiếp theo')!.click();
          await fixture.whenStable();
        }
      }
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

    /** The summary's rows: name and what was done. */
    const items = () =>
      Array.from(el.querySelectorAll('lu-practice-summary .item')).map(
        (li) => `${text(li.querySelector('.name'))}: ${text(li.querySelector('.result'))}`,
      );

    it('goes through the sentences, then the summary says what was done in each step', async () => {
      await open();
      await next();
      await next();
      await fillAll(['name', 'from']);
      await next();
      await build(['My', 'name', 'is', 'Minh.']);
      expect(text(el.querySelector('lu-translate-step [role="status"]'))).toBe('Chính xác!');
      button('Câu tiếp theo')!.click();
      await fixture.whenStable();
      expect(progress().getAttribute('aria-valuenow')).toBe('4');
      expect(text(el.querySelector('lu-translate-step'))).toContain('Câu 2/2');
      await next(); // the second sentence not checked: not counted as wrong
      expect(text(el.querySelector('#summary-heading'))).toBe('Xong phần luyện tập');
      expect(items()).toEqual([
        'Hội thoại: Đã nghe hết',
        'Điền ô trống: Đúng 1/2 ô',
        'Dịch câu: Đã làm 1/2 câu',
      ]);
      expect(el.textContent).not.toContain('chỉ giữ trên trang này');
      expect(el.querySelector('.next-bar')).toBeNull();
      // Results stay on the page: nothing was sent.
      http.expectNone(() => true);
    });

    it('offers reread, relisten and the writing for a finished lesson', async () => {
      await open({
        study: other({
          status: 'completed',
          steps: { read: 'done', listen: 'done', write: 'done' },
          currentStep: 'done',
        }),
      });
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      await toStep4();
      await next();
      expect(text(el.querySelector('a[href="/lessons/l1/read?review=1"]'))).toBe('Đọc lại');
      expect(text(el.querySelector('a[href="/lessons/l1/listen?review=1"]'))).toBe('Nghe lại');
      expect(text(el.querySelector('a[href="/lessons/l1/write?review=1"]'))).toBe('Bài viết');
    });

    it('leaves out summary lines for parts the lesson does not have', async () => {
      await open({ practice: noPractice });
      await next(); // the words step is the only one
      await fixture.whenStable();
      expect(el.querySelector('lu-practice-list')).toBeNull();
      expect(text(el.querySelector('#summary-heading'))).toBe('Xong phần luyện tập');
    });

    it('says "Chưa làm" for a step that was skipped, with a way into it', async () => {
      await open();
      await next();
      await next();
      await next(); // fill skipped
      await next(); // translation skipped
      expect(items()).toEqual([
        'Hội thoại: Đã nghe hết',
        'Điền ô trống: Chưa làm',
        'Dịch câu: Chưa làm',
      ]);
      // The whole row is the button.
      el.querySelector<HTMLButtonElement>('lu-practice-summary button[aria-label="Làm Điền ô trống: chưa làm"]')!.click();
      await fixture.whenStable();
      expect(el.querySelector('lu-fill-step')).not.toBeNull();
      expect(el.querySelector('lu-practice-summary')).toBeNull();
    });

    it('opens a step from anywhere on its row of the summary, its name included', async () => {
      await open();
      await next();
      await next();
      await next();
      await next();
      const row = Array.from(el.querySelectorAll('lu-practice-summary .item')).find((r) =>
        text(r).includes('Dịch câu'),
      )!;
      (row.querySelector('.name') as HTMLElement).click();
      await fixture.whenStable();
      expect(el.querySelector('lu-translate-step')).not.toBeNull();
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
      expect(items()).toEqual([
        'Hội thoại: Đã nghe hết',
        'Điền ô trống: Đúng 2/2 ô',
        'Dịch câu: Đã làm 1/2 câu',
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
      expect(items()).toEqual([
        'Hội thoại: Đã nghe hết',
        'Điền ô trống: Chưa làm',
        'Dịch câu: Chưa làm',
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
      random.mockReturnValue(0.7);
      button('Làm lại')!.click();
      await fixture.whenStable();
      await next();
      await next();
      expect(bank()).not.toEqual(first);
    });
  });

  describe('the lesson being studied', () => {
    const reply = async (url: string, body: LessonStudy) => {
      const req = http.expectOne(url);
      expect(req.request.method).toBe('POST');
      req.flush(body);
      await fixture.whenStable();
    };
    /** The cards due now, asked once the lesson is done. */
    const flushDue = async (total: number) => {
      http
        .expectOne((r) => r.url === '/vocab/review/due')
        .flush({ cards: [], total, nextDue: null });
      await fixture.whenStable();
    };
    /** Studying lesson l1, up to the first step of the practice (Đọc and Nghe done). */
    const toPractice = async () => {
      await open({ study: studying() });
      await next();
      child(ReadingStub)!.completed.emit();
      await reply(
        '/lessons/l1/steps/read/complete',
        studying({
          steps: { read: 'done', listen: 'current', write: 'locked' },
          currentStep: 'listen',
        }),
      );
      child(ListeningStub)!.completed.emit();
      await reply(
        '/lessons/l1/steps/listen/complete',
        studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      );
    };

    it('puts the words got wrong in the practice into the review queue, once each', async () => {
      await toPractice();
      await openExtra('Điền ô trống');
      child(FillStep)!.missed.emit(['name']);
      const req = http.expectOne('/vocab/practice-misses');
      expect(req.request.body).toEqual({ lessonId: 'l1', words: ['name'] });
      req.flush({ added: 1, rescheduled: 0 });
      await fixture.whenStable();
      expect(text(el.querySelector('.miss-note'))).toBe('Đã đưa 1 từ làm sai vào lịch ôn.');

      // The same word again sends nothing.
      child(FillStep)!.missed.emit(['name']);
      await next(); // back to the lesson
      await openExtra('Dịch câu'); // sentence 1 ("My name is Minh.")
      child(TranslateStep)!.checked.emit(false);
      http.expectNone('/vocab/practice-misses');

      // A wrong sentence sends the lesson words it contains; a right one sends nothing.
      child(TranslateStep)!.go.emit(1); // sentence 2 ("Nice to meet you.")
      await fixture.whenStable();
      child(TranslateStep)!.checked.emit(true);
      http.expectNone('/vocab/practice-misses');
      child(TranslateStep)!.checked.emit(false);
      const second = http.expectOne('/vocab/practice-misses');
      expect(second.request.body).toEqual({ lessonId: 'l1', words: ['meet'] });
      second.flush({ added: 0, rescheduled: 1 });
      await fixture.whenStable();
      expect(text(el.querySelector('.miss-note'))).toBe('Đã đưa 2 từ làm sai vào lịch ôn.');
    });

    it('ignores a failure of the review queue', async () => {
      await toPractice();
      await openExtra('Điền ô trống');
      child(FillStep)!.missed.emit(['name']);
      http
        .expectOne('/vocab/practice-misses')
        .flush('down', { status: 500, statusText: 'Error' });
      await fixture.whenStable();
      expect(el.querySelector('.miss-note')).toBeNull();
      expect(el.querySelector('[role="alert"]')).toBeNull();
    });

    it('sends nothing from a lesson that is not being studied', async () => {
      await open(); // another lesson: the practice only
      await next();
      await next(); // fill
      child(FillStep)!.missed.emit(['name']);
      http.expectNone('/vocab/practice-misses');
    });

    it('opens on the overview: the steps, the time, Luyện thêm, then Bắt đầu', async () => {
      await open({ study: studying(), overview: true });
      const overview = el.querySelector('#lesson-overview')!;
      expect(text(overview.querySelector('.estimate'))).toBe('4 bước · khoảng 15 phút');
      expect(Array.from(overview.querySelectorAll('.info-step')).map((b) => text(b))).toEqual([
        '1 Từ vựng',
        '2 Đọc',
        '3 Nghe',
        '4 Viết',
      ]);
      // Luyện thêm is shown but locked until Nghe is done: before that it gives the answers away.
      expect(text(overview.querySelector('.info-extras'))).toContain('Luyện thêm (tuỳ chọn)');
      expect(text(overview.querySelector('.extras-note'))).toBe('Mở sau khi xong bước Nghe');
      const chips = Array.from(overview.querySelectorAll<HTMLButtonElement>('.info-extras .chip-btn'));
      expect(chips.map((b) => b.disabled)).toEqual([true, true, true]);
      expect(chips[0].getAttribute('aria-describedby')).toBe('extras-note');
      expect(el.querySelector('lu-vocab-step')).toBeNull();
      expect(el.querySelector('.next-bar .back')).toBeNull();
      expect(text(nextButton())).toBe('Bắt đầu: Từ vựng');
      await next();
      expect(el.querySelector('#lesson-overview')).toBeNull();
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      // From the first step, ← goes back to the overview.
      el.querySelector<HTMLButtonElement>('.next-bar .back')!.click();
      await fixture.whenStable();
      expect(el.querySelector('#lesson-overview')).not.toBeNull();
    });

    it('says Tiếp tục on the overview of a lesson already started', async () => {
      await open({
        study: studying({ steps: { read: 'done', listen: 'current', write: 'locked' }, currentStep: 'listen' }),
        overview: true,
      });
      expect(text(nextButton())).toBe('Tiếp tục: Nghe');
      expect(stepButton('Đọc').classList).toContain('is-done');
    });

    it('goes words, Đọc, Nghe, then Viết: five main steps at most, the practice kept as Luyện thêm', async () => {
      await open({ study: studying() });
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      await next();
      expect(text(el.querySelector('.progress-text'))).toBe('2/4');
      const reading = child(ReadingStub)!;
      expect(reading.mode()).toBe('study');
      expect(reading.lessonId()).toBe('l1');
      // Đọc has its own bar: ← and a quiet way to leave it for later.
      expect(el.querySelector('.next-bar')).toBeNull();
      expect(reading.number()).toBe(2);
      expect(reading.canBack()).toBe(true);
      expect(reading.laterLabel()).toBe('Để sau, sang bước Nghe');
      // The questions' result goes on with the same step's name.
      expect(reading.doneLabel()).toBe('Sang bước Nghe');

      reading.completed.emit();
      await reply(
        '/lessons/l1/steps/read/complete',
        studying({
          steps: { read: 'done', listen: 'current', write: 'locked' },
          currentStep: 'listen',
        }),
      );
      expect(text(el.querySelector('.progress-text'))).toBe('3/4');
      expect(child(ListeningStub)!.mode()).toBe('study');

      // The details go back to Đọc, shown again read only, with the bar to go on.
      await openInfo();
      stepButton('Đọc').click();
      await fixture.whenStable();
      expect(child(ReadingStub)!.mode()).toBe('review');
      expect(child(ReadingStub)!.nextLabel()).toBe('Sang bước Nghe');
      await next();
      child(ListeningStub)!.completed.emit();
      await reply(
        '/lessons/l1/steps/listen/complete',
        studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      );
      // Viết comes right after Nghe; the practice on the text is Luyện thêm.
      expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      expect(child(WritingStub)!.mode()).toBe('study');
      expect(el.querySelector('lu-practice-summary')).toBeNull();
      await openInfo();
      expect(Array.from(el.querySelectorAll('.info-extras button')).map((b) => text(b))).toEqual([
        'Hội thoại',
        'Điền ô trống',
        'Dịch câu',
      ]);

      // A Luyện thêm practice opens over Viết, then goes back to it.
      await openExtra('Hội thoại');
      expect(text(el.querySelector('#dialogue-heading'))).toBe('Hội thoại');
      expect(text(el.querySelector('.extra-label'))).toBe('Luyện thêm · tuỳ chọn');
      expect(text(el.querySelector('lu-dialogue-step .finish'))).toBe('Xong, về bài học');
      expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      await next();
      expect(child(DialogueStep)).toBeUndefined();
      expect(child(WritingStub)!.mode()).toBe('study');
    });

    it('opens Luyện thêm once Nghe is done, and offers it on Viết', async () => {
      await toPractice();
      expect(child(WritingStub)!.mode()).toBe('study');
      const offer = el.querySelector('.extras-offer')!;
      expect(text(offer)).toContain('Muốn luyện thêm trước khi viết?');
      Array.from(offer.querySelectorAll<HTMLButtonElement>('button'))
        .find((b) => text(b) === 'Điền ô trống')!
        .click();
      await fixture.whenStable();
      expect(child(FillStep)).toBeDefined();
      expect(el.querySelector('.extras-offer')).toBeNull();
      await openInfo();
      expect(el.querySelector('.extras-note')).toBeNull();
      expect(
        Array.from(el.querySelectorAll<HTMLButtonElement>('.info-extras .chip-btn')).every((b) => !b.disabled),
      ).toBe(true);
    });

    it('opens Luyện thêm from the summary reached with Nghe left for later', async () => {
      // Viết done, Đọc and Nghe left for later: the last step's Hoàn thành shows the summary.
      await open({ study: studying({ steps: { read: 'current', listen: 'locked', write: 'done' }, currentStep: 'read' }) });
      el.querySelector<HTMLButtonElement>('.info-btn')!.click();
      await fixture.whenStable();
      Array.from(el.querySelectorAll<HTMLButtonElement>('.info-step')).find((b) => text(b).includes('Viết'))!.click();
      await fixture.whenStable();
      nextButton()!.click();
      await fixture.whenStable();
      // The lesson is not done: the summary says what was left for later.
      expect(Array.from(el.querySelectorAll('.left-behind button')).map((b) => text(b))).toEqual([
        'Làm bước Đọc',
        'Làm bước Nghe',
      ]);
      const row = el.querySelector<HTMLButtonElement>('lu-practice-summary button[aria-label="Làm Hội thoại: chưa làm"]')!;
      row.click();
      await fixture.whenStable();
      expect(el.querySelector('lu-dialogue-step')).not.toBeNull();
    });

    it('goes back from Luyện thêm with Về bài học', async () => {
      await toPractice();
      await openExtra('Dịch câu');
      expect(child(TranslateStep)).toBeDefined();
      button('Về bài học')!.click();
      await fixture.whenStable();
      expect(child(TranslateStep)).toBeUndefined();
      expect(child(WritingStub)).toBeDefined();
    });

    it('leaves Đọc for later, and points back to it from Viết', async () => {
      await open({ study: studying() });
      await next(); // words → Đọc
      await next(); // Để sau: nothing is recorded
      expect(child(ListeningStub)!.mode()).toBe('study');
      http.expectNone(() => true);
      await next(); // Viết
      expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      // Viết has its own buttons: nothing to skip to, ← only.
      expect(nextButton()).toBeNull();
      expect(el.querySelector('.next-bar .back')).not.toBeNull();
      expect(text(el.querySelector('.left-behind'))).toContain('Còn để lại');
      const steps = Array.from(el.querySelectorAll('.left-behind button')).map((b) => text(b));
      expect(steps).toEqual(['Làm bước Đọc', 'Làm bước Nghe']);
      button('Làm bước Đọc')!.click();
      await fixture.whenStable();
      expect(child(ReadingStub)!.mode()).toBe('study');
    });

    it('Bỏ qua the writing finishes the lesson and offers the next one', async () => {
      await open({
        study: studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      });
      // Back where the learner left off.
      expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      child(WritingStub)!.skipped.emit();
      await reply(
        '/lessons/l1/steps/write/skip',
        other({
          status: 'completed',
          steps: { read: 'done', listen: 'done', write: 'done' },
          currentStep: 'done',
          next: { id: 'l2', title: 'Family 2' },
          streak: 4,
        }),
      );
      await flushDue(0);
      expect(text(el.querySelector('#done-heading'))).toBe('Hoàn thành bài học');
      expect(text(el.querySelector('.done-card'))).toContain('4 ngày');
      const nextLesson = el.querySelector('.done-card a.btn-primary')!;
      expect(nextLesson.getAttribute('href')).toBe('/lessons/l2');
      expect(text(nextLesson)).toBe('Sang bài tiếp theo: Family 2');
      expect(el.querySelector('.review-nudge')).toBeNull();
      expect(el.querySelector('.next-bar')).toBeNull();
      // Luyện thêm stays on offer once the lesson is done.
      expect(
        Array.from(el.querySelectorAll('.done-card .item .result')).map((r) => text(r)),
      ).toEqual(['Chưa làm', 'Chưa làm', 'Chưa làm']);

      // Going to the next lesson loads it on the same page, from its start.
      params.next(convertToParamMap({ id: 'l2' }));
      await fixture.whenStable();
      await load('l2', { study: studying() });
      expect(el.querySelector('.done-card')).toBeNull();
      expect(text(el.querySelector('.progress-text'))).toBe('1/4');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
    });

    it('says when the topic has no next lesson yet', async () => {
      await open({
        study: studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      });
      child(WritingStub)!.completed.emit();
      await reply(
        '/lessons/l1/steps/write/complete',
        other({ status: 'completed', currentStep: 'done' }),
      );
      await flushDue(35);
      // The submitted writing is linked with its score.
      http.expectOne('/lessons/l1/writing').flush({
        prompt: 'Write about your family.',
        level: 'A1',
        canWrite: false,
        writing: {
          id: 'w1',
          lessonId: 'l1',
          lessonTitle: 'Family',
          prompt: 'Write about your family.',
          text: 'My family is big.',
          status: 'submitted',
          submittedAt: '2026-10-12T08:00:00Z',
          grade: { status: 'done', error: '', criteria: [], average: 3.5, overallVi: '' },
        },
      });
      await fixture.whenStable();
      const result = el.querySelector('.done-card a.writing-result')!;
      expect(result.getAttribute('href')).toBe('/writings/w1');
      expect(text(result)).toBe('Xem nhận xét bài viết (3,5/5)');
      expect(text(el.querySelector('.done-card'))).toContain('Chủ đề này chưa có bài tiếp theo');
      expect(el.querySelector('.done-card a[href="/lessons/levels/A1"]')).not.toBeNull();
      // A large backlog is said so; with no next lesson the review link carries none.
      const nudge = el.querySelector('.review-nudge')!;
      expect(text(nudge)).toContain('35 thẻ');
      expect(text(nudge)).toContain('khoảng 12 phút');
      expect(text(nudge)).toContain('tồn nhiều');
      expect(nudge.querySelector('a')!.getAttribute('href')).toBe('/vocabulary/review');
    });

    it('suggests reviewing the due cards before the next lesson, which stays one tap away', async () => {
      await open({
        study: studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      });
      child(WritingStub)!.skipped.emit();
      await reply(
        '/lessons/l1/steps/write/skip',
        other({
          status: 'completed',
          currentStep: 'done',
          next: { id: 'l2', title: 'Family 2' },
        }),
      );
      await flushDue(5);
      const nudge = el.querySelector('.review-nudge')!;
      expect(text(nudge)).toContain('5 thẻ');
      expect(text(nudge)).toContain('khoảng 2 phút');
      expect(text(nudge)).not.toContain('tồn nhiều');
      const review = nudge.querySelector('a')!;
      expect(review.getAttribute('href')).toBe('/vocabulary/review?next=l2');
      // Review is the main button now; the next lesson is a secondary one.
      const nextLesson = el.querySelector('.done-card a[href="/lessons/l2"]')!;
      expect(nextLesson.classList.contains('btn')).toBe(true);
      expect(nextLesson.classList.contains('btn-primary')).toBe(false);
    });

    it('still finishes when the number of due cards cannot be loaded', async () => {
      await open({
        study: studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      });
      child(WritingStub)!.skipped.emit();
      await reply(
        '/lessons/l1/steps/write/skip',
        other({ status: 'completed', currentStep: 'done', next: { id: 'l2', title: 'Family 2' } }),
      );
      http
        .expectOne((r) => r.url === '/vocab/review/due')
        .flush('down', { status: 500, statusText: 'Error' });
      await fixture.whenStable();
      expect(el.querySelector('.review-nudge')).toBeNull();
      expect(el.querySelector('.done-card a.btn-primary')!.getAttribute('href')).toBe('/lessons/l2');
    });

    it('goes to the congratulations at the end of the roadmap', async () => {
      await open({
        study: studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        }),
      });
      const navigate = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
      child(WritingStub)!.skipped.emit();
      await reply(
        '/lessons/l1/steps/write/skip',
        other({ status: 'completed', currentStep: 'done', goalCompleted: true }),
      );
      expect(navigate).toHaveBeenCalledWith('/goal?completed=1');
    });

    it('shows why a step cannot be completed', async () => {
      await open({
        practice: noPractice,
        vocabulary: { available: false, items: [] },
        study: studying(),
      });
      // Without practice the lesson starts on Đọc, still with Bài đọc.
      expect(text(el.querySelector('.progress-text'))).toBe('1/3');
      expect(el.querySelector('.reading-toggle')).not.toBeNull();
      child(ReadingStub)!.completed.emit();
      http
        .expectOne('/lessons/l1/steps/read/complete')
        .flush(
          { error: 'read_incomplete', message: 'Hãy trả lời hết câu hỏi hiểu bài' },
          { status: 409, statusText: 'Conflict' },
        );
      await fixture.whenStable();
      expect(text(el.querySelector('[role="alert"]'))).toBe('Hãy trả lời hết câu hỏi hiểu bài');
      expect(child(ReadingStub)).toBeDefined();
    });

    it('saves the position one second after the last move', async () => {
      await open({
        practice: noPractice,
        vocabulary: { available: false, items: [] },
        study: studying(),
      });
      vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
      try {
        const reading = child(ReadingStub)!;
        reading.position.emit(2);
        reading.position.emit(3);
        vi.advanceTimersByTime(999);
        http.expectNone('/lessons/l1/position');
        vi.advanceTimersByTime(1);
        const req = http.expectOne('/lessons/l1/position');
        expect(req.request.method).toBe('PUT');
        expect(req.request.body).toEqual({ step: 'read', sentenceIndex: 3 });
        req.flush(null, { status: 204, statusText: 'No Content' });
      } finally {
        vi.useRealTimers();
      }
    });

    describe('the step kept on this device', () => {
      const writeCurrent = () =>
        studying({
          steps: { read: 'done', listen: 'done', write: 'current' },
          currentStep: 'write',
        });

      it('is where the learner comes back to, not Viết, when they left in Luyện nói', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'speak');
        await open({ study: writeCurrent(), speaking: true });
        expect(text(el.querySelector('.progress-text'))).toBe('4/5');
        expect(el.querySelector('lu-speak-step')).not.toBeNull();
        expect(child(WritingStub)).toBeUndefined();
      });

      it('never puts the learner behind what the server already knows', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'words');
        await open({ study: writeCurrent() });
        expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      });

      it('ignores a saved step that does not exist any more', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'speaking');
        await open({ study: writeCurrent() });
        expect(text(el.querySelector('.progress-text'))).toBe('4/4');
      });

      it('ignores a saved practice step, now Luyện thêm', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'fill');
        await open({ study: writeCurrent() });
        expect(text(el.querySelector('.progress-text'))).toBe('4/4');
        expect(child(WritingStub)).toBeDefined();
      });

      it('is saved as the learner moves, and the opening does not overwrite it', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'speak');
        await open({ study: writeCurrent(), speaking: true });
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBe('speak');
        await next();
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBe('write');
      });

      it('is not kept for a lesson that is not being studied', async () => {
        await open(); // another lesson: the practice only
        await next();
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBeNull();
      });

      it('is forgotten once the lesson is done', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'write');
        await open({ study: writeCurrent() });
        child(WritingStub)!.skipped.emit();
        await reply(
          '/lessons/l1/steps/write/skip',
          other({ status: 'completed', currentStep: 'done', next: { id: 'l2', title: 'Family 2' } }),
        );
        await flushDue(0);
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBeNull();
      });
    });

    it('starts again where a saved position is', async () => {
      await open({ study: studying({ sentenceIndex: 4 }) });
      expect(text(el.querySelector('.progress-text'))).toBe('2/4');
      expect(child(ReadingStub)!.startSentence()).toBe(4);
    });
  });
});
