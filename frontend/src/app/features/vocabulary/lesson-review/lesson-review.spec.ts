import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { PracticeView } from '../../../core/models/practice';
import { DueCard, LessonResult, LessonResultInput, VocabItem } from '../../../core/models/vocab';
import { FocusMode } from '../../../core/services/focus-mode.service';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { LessonReview } from './lesson-review';

const item = (lemma: string, meaningVi: string): VocabItem => ({
  lemma,
  text: lemma,
  meaningVi,
  ipa: '',
  sentenceIndex: 0,
  sentence: `The ${lemma} is here.`,
  pos: 'noun',
});

const due: DueCard = {
  id: 'c1',
  text: 'meeting',
  lemma: 'meeting',
  ipa: '/ˈmiːtɪŋ/',
  meaningVi: 'cuộc họp',
  contextSentence: 'The meeting starts.',
  lessonId: 'l1',
  source: 'ai',
  createdAt: '',
  due: '',
  reps: 2,
  state: 'review',
  intervals: { again: 60, hard: 300, good: 600, easy: 86400 },
};
const otherLesson: DueCard = { ...due, id: 'c2', lemma: 'go', text: 'go', lessonId: 'l2' };

const practice: PracticeView = {
  status: 'done',
  lessonNumber: 3,
  objectiveVi: '',
  examples: [],
  dialogue: null,
  grammarTipVi: '',
  fill: {
    turns: [
      {
        speaker: 0,
        turnIndex: 0,
        meaningVi: 'Sếp tôi ở đây.',
        parts: [{ text: 'My ' }, { blank: 0 }, { text: ' is here.' }],
      },
    ],
    blanks: [{ answer: 'boss' }],
    wordBank: ['boss', 'desk'],
  },
  translations: [{ vi: 'Tôi đi.', answer: ['I', 'go.'], tiles: ['go.', 'I', 'we'] }],
};

describe('LessonReview', () => {
  let fixture: ComponentFixture<LessonReview>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find(
      (b) => b.textContent?.replace(/\s+/g, ' ').trim() === label,
    );
  const click = async (b: HTMLElement | null | undefined) => {
    b!.click();
    await settle();
  };
  const choice = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.choice')).find(
      (c) => c.querySelector('.text')?.textContent?.trim() === label,
    );
  const tile = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.bank .tile')).find(
      (t) => t.textContent?.trim() === label,
    );

  const setup = async (withPractice = true, result: LessonResult | null = null) => {
    TestBed.configureTestingModule({
      imports: [LessonReview],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: convertToParamMap({ id: 'l1' }) } },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonReview);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/lessons/l1').flush({ lesson: { id: 'l1', title: 'At the office' } });
    http
      .expectOne('/lessons/l1/vocabulary')
      .flush({ available: true, items: [item('desk', 'bàn làm việc'), item('boss', 'sếp')] });
    const p = http.expectOne('/lessons/l1/practice');
    if (withPractice) {
      p.flush(practice);
    } else {
      p.flush({ error: 'not_found', message: 'x' }, { status: 404, statusText: 'Not Found' });
    }
    http
      .expectOne((r) => r.url === '/vocab/review/due')
      .flush({ cards: [due, otherLesson], total: 2, nextDue: null });
    http.expectOne('/vocab/review/lessons').flush({
      lessons: result
        ? [
            {
              lessonId: 'l1',
              cards: 1,
              due: 1,
              remembered: 0,
              lastReview: null,
              nextDue: null,
              result,
            },
          ]
        : [],
    });
    await settle();
  };
  /** The score kept after a round. */
  const expectSaved = (body: LessonResultInput) => {
    const req = http.expectOne('/vocab/review/lessons/l1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual(body);
    req.flush({ result: { ...body, reviewedAt: new Date().toISOString() } });
  };

  afterEach(() => {
    http.verify();
  });

  it('says what the round holds and lets a part be left out', async () => {
    await setup();
    expect(text('.intro-title')).toBe('At the office');
    expect(
      Array.from(el.querySelectorAll('.part-row')).map((r) =>
        r.textContent?.replace(/\s+/g, ' ').trim(),
      ),
    ).toEqual([
      '1. Từ vựng Chọn nghĩa đúng của từ · 1 từ đến hạn luôn được hỏi 3 câu',
      '2. Điền từ Câu trong bài, chọn từ còn thiếu 1 câu',
      '3. Dịch câu Ghép từ thành câu tiếng Anh 1 câu',
    ]);
    expect(text('.intro-head .muted')).toBe('5 câu · khoảng 2 phút');

    await click(el.querySelectorAll<HTMLButtonElement>('.part-row')[2]);
    expect(el.querySelectorAll('.part-row')[2].getAttribute('aria-pressed')).toBe('false');
    expect(text('.intro-head .muted')).toBe('4 câu · khoảng 2 phút');
  });

  it('leaves out the parts the lesson has no practice for', async () => {
    await setup(false);
    expect(el.querySelectorAll('.part-row')).toHaveLength(1);
  });

  it('goes through the three parts and shows the score', async () => {
    await setup();
    const focus = TestBed.inject(FocusMode);
    await click(button('Bắt đầu ôn'));
    expect(focus.on()).toBe(true);
    expect(el.querySelectorAll('.segment')).toHaveLength(3);

    // Words: the due card first, rated as soon as it is answered.
    expect(text('.part-label')).toBe('Từ vựng · 1/3');
    expect(text('.word')).toBe('meeting');
    await click(choice('cuộc họp'));
    await click(button('Nộp'));
    expect(text('.counter')).toBe('✓ Đúng rồi');
    const rated = http.expectOne('/vocab/cards/c1/review');
    expect(rated.request.body).toEqual({ rating: 3, mode: 'flip', reps: 2, context: 'free' });
    rated.flush({ card: due });
    expect(text('.btn-next')).toBe('Tiếp ôn lại sau 10 phút');
    await click(el.querySelector('.btn-next') as HTMLElement);

    // The two other words: one forgotten, one right.
    await click(button('Quên'));
    expect(text('.counter')).toBe('Nghĩa của từ');
    const forgotten = text('.word');
    await click(el.querySelector('.btn-next') as HTMLElement);
    const meaning = forgotten === 'desk' ? 'sếp' : 'bàn làm việc';
    await click(choice(meaning));
    await click(button('Nộp'));
    await click(el.querySelector('.btn-next') as HTMLElement);

    // Fill-in: the word in the gap.
    expect(text('.part-label')).toBe('Điền từ · 1/1');
    expect(text('.line')).toBe('My is here.');
    await click(choice('desk'));
    expect(text('.gap')).toBe('desk');
    await click(button('Nộp'));
    expect(text('.counter')).toBe('✗ Chưa đúng');
    expect(text('.gap')).toBe('boss');
    await click(el.querySelector('.btn-next') as HTMLElement);

    // Translation: tiles in order.
    expect(text('.part-label')).toBe('Dịch câu · 1/1');
    await click(tile('I'));
    await click(tile('go.'));
    expect(text('.built')).toBe('I go.');
    await click(button('Nộp'));
    expect(text('.counter')).toBe('✓ Đúng rồi');
    await click(button('Xem kết quả'));

    // The words got wrong come back for review.
    const misses = http.expectOne('/vocab/practice-misses');
    // Each word once ("boss" may be both the word forgotten and the blank got wrong).
    expect(misses.request.body).toEqual({
      lessonId: 'l1',
      words: [...new Set([forgotten, 'boss'])],
    });
    misses.flush({ added: 1, rescheduled: 0 });
    expectSaved({ total: 5, correct: 3, wrong: [`v:${forgotten}`, 'f:0'] });
    await settle();

    expect(focus.on()).toBe(false);
    expect(text('.score')).toBe('3/5');
    expect(
      Array.from(el.querySelectorAll('.part-scores li')).map((li) =>
        li.textContent?.replace(/\s+/g, ' ').trim(),
      ),
    ).toEqual(['Từ vựng 2/3', 'Điền từ 0/1', 'Dịch câu 1/1']);
    expect(text('.wrongs-title')).toBe('Câu chưa đúng (2)');
    expect(text('.wrongs')).toContain('My ___ is here. desk → boss');

    // Again, only the questions got wrong; the due card is not rated twice.
    await click(button('Ôn lại câu sai'));
    expect(text('.progress-text')).toBe('1/2');
    expect(text('.word')).toBe(forgotten);
  });

  it('starts the whole lesson again from the result', async () => {
    await setup(false);
    await click(button('Bắt đầu ôn'));
    for (let i = 0; i < 3; i++) {
      await click(button('Quên'));
      if (text('.word') === 'meeting') {
        http.expectOne('/vocab/cards/c1/review').flush({ card: due });
      }
      await click(el.querySelector('.btn-next') as HTMLElement);
    }
    http.expectOne('/vocab/practice-misses').flush({ added: 2, rescheduled: 0 });
    http
      .expectOne('/vocab/review/lessons/l1')
      .flush({ result: { correct: 0, total: 3, wrong: [], reviewedAt: '' } });
    expect(text('.score')).toBe('0/3');

    await click(button('Ôn lại cả bài'));
    expect(text('.progress-text')).toBe('1/3');
    expect(el.querySelector('.choice.right')).toBeNull();
    // The due card was rated in the first round: answering its word again sends no rating.
    for (let i = 0; i < 3; i++) {
      await click(button('Quên'));
      await click(el.querySelector('.btn-next') as HTMLElement);
    }
    http.expectOne('/vocab/practice-misses').flush({ added: 0, rescheduled: 3 });
    http
      .expectOne('/vocab/review/lessons/l1')
      .flush({ result: { correct: 0, total: 3, wrong: [], reviewedAt: '' } });
    expect(text('.score')).toBe('0/3');
  });

  it('offers the questions got wrong last time, and mends the score', async () => {
    const reviewedAt = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    // "v:gone": a word the lesson no longer has.
    await setup(true, { correct: 2, total: 5, wrong: ['v:desk', 'f:0', 'v:gone'], reviewedAt });
    expect(text('.last-result')).toBe('Lần trước: 2/5 câu đúng · hôm qua');
    expect(button('Ôn cả bài')).toBeTruthy();
    expect(button('Bắt đầu ôn')).toBeUndefined();

    await click(button('Ôn lại câu sai (2)'));
    expect(text('.progress-text')).toBe('1/2');
    expect(text('.word')).toBe('desk');
    await click(choice('bàn làm việc'));
    await click(button('Nộp'));
    await click(el.querySelector('.btn-next') as HTMLElement);
    await click(choice('boss'));
    await click(button('Nộp'));
    await click(button('Xem kết quả'));

    expectSaved({ total: 5, correct: 5, wrong: [] });
    await settle();
    expect(text('.score')).toBe('2/2');
    expect(text('.lesson-score')).toBe('Điểm cả bài: 5/5');
  });

  it('still asks the word due with every part left out', async () => {
    await setup(false);
    await click(el.querySelectorAll<HTMLButtonElement>('.part-row')[0]);
    expect(text('.intro-head .muted')).toBe('1 câu · khoảng 1 phút');
    await click(button('Bắt đầu ôn'));
    expect(text('.word')).toBe('meeting');
  });
});
