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
import { LessonStudy, Step, STEPS } from '../../../core/models/study';
import { LessonVocabulary } from '../../../core/models/vocab';
import { Reading } from '../reading/reading';
import { Writing } from '../writing/writing';
import { DialogueStep } from './dialogue-step/dialogue-step';
import { FillStep } from './fill-step/fill-step';
import { GrammarStep } from './grammar-step/grammar-step';
import { LessonDetail, REVIEW_OFFER, SPEAKING_PRACTICE } from './lesson-detail';
import { ListenStep } from './listen-step/listen-step';
import { SpeakLesson } from './speak-lesson/speak-lesson';
import { SpeakStep } from './speak-step/speak-step';
import { TranslateStep } from './translate-step/translate-step';
import { VocabStep } from './vocab-step/vocab-step';

@Component({ selector: 'lu-reading', template: '' })
class ReadingStub {
  readonly lessonId = input('');
  readonly startSentence = input<number | null>(null);
  readonly mode = input<string | null>(null);
  readonly guideVi = input('');
  readonly targetWords = input<string[]>([]);
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

@Component({ selector: 'lu-listen-step', template: '' })
class ListenStub {
  readonly lessonId = input('');
  readonly practice = input<unknown>(null);
  readonly mode = input<string | null>(null);
  readonly startSentence = input<number | null>(null);
  readonly number = input(0);
  readonly canBack = input(false);
  readonly laterLabel = input('');
  readonly nextLabel = input('');
  readonly back = output<void>();
  readonly leave = output<void>();
  readonly completed = output<void>();
  readonly position = output<number>();
}

@Component({ selector: 'lu-grammar-step', template: '' })
class GrammarStub {
  readonly content = input<unknown>(null);
  readonly note = input<unknown>(null);
  readonly grammarPointId = input('');
  readonly grammarTitleVi = input('');
  readonly number = input(0);
  readonly finishLabel = input('');
  readonly canBack = input(false);
  readonly back = output<void>();
  readonly done = output<void>();
}

@Component({ selector: 'lu-speak-lesson', template: '' })
class SpeakLessonStub {
  readonly lessonId = input('');
  readonly topicId = input('');
  readonly topicName = input('');
  readonly readAloud = input<unknown[]>([]);
  readonly fallbackSentences = input<string[]>([]);
  readonly questions = input<unknown[]>([]);
  readonly number = input(0);
  readonly finishLabel = input('');
  readonly canBack = input(false);
  readonly back = output<void>();
  readonly done = output<void>();
}

@Component({ selector: 'lu-writing', template: '' })
class WritingStub {
  readonly lessonId = input('');
  readonly mode = input<string | null>(null);
  readonly help = input<unknown>(null);
  readonly targetWords = input<string[]>([]);
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

/** The six steps' states: those in `done` done, `current` current, the others locked. */
const stepsOf = (done: Step[], current: Step | 'done'): LessonStudy['steps'] =>
  Object.fromEntries(
    STEPS.map((s) => [s, done.includes(s) ? 'done' : s === current ? 'current' : 'locked']),
  );

/** The lesson being studied, before any of its steps. */
const studying = (over: Partial<LessonStudy> = {}): LessonStudy =>
  other({
    status: 'studying',
    steps: stepsOf([], 'vocab'),
    currentStep: 'vocab',
    ...over,
  });

/** The lesson being studied with the steps in `done` done, on to `current`. */
const at = (done: Step[], current: Step, over: Partial<LessonStudy> = {}): LessonStudy =>
  studying({ steps: stepsOf(done, current), currentStep: current, ...over });

/** The lesson being studied with every step done but Viết. */
const writeNext = () => at(['vocab', 'listen', 'read', 'grammar', 'speak'], 'write');

/** A completed lesson: every step done. */
const completedStudy = (over: Partial<LessonStudy> = {}) =>
  other({ status: 'completed', steps: stepsOf([...STEPS], 'done'), currentStep: 'done', ...over });

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
  /** Offer to review the lesson before (off by default: it adds a request to the others). */
  reviewOffer?: boolean;
  /** Fields replacing those of the lesson. */
  lesson?: Partial<ReadingLesson>;
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
   * On to the next step the way the learner does: a practice step's own finishing button, else
   * leaving Đọc or Nghe (for later, or once done), else the bar. The words end with nothing known
   * or missed (a lesson being studied saves them: see `finishVocab`); Ngữ pháp and Nói are left to
   * each test, since a lesson being studied records them.
   */
  const next = async () => {
    const vocab = child(VocabStep);
    const finish =
      child(DialogueStep)?.finished ??
      child(FillStep)?.done ??
      child(TranslateStep)?.done ??
      child(SpeakStep)?.done ??
      child(ReadingStub)?.leave ??
      child(ListenStub)?.leave;
    if (vocab) {
      vocab.done.emit({ known: [], missed: [] });
    } else if (finish) {
      finish.emit();
    } else {
      nextButton()!.click();
    }
    await fixture.whenStable();
  };
  /** Lets the requests and promises started by a step go on. */
  const settle = async () => {
    await fixture.whenStable();
    await new Promise((r) => setTimeout(r));
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
    http.expectOne(`/lessons/${id}`).flush({ lesson: { ...lesson, ...options.lesson, id } });
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
      remove: { imports: [Reading, ListenStep, GrammarStep, SpeakLesson, Writing] },
      add: { imports: [ReadingStub, ListenStub, GrammarStub, SpeakLessonStub, WritingStub] },
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
        { provide: REVIEW_OFFER, useValue: options.reviewOffer ?? false },
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

    // Added 2026-10-20.
    it('shows the grammar point of the lesson with a link to study it', async () => {
      await open({ lesson: { grammarPointId: 'a1-to-be', grammarPointTitle: 'Động từ to be' } });
      await openInfo();
      expect(text(el.querySelector('.lesson-grammar'))).toContain('Ngữ pháp của bài: Động từ to be');
      expect(el.querySelector('.lesson-grammar a')?.getAttribute('href')).toBe('/grammar/a1-to-be');
    });

    it('hides the grammar point when the lesson has none', async () => {
      await open();
      await openInfo();
      expect(el.querySelector('.lesson-grammar')).toBeNull();
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

  describe('offering to review the lesson before', () => {
    const mine = {
      current: { id: 'l1', title: 'Introductions' },
      completed: [
        {
          id: 'l0',
          title: 'Greetings',
          topicName: 'Làm quen',
          completedAt: '2026-10-17T08:00:00Z',
        },
        { id: 'lb', title: 'Family', topicName: 'Làm quen', completedAt: '2026-10-18T08:00:00Z' },
      ],
      upcoming: [],
    };
    const dialog = () => el.querySelectorAll('lu-confirm-dialog')[1];
    const offered = async () => {
      await open({ study: studying(), overview: true, reviewOffer: true });
      http.expectOne((r) => r.url === '/lessons/mine' && r.params.get('all') === '1').flush(mine);
      await fixture.whenStable();
    };

    it('asks to review the lesson completed last when a new lesson starts', async () => {
      await offered();
      expect(text(dialog())).toContain('Ôn lại bài vừa học?');
      expect(text(dialog())).toContain('"Family"');
    });

    it('"Ôn tập" opens that lesson\'s review, which leads on to the new lesson', async () => {
      await offered();
      const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
      dialog().querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await fixture.whenStable();
      expect(navigate).toHaveBeenCalledWith(['/vocabulary/review/lessons', 'lb'], {
        queryParams: { next: 'l1' },
      });
      // Settled only once the review is done: left unfinished, the offer comes back.
      expect(localStorage.getItem('luna.review-settled.l1')).toBeNull();
    });

    it('"Bỏ qua" closes it, and it comes back while no step of the lesson is done', async () => {
      await offered();
      dialog().querySelector<HTMLButtonElement>('.btn:not(.btn-primary)')!.click();
      await fixture.whenStable();
      expect(dialog().querySelector('[role="alertdialog"]')).toBeNull();
      expect(el.querySelector('#lesson-overview')).not.toBeNull();

      // A skip may be a slip: the lesson not begun yet, it asks again.
      TestBed.resetTestingModule();
      await offered();
      expect(text(dialog())).toContain('Ôn lại bài vừa học?');
    });

    it('does not ask again once the review was done', async () => {
      localStorage.setItem('luna.review-settled.l1', '1');
      await open({ study: studying(), overview: true, reviewOffer: true });
      http.expectNone((r) => r.url === '/lessons/mine');
    });

    it('still asks when the lesson was opened before but no step of it is done', async () => {
      localStorage.setItem('luna.lesson-step.l1', 'read');
      await offered();
      expect(text(dialog())).toContain('Ôn lại bài vừa học?');
    });

    it('asks again when the review was left unfinished', async () => {
      await offered();
      vi.spyOn(router, 'navigate').mockResolvedValue(true);
      dialog().querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await fixture.whenStable();

      TestBed.resetTestingModule();
      await offered();
      expect(text(dialog())).toContain('Ôn lại bài vừa học?');
    });

    it('does not ask once the lesson has begun', async () => {
      await open({ study: at(['vocab'], 'listen'), reviewOffer: true });
      http.expectNone((r) => r.url === '/lessons/mine');
    });

    it('does not ask with no lesson completed yet', async () => {
      await open({ study: studying(), overview: true, reviewOffer: true });
      http
        .expectOne((r) => r.url === '/lessons/mine' && r.params.get('all') === '1')
        .flush({ ...mine, completed: [] });
      await fixture.whenStable();
      expect(dialog().querySelector('[role="alertdialog"]')).toBeNull();
    });
  });

  describe('learning a completed lesson again', () => {
    const completed = () => completedStudy();
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

      // Learnt again, the lesson goes through its own steps (the speaking is off here).
      expect(text(el.querySelector('.progress-text'))).toBe('1/5');
      expect(text(el.querySelector('.relearn-note'))).toContain('Đang học lại bài');
      expect(button('Học lại bài')).toBeUndefined();
      // Learning again starts from the overview.
      expect(el.querySelector('#lesson-overview')).not.toBeNull();
      expect(text(nextButton())).toBe('Bắt đầu: Từ vựng');
      await next(); // the overview → words
      await next(); // words → Nghe, no cards saved again
      http.expectNone(() => true);
      expect(child(ListenStub)!.mode()).toBe('study');
      child(ListenStub)!.completed.emit();
      await fixture.whenStable();
      expect(child(ReadingStub)!.mode()).toBe('study');
      child(ReadingStub)!.completed.emit();
      await fixture.whenStable();
      child(GrammarStub)!.done.emit();
      await fixture.whenStable();
      http.expectNone((r) => r.url.includes('/steps/'));
      // The practice is Luyện thêm: Viết comes last, its writing only shown again.
      expect(text(el.querySelector('.progress-text'))).toBe('5/5');
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
    it('is Nói, after Ngữ pháp, in the lesson being studied, and records the step', async () => {
      localStorage.setItem('luna.lesson-step.l1', 'speak');
      await open({ study: at(['vocab', 'listen', 'read', 'grammar'], 'speak'), speaking: true });
      expect(text(el.querySelector('.progress-text'))).toBe('5/6');
      expect(el.querySelector('lu-speak-step')).toBeNull();
      const speak = child(SpeakLessonStub)!;
      expect(speak.lessonId()).toBe('l1');
      expect(speak.number()).toBe(5);
      expect(speak.topicName()).toBe('Làm quen');
      // Without the lesson's own sentences, two short turns of the dialogue.
      expect(speak.readAloud()).toEqual([
        { text: 'Hi, my name is Minh.', tipVi: '' },
        { text: 'Nice to meet you.', tipVi: 'meet /mit/: nhớ đọc rõ âm cuối /t/. Người Việt hay nuốt âm cuối.' },
      ]);
      expect(speak.fallbackSentences()).toEqual(['Hi, my name is Minh.', 'Nice to meet you.']);
      expect(speak.finishLabel()).toBe('Sang bước Viết');
      expect(speak.canBack()).toBe(true);
      speak.done.emit();
      const req = http.expectOne('/lessons/l1/steps/speak/complete');
      expect(req.request.method).toBe('POST');
      req.flush(writeNext());
      await fixture.whenStable();
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      expect(child(WritingStub)!.mode()).toBe('study');
    });

    it('comes last in another lesson, practised on the sentences', async () => {
      await open({ speaking: true });
      expect(text(el.querySelector('.progress-text'))).toBe('1/5');
      await next();
      await next();
      await next();
      await next();
      expect(text(el.querySelector('.progress-text'))).toBe('5/5');
      expect(text(el.querySelector('lu-speak-step .sentence'))).toBe('Hi, my name is Minh.');
      expect(child(SpeakLessonStub)).toBeUndefined();
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
      // One word known leaves nothing to recall: the card goes on to the next step.
      el.querySelector<HTMLButtonElement>('lu-vocab-step .known-btn')!.click();
      await fixture.whenStable();
      expect(text(cardNext())).toBe('Sang bước Hội thoại');
      cardNext().click();
      await fixture.whenStable();
      // Another lesson only practises the words: nothing is saved.
      http.expectNone(() => true);
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

    it("puts ← and the step's main button in one bar; the overview lists every step", async () => {
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

    it("names each step's last button after the next step", async () => {
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
      // The last word leads to the quick recall.
      expect(text(cardNext())).toBe('Nhớ lại nhanh');
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
      const sentences = Array.from(
        el.querySelectorAll<HTMLButtonElement>('#panel-reading .read-sentence'),
      );
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
      await open({ study: completedStudy() });
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
      el.querySelector<HTMLButtonElement>(
        'lu-practice-summary button[aria-label="Làm Điền ô trống: chưa làm"]',
      )!.click();
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
    /** Studying lesson l1 on Viết, every other step done: Luyện thêm is open. */
    const toPractice = async () => {
      await open({ study: writeNext() });
    };
    /** The end of the words of the lesson being studied: the cards saved, then the step recorded. */
    const finishVocab = async (lemmas: string[], after: LessonStudy) => {
      const bulk = http.expectOne('/vocab/cards/bulk');
      expect(bulk.request.method).toBe('POST');
      expect(bulk.request.body).toEqual({ lessonId: 'l1', lemmas });
      bulk.flush({ added: lemmas.length, cards: [] });
      await settle();
      await reply('/lessons/l1/steps/vocab/complete', after);
      await settle();
    };
    /** The chips of Luyện thêm in the overview, whether each one is disabled. */
    const extraChips = () =>
      Array.from(el.querySelectorAll<HTMLButtonElement>('.info-extras .chip-btn')).map(
        (b) => b.disabled,
      );

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
      http.expectOne('/vocab/practice-misses').flush('down', { status: 500, statusText: 'Error' });
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

    it('opens on the overview: the six steps, the time, Luyện thêm, then Bắt đầu', async () => {
      await open({ study: studying(), overview: true, speaking: true });
      const overview = el.querySelector('#lesson-overview')!;
      expect(text(overview.querySelector('.estimate'))).toBe('6 bước · khoảng 21 phút');
      expect(Array.from(overview.querySelectorAll('.info-step')).map((b) => text(b))).toEqual([
        '1 Từ vựng',
        '2 Nghe',
        '3 Đọc',
        '4 Ngữ pháp',
        '5 Nói',
        '6 Viết',
      ]);
      // Luyện thêm is shown but locked until Đọc is done: before that it gives the answers away.
      expect(text(overview.querySelector('.info-extras'))).toContain('Luyện thêm (tuỳ chọn)');
      expect(text(overview.querySelector('.extras-note'))).toBe('Mở sau khi xong bước Đọc');
      const chips = Array.from(
        overview.querySelectorAll<HTMLButtonElement>('.info-extras .chip-btn'),
      );
      expect(chips.map((b) => b.disabled)).toEqual([true, true, true]);
      expect(chips[0].getAttribute('aria-describedby')).toBe('extras-note');
      expect(el.querySelector('lu-vocab-step')).toBeNull();
      expect(el.querySelector('.next-bar .back')).toBeNull();
      expect(text(nextButton())).toBe('Bắt đầu: Từ vựng');
      await next();
      expect(el.querySelector('#lesson-overview')).toBeNull();
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      // The words card has its own bar; from the first step, its ← goes back to the overview.
      expect(el.querySelector('.next-bar')).toBeNull();
      el.querySelector<HTMLButtonElement>('lu-vocab-step .prev')!.click();
      await fixture.whenStable();
      expect(el.querySelector('#lesson-overview')).not.toBeNull();
    });

    it('says Tiếp tục on the overview of a lesson already started', async () => {
      await open({ study: at(['vocab'], 'listen'), overview: true });
      expect(text(nextButton())).toBe('Tiếp tục: Nghe');
      expect(stepButton('Từ vựng').classList).toContain('is-done');
    });

    it('saves the words but those known, sends those got wrong, then records Từ vựng', async () => {
      await open({ study: studying() });
      child(VocabStep)!.done.emit({ known: ['meet'], missed: ['name', 'meet'] });
      await settle();
      const bulk = http.expectOne('/vocab/cards/bulk');
      expect(bulk.request.body).toEqual({ lessonId: 'l1', lemmas: ['name'] });
      // Nothing is recorded before the cards are saved.
      http.expectNone('/lessons/l1/steps/vocab/complete');
      bulk.flush({ added: 1, cards: [] });
      await settle();
      // A known word is not sent as missed.
      const misses = http.expectOne('/vocab/practice-misses');
      expect(misses.request.body).toEqual({ lessonId: 'l1', words: ['name'] });
      misses.flush({ added: 1, rescheduled: 0 });
      await reply('/lessons/l1/steps/vocab/complete', at(['vocab'], 'listen'));
      expect(text(el.querySelector('.progress-text'))).toBe('2/5');
      expect(child(ListenStub)!.mode()).toBe('study');
      expect(text(el.querySelector('.miss-note'))).toBe('Đã đưa 1 từ làm sai vào lịch ôn.');
    });

    it('goes through the words on the card, the known ones left out of the cards', async () => {
      await open({ study: studying() });
      const cardNext = () => el.querySelector<HTMLButtonElement>('lu-vocab-step .next-word')!;
      cardNext().click();
      await fixture.whenStable();
      el.querySelector<HTMLButtonElement>('lu-vocab-step .known-btn')!.click();
      await fixture.whenStable();
      expect(text(cardNext())).toBe('Sang bước Nghe');
      cardNext().click();
      await finishVocab(['name'], at(['vocab'], 'listen'));
      http.expectNone('/vocab/practice-misses');
      expect(child(ListenStub)).toBeDefined();
    });

    it('still records Từ vựng when the cards cannot be saved', async () => {
      await open({ study: studying() });
      child(VocabStep)!.done.emit({ known: [], missed: [] });
      await settle();
      http.expectOne('/vocab/cards/bulk').flush('down', { status: 500, statusText: 'Error' });
      await settle();
      await reply('/lessons/l1/steps/vocab/complete', at(['vocab'], 'listen'));
      await settle();
      expect(child(ListenStub)).toBeDefined();
      expect(el.querySelector('[role="alert"]')).toBeNull();
    });

    it('goes Từ vựng, Nghe, Đọc, Ngữ pháp, Nói, then Viết, each recorded, the practice kept as Luyện thêm', async () => {
      await open({ study: studying(), speaking: true });
      expect(text(el.querySelector('.progress-text'))).toBe('1/6');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
      child(VocabStep)!.done.emit({ known: [], missed: [] });
      await settle();
      await finishVocab(['name', 'meet'], at(['vocab'], 'listen'));

      // Nghe has its own bar: ← and a quiet way to leave it for later.
      expect(text(el.querySelector('.progress-text'))).toBe('2/6');
      const listen = child(ListenStub)!;
      expect(listen.mode()).toBe('study');
      expect(listen.lessonId()).toBe('l1');
      expect(listen.practice()).toEqual(practice);
      expect(listen.number()).toBe(2);
      expect(listen.canBack()).toBe(true);
      expect(listen.laterLabel()).toBe('Để sau, sang bước Đọc');
      expect(listen.nextLabel()).toBe('Sang bước Đọc');
      expect(el.querySelector('.next-bar')).toBeNull();
      listen.completed.emit();
      await reply('/lessons/l1/steps/listen/complete', at(['vocab', 'listen'], 'read'));

      expect(text(el.querySelector('.progress-text'))).toBe('3/6');
      const reading = child(ReadingStub)!;
      expect(reading.mode()).toBe('study');
      expect(reading.number()).toBe(3);
      expect(reading.canBack()).toBe(true);
      expect(reading.laterLabel()).toBe('Để sau, sang bước Ngữ pháp');
      // The questions' result goes on with the same step's name.
      expect(reading.doneLabel()).toBe('Sang bước Ngữ pháp');
      expect(reading.targetWords()).toEqual(['name', 'meet']);
      expect(el.querySelector('.next-bar')).toBeNull();

      // The details go back to Nghe, shown again as done, with its own way on.
      await openInfo();
      stepButton('Nghe').click();
      await fixture.whenStable();
      expect(child(ListenStub)!.mode()).toBe('review');
      expect(child(ListenStub)!.nextLabel()).toBe('Sang bước Đọc');
      await next();
      child(ReadingStub)!.completed.emit();
      await reply('/lessons/l1/steps/read/complete', at(['vocab', 'listen', 'read'], 'grammar'));

      expect(text(el.querySelector('.progress-text'))).toBe('4/6');
      const grammar = child(GrammarStub)!;
      expect(grammar.number()).toBe(4);
      expect(grammar.note()).toEqual(lesson.grammarNote);
      expect(grammar.content()).toBeNull();
      expect(grammar.finishLabel()).toBe('Sang bước Nói');
      expect(el.querySelector('.next-bar')).toBeNull();
      grammar.done.emit();
      await reply(
        '/lessons/l1/steps/grammar/complete',
        at(['vocab', 'listen', 'read', 'grammar'], 'speak'),
      );

      expect(text(el.querySelector('.progress-text'))).toBe('5/6');
      expect(el.querySelector('.next-bar')).toBeNull();
      child(SpeakLessonStub)!.done.emit();
      await reply('/lessons/l1/steps/speak/complete', writeNext());

      // Viết is last, with the page's bar (← only: the writing has its own buttons).
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      expect(child(WritingStub)!.mode()).toBe('study');
      expect(child(WritingStub)!.targetWords()).toEqual(['name', 'meet']);
      expect(el.querySelector('.next-bar .back')).not.toBeNull();
      expect(nextButton()).toBeNull();
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
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      await next();
      expect(child(DialogueStep)).toBeUndefined();
      expect(child(WritingStub)!.mode()).toBe('study');
    });

    it('keeps Luyện thêm locked until Đọc is done', async () => {
      await open({ study: at(['vocab', 'listen'], 'read'), overview: true });
      expect(extraChips()).toEqual([true, true, true]);
      expect(text(el.querySelector('.extras-note'))).toBe('Mở sau khi xong bước Đọc');
      nextButton()!.click();
      await fixture.whenStable();
      child(ReadingStub)!.completed.emit();
      await reply('/lessons/l1/steps/read/complete', at(['vocab', 'listen', 'read'], 'grammar'));
      await openInfo();
      expect(el.querySelector('.extras-note')).toBeNull();
      expect(extraChips()).toEqual([false, false, false]);
    });

    it('does not record again a step already done: it only moves on', async () => {
      await open({ study: writeNext(), speaking: true });
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      await openInfo();
      stepButton('Ngữ pháp').click();
      await fixture.whenStable();
      child(GrammarStub)!.done.emit();
      await fixture.whenStable();
      expect(text(el.querySelector('.progress-text'))).toBe('5/6');
      child(SpeakLessonStub)!.done.emit();
      await fixture.whenStable();
      http.expectNone((r) => r.url.includes('/steps/'));
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      expect(child(WritingStub)!.mode()).toBe('study');
    });

    it('opens Luyện thêm once Đọc is done, and offers it on Viết', async () => {
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
        Array.from(el.querySelectorAll<HTMLButtonElement>('.info-extras .chip-btn')).every(
          (b) => !b.disabled,
        ),
      ).toBe(true);
    });

    it('opens Luyện thêm from the summary reached with Đọc left for later', async () => {
      // Viết done, the others left for later: the last step's Hoàn thành shows the summary.
      await open({
        study: studying({ steps: stepsOf(['write'], 'vocab'), currentStep: 'vocab' }),
        speaking: true,
      });
      el.querySelector<HTMLButtonElement>('.info-btn')!.click();
      await fixture.whenStable();
      Array.from(el.querySelectorAll<HTMLButtonElement>('.info-step'))
        .find((b) => text(b).includes('Viết'))!
        .click();
      await fixture.whenStable();
      nextButton()!.click();
      await fixture.whenStable();
      // The lesson is not done: the summary says what was left for later.
      expect(Array.from(el.querySelectorAll('.left-behind button')).map((b) => text(b))).toEqual([
        'Làm bước Từ vựng',
        'Làm bước Nghe',
        'Làm bước Đọc',
        'Làm bước Ngữ pháp',
        'Làm bước Nói',
      ]);
      const row = el.querySelector<HTMLButtonElement>(
        'lu-practice-summary button[aria-label="Làm Hội thoại: chưa làm"]',
      )!;
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

    it('leaves Nghe and Đọc for later, and points back to them from Viết', async () => {
      await open({ study: at(['vocab'], 'listen'), speaking: true });
      await next(); // Để sau on Nghe: nothing is recorded
      expect(child(ReadingStub)!.mode()).toBe('study');
      await next(); // Để sau on Đọc
      expect(child(GrammarStub)).toBeDefined();
      http.expectNone(() => true);
      await openInfo();
      stepButton('Viết').click();
      await fixture.whenStable();
      expect(text(el.querySelector('.progress-text'))).toBe('6/6');
      // Viết has its own buttons: nothing to skip to, ← only.
      expect(nextButton()).toBeNull();
      expect(el.querySelector('.next-bar .back')).not.toBeNull();
      expect(text(el.querySelector('.left-behind'))).toContain('Còn để lại');
      const steps = Array.from(el.querySelectorAll('.left-behind button')).map((b) => text(b));
      expect(steps).toEqual(['Làm bước Nghe', 'Làm bước Đọc', 'Làm bước Ngữ pháp', 'Làm bước Nói']);
      button('Làm bước Đọc')!.click();
      await fixture.whenStable();
      expect(child(ReadingStub)!.mode()).toBe('study');
    });

    it('Bỏ qua the writing finishes the lesson and offers the next one', async () => {
      await open({ study: writeNext() });
      // Back where the learner left off.
      expect(text(el.querySelector('.progress-text'))).toBe('5/5');
      child(WritingStub)!.skipped.emit();
      await reply(
        '/lessons/l1/steps/write/skip',
        other({
          status: 'completed',
          steps: stepsOf([...STEPS], 'done'),
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
      expect(text(el.querySelector('.progress-text'))).toBe('1/5');
      expect(el.querySelector('lu-vocab-step')).not.toBeNull();
    });

    it('says when the topic has no next lesson yet', async () => {
      await open({ study: writeNext() });
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
      await open({ study: writeNext() });
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
      await open({ study: writeNext() });
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
      expect(el.querySelector('.done-card a.btn-primary')!.getAttribute('href')).toBe(
        '/lessons/l2',
      );
    });

    it('goes to the congratulations at the end of the roadmap', async () => {
      await open({ study: writeNext() });
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
        study: at(['vocab', 'listen'], 'read'),
      });
      // Without practice the lesson still has its steps, and Bài đọc.
      expect(text(el.querySelector('.progress-text'))).toBe('3/5');
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
        study: at(['vocab', 'listen'], 'read'),
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
      const writeCurrent = () => writeNext();

      it('is where the learner comes back to when they left steps for later', async () => {
        // The server is on Nghe; the learner left it (and Đọc) for later and was on Ngữ pháp.
        localStorage.setItem('luna.lesson-step.l1', 'grammar');
        await open({ study: at(['vocab'], 'listen') });
        expect(text(el.querySelector('.progress-text'))).toBe('4/5');
        expect(child(GrammarStub)).toBeDefined();
        expect(child(ListenStub)).toBeUndefined();
      });

      it('never puts the learner behind what the server already knows', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'vocab');
        await open({ study: writeCurrent() });
        expect(text(el.querySelector('.progress-text'))).toBe('5/5');
      });

      it('ignores a saved step that does not exist any more', async () => {
        // The words step was called "words" before the six steps.
        localStorage.setItem('luna.lesson-step.l1', 'words');
        await open({ study: at(['vocab'], 'listen') });
        expect(text(el.querySelector('.progress-text'))).toBe('2/5');
      });

      it('ignores a saved practice step, now Luyện thêm', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'fill');
        await open({ study: writeCurrent() });
        expect(text(el.querySelector('.progress-text'))).toBe('5/5');
        expect(child(WritingStub)).toBeDefined();
      });

      it('is saved as the learner moves, and the opening does not overwrite it', async () => {
        localStorage.setItem('luna.lesson-step.l1', 'grammar');
        await open({ study: at(['vocab'], 'listen') });
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBe('grammar');
        child(GrammarStub)!.back.emit();
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBe('read');
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
          other({
            status: 'completed',
            currentStep: 'done',
            next: { id: 'l2', title: 'Family 2' },
          }),
        );
        await flushDue(0);
        await fixture.whenStable();
        expect(localStorage.getItem('luna.lesson-step.l1')).toBeNull();
      });
    });

    it('starts again where a saved position is', async () => {
      await open({ study: at(['vocab', 'listen'], 'read', { sentenceIndex: 4 }) });
      expect(text(el.querySelector('.progress-text'))).toBe('3/5');
      expect(child(ReadingStub)!.startSentence()).toBe(4);
    });

    it('starts Nghe again where a saved position is, and saves it there', async () => {
      await open({ study: at(['vocab'], 'listen', { sentenceIndex: 2 }) });
      expect(text(el.querySelector('.progress-text'))).toBe('2/5');
      expect(child(ListenStub)!.startSentence()).toBe(2);
      vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
      try {
        child(ListenStub)!.position.emit(3);
        vi.advanceTimersByTime(1000);
        const req = http.expectOne('/lessons/l1/position');
        expect(req.request.body).toEqual({ step: 'listen', sentenceIndex: 3 });
        req.flush(null, { status: 204, statusText: 'No Content' });
      } finally {
        vi.useRealTimers();
      }
    });
  });
});
