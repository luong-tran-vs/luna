import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { Quiz, QuizAnswer } from '../../../../core/models/reading';
import { ComprehensionQuiz, optionOrder, QuizScore } from './comprehension-quiz';

const quiz = (answers: QuizAnswer[] = []): Quiz => ({
  version: 3,
  questions: [
    { prompt: 'Where did we go?', options: ['Park', 'Home', 'School', 'Work'] },
    { prompt: 'What did he give up?', options: ['Tea', 'Smoking', 'Sport', 'Work'] },
    { prompt: 'Who cooks dinner?', options: ['Mum', 'Dad', 'Me', 'Nobody'] },
  ],
  answers,
});

const answer = (questionIndex: number, choice: number, answerIndex: number): QuizAnswer => ({
  questionIndex,
  choice,
  correct: choice === answerIndex,
  answerIndex,
  explanationVi: `Giải thích ${questionIndex + 1}`,
});

/** The six-step lesson's questions (2026-10-22): one of each kind but detail. */
const kinded = (answers: QuizAnswer[] = []): Quiz => ({
  version: 4,
  questions: [
    { kind: 'gist', prompt: 'What is the text about?', promptVi: 'Bài nói về điều gì?', options: ['Music', 'Sport', 'Food', 'Work'] },
    { kind: 'tfn', prompt: 'The sister plays the guitar.', promptVi: 'Em gái chơi đàn guitar.', options: ['Đúng', 'Sai', 'Không có trong bài'] },
    {
      kind: 'vocab',
      prompt: 'Trong câu này, quiet nghĩa là gì?',
      quote: '…but sometimes it is quiet.',
      options: ['chậm', 'nhỏ, khẽ', 'vắng vẻ', 'ít nói'],
    },
  ],
  answers,
});

/** Every question answered: the first one wrong. */
const allAnswered = () => [answer(0, 1, 0), answer(1, 1, 1), answer(2, 2, 2)];

describe('ComprehensionQuiz', () => {
  let fixture: ComponentFixture<ComprehensionQuiz>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let scores: QuizScore[];
  let reloads: number;
  let backs: number;
  let finishes: number;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const options = () => Array.from(el.querySelectorAll<HTMLInputElement>('.option input'));
  const optionTexts = () => Array.from(el.querySelectorAll('.option .option-text')).map((o) => text(o));
  const prompt = () => text(el.querySelector('.prompt'));
  const label = () => text(el.querySelector('lu-asker .counter'));
  const main = () => el.querySelector<HTMLButtonElement>('.bottom-bar .btn-primary')!;
  const nav = () => el.querySelector<HTMLButtonElement>('.bottom-bar .nav-btn');
  const tabs = () => Array.from(el.querySelectorAll<HTMLButtonElement>('.tabs .tab'));
  const button = (name: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === name || text(b).startsWith(name));
  const click = async (b: HTMLElement | null | undefined) => {
    b!.click();
    await settle();
  };
  /** Picks an option, then Kiểm tra. */
  const pick = async (j: number) => {
    options()[j].click();
    await fixture.whenStable();
    main().click();
  };
  const setup = async (q: Quiz, inputs: { review?: boolean; canBack?: boolean; finishLabel?: string } = {}) => {
    await TestBed.configureTestingModule({
      imports: [ComprehensionQuiz],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ComprehensionQuiz);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('quiz', q);
    fixture.componentRef.setInput('review', inputs.review ?? false);
    fixture.componentRef.setInput('canBack', inputs.canBack ?? false);
    fixture.componentRef.setInput('finishLabel', inputs.finishLabel ?? '');
    scores = [];
    reloads = 0;
    backs = 0;
    finishes = 0;
    fixture.componentInstance.allAnswered.subscribe((s) => scores.push(s));
    fixture.componentInstance.reload.subscribe(() => reloads++);
    fixture.componentInstance.back.subscribe(() => backs++);
    fixture.componentInstance.finished.subscribe(() => finishes++);
    el = fixture.nativeElement;
    await fixture.whenStable();
  };
  const respond = async (body: object, status = 200) => {
    const req = http.expectOne('/lessons/l1/answers');
    if (status === 200) {
      req.flush(body);
    } else {
      req.flush(body, { status, statusText: 'Error' });
    }
    await settle();
    return req;
  };
  /** Answers the question on screen with choice j, the server saying answerIndex is right. */
  const answerShown = async (j: number, answerIndex: number) => {
    const i = Number(el.querySelector('.prompt')!.id.replace('quiz-q-', ''));
    await pick(j);
    await respond({ answer: answer(i, j, answerIndex), answered: 1, total: 3, correct: 0 });
  };

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  it('shows the first question with its four options and a tab per question', async () => {
    await setup(quiz());
    expect(prompt()).toBe('Where did we go?');
    expect(label()).toBe('Câu 1/3');
    expect(optionTexts()).toEqual(['Park', 'Home', 'School', 'Work']);
    expect(tabs().map((t) => text(t))).toEqual(['1 Câu', '2 Câu', '3 Câu']);
    expect(tabs().map((t) => t.getAttribute('aria-label'))).toEqual([
      'Câu 1: chưa trả lời',
      'Câu 2: chưa trả lời',
      'Câu 3: chưa trả lời',
    ]);
    expect(tabs()[0].getAttribute('aria-current')).toBe('step');
    expect(main().textContent?.trim()).toBe('Kiểm tra');
    expect(main().disabled).toBe(true);
  });

  it('checks the picked option; the bubble says the result and the options stay where they were', async () => {
    await setup(quiz());
    options()[1].click();
    await fixture.whenStable();
    expect(main().disabled).toBe(false);
    main().click();
    await fixture.whenStable();
    expect(main().textContent?.trim()).toBe('Đang kiểm tra…');
    expect(options().every((o) => o.disabled)).toBe(true);

    const req = await respond({ answer: answer(0, 1, 0), answered: 1, total: 3, correct: 0 });
    expect(req.request.body).toEqual({ version: 3, questionIndex: 0, choice: 1 });
    expect(prompt()).toBe('Where did we go?');
    expect(label()).toBe('Câu 1/3 · ✗ Sai rồi');
    expect(el.querySelector('lu-asker .bubble.wrong')).not.toBeNull();
    // The verdict is in the label: the bubble holds the question only.
    expect(text(el.querySelector('lu-asker .bubble'))).not.toContain('Giải thích');
    const results = el.querySelectorAll('.results .result');
    expect(results).toHaveLength(4);
    expect(text(results[0])).toContain('Đáp án đúng');
    expect(text(results[1])).toContain('Bạn chọn');
    expect(results[1].classList).toContain('is-wrong');
    expect(text(el.querySelector('.explanation'))).toBe('Giải thích: Giải thích 1');
    expect(tabs()[0].getAttribute('aria-label')).toBe('Câu 1: sai');
    expect(text(tabs()[0])).toBe('1 Câu');
    expect(tabs()[0].querySelector('.tab-mark')).not.toBeNull();
    expect(main().textContent?.trim()).toBe('Câu tiếp theo');
    expect(text(el.querySelector('[aria-live]'))).toBe('Câu 1: Sai.');
  });

  it('says right in the bubble for a right answer', async () => {
    await setup(quiz());
    await answerShown(0, 0);
    expect(label()).toBe('Câu 1/3 · ✓ Chính xác');
    expect(el.querySelector('lu-asker .bubble.right')).not.toBeNull();
    expect(text(el.querySelector('.results .is-answer'))).toContain('Bạn chọn đúng');
  });

  it('goes through the questions in order with ← and Câu tiếp theo, answered or not', async () => {
    await setup(quiz([answer(0, 0, 0), answer(1, 1, 1)]), { canBack: true });
    // It opens on the first question not answered.
    expect(prompt()).toBe('Who cooks dinner?');
    await click(nav());
    expect(prompt()).toBe('What did he give up?');
    await click(nav());
    expect(prompt()).toBe('Where did we go?');
    // Câu tiếp theo is the very next question, not the next one to answer.
    await click(main());
    expect(prompt()).toBe('What did he give up?');
    expect(main().textContent?.trim()).toBe('Câu tiếp theo');
    await click(main());
    expect(prompt()).toBe('Who cooks dinner?');

    await click(tabs()[0]);
    expect(nav()!.getAttribute('aria-label')).toBe('Về bài đọc');
    await click(nav());
    expect(backs).toBe(1);
  });

  it('has no ← on the first question when it cannot go back to the text', async () => {
    await setup(quiz());
    expect(nav()).toBeNull();
  });

  it('keeps the option picked on a question while going to others', async () => {
    await setup(quiz());
    options()[2].click();
    await fixture.whenStable();
    await click(tabs()[1]);
    expect(options().some((o) => o.checked)).toBe(false);
    await click(tabs()[0]);
    expect(options()[2].checked).toBe(true);
    expect(main().disabled).toBe(false);
  });

  it('skips a question for later, and on the last one sends to the one left', async () => {
    await setup(quiz());
    await click(button('Bỏ qua, làm sau'));
    expect(prompt()).toBe('What did he give up?');
    await click(button('Bỏ qua, làm sau'));
    expect(prompt()).toBe('Who cooks dinner?');
    // The last one: "Bỏ qua" goes back to the first question left.
    await click(button('Bỏ qua, làm sau'));
    expect(prompt()).toBe('Where did we go?');
    await click(tabs()[2]);
    await answerShown(2, 2);
    expect(main().textContent?.trim()).toBe('Làm câu còn thiếu (2)');
    await click(main());
    expect(prompt()).toBe('Where did we go?');
  });

  it('stays on the last answer, then shows the result; the step goes on only from there', async () => {
    await setup(quiz([answer(0, 1, 0), answer(1, 1, 1)]), { finishLabel: 'Sang bước Nghe' });
    await answerShown(2, 2);
    // The last answer stays on screen with its result.
    expect(label()).toBe('Câu 3/3 · ✓ Chính xác');
    expect(scores).toEqual([{ correct: 2, total: 3 }]);
    expect(main().textContent?.trim()).toBe('Xem kết quả');
    expect(finishes).toBe(0);

    await click(main());
    expect(fixture.componentInstance.resultShown()).toBe(true);
    expect(el.querySelector('.tabs')).toBeNull();
    expect(text(el.querySelector('.score'))).toBe('Kết quả 2/3 đúng');
    const rows = Array.from(el.querySelectorAll('.score-row'));
    expect(rows.map((r) => text(r))).toEqual([
      'Sai: 1. Where did we go? Xem lại ›',
      'Đúng: 2. What did he give up?',
      'Đúng: 3. Who cooks dinner?',
    ]);
    expect(button('Làm lại các câu sai')).toBeDefined();
    await click(button('Sang bước Nghe'));
    expect(finishes).toBe(1);
  });

  it('opens a question from the result, and ← on the result goes to the last question', async () => {
    await setup(quiz(allAnswered()));
    expect(fixture.componentInstance.resultShown()).toBe(true);
    await click(el.querySelectorAll<HTMLButtonElement>('.score-row')[0]);
    expect(prompt()).toBe('Where did we go?');
    expect(label()).toBe('Câu 1/3 · ✗ Sai rồi');

    await click(tabs()[2]);
    expect(main().textContent?.trim()).toBe('Xem kết quả');
    await click(main());
    await click(nav());
    expect(prompt()).toBe('Who cooks dinner?');
  });

  it('says so when every answer is right, with no Làm lại các câu sai', async () => {
    await setup(quiz([answer(0, 0, 0), answer(1, 1, 1), answer(2, 2, 2)]));
    expect(text(el.querySelector('.score'))).toBe('🎉 Kết quả 3/3 đúng');
    expect(button('Làm lại các câu sai')).toBeUndefined();
    expect(button('Làm lại tất cả')).toBeDefined();
  });

  it('opens the wrong answers again with Làm lại các câu sai; the new answer replaces the old', async () => {
    await setup(quiz(allAnswered()));
    await click(button('Làm lại các câu sai'));
    expect(prompt()).toBe('Where did we go?');
    expect(label()).toBe('Câu 1/3');
    expect(tabs().map((t) => t.getAttribute('aria-label'))).toEqual(['Câu 1: chưa trả lời', 'Câu 2: đúng', 'Câu 3: đúng']);

    await answerShown(0, 0);
    expect(label()).toBe('Câu 1/3 · ✓ Chính xác');
    expect(scores).toEqual([{ correct: 3, total: 3 }]);
  });

  it('answers one question again with Trả lời lại câu này', async () => {
    await setup(quiz([answer(0, 1, 0)]));
    await click(tabs()[0]);
    await click(button('Trả lời lại câu này'));
    expect(options()).toHaveLength(4);
    expect(tabs()[0].getAttribute('aria-label')).toBe('Câu 1: chưa trả lời');
    await answerShown(0, 0);
    expect(label()).toBe('Câu 1/3 · ✓ Chính xác');
  });

  it('starts every question again with Làm lại tất cả', async () => {
    await setup(quiz(allAnswered()));
    await click(button('Làm lại tất cả'));
    const req = http.expectOne('/lessons/l1/answers');
    expect(req.request.method).toBe('DELETE');
    req.flush(null);
    await settle();
    expect(prompt()).toBe('Where did we go?');
    expect(tabs().every((t) => t.getAttribute('aria-label')!.endsWith('chưa trả lời'))).toBe(true);
    expect(text(el.querySelector('[aria-live]'))).toBe('Đã xoá các câu trả lời, làm lại từ câu 1.');
  });

  it('keeps the answers when they cannot be forgotten', async () => {
    await setup(quiz(allAnswered()));
    await click(button('Làm lại tất cả'));
    http.expectOne('/lessons/l1/answers').flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
    await settle();
    expect(text(el.querySelector('.form-alert'))).toBe('Chưa làm lại được, vui lòng thử lại.');
    expect(text(el.querySelector('.score'))).toBe('Kết quả 2/3 đúng');
  });

  it('does not report completion in review mode, and has no button on the result without a label', async () => {
    await setup(quiz([answer(0, 0, 0), answer(1, 1, 1)]), { review: true });
    await answerShown(2, 2);
    expect(scores).toEqual([]);
    await click(main());
    expect(el.querySelectorAll('.bottom-bar .btn-primary')).toHaveLength(0);
  });

  it('shows the stored answer when the question was answered elsewhere', async () => {
    await setup(quiz());
    await pick(0);
    await respond({ error: 'already_answered', message: 'x', answer: answer(0, 2, 0) }, 409);
    expect(label()).toBe('Câu 1/3 · ✗ Sai rồi');
    expect(text(el.querySelector('.results .is-wrong'))).toContain('School');
  });

  it('asks to reload when the questions changed', async () => {
    await setup(quiz());
    await pick(0);
    await respond({ error: 'quiz_changed', message: 'x' }, 409);
    expect(text(el.querySelector('.changed'))).toContain('Câu hỏi vừa được cập nhật');
    expect(main().disabled).toBe(true);
    await click(button('Tải lại'));
    expect(reloads).toBe(1);
  });

  it('unlocks the options after a network error', async () => {
    await setup(quiz());
    await pick(0);
    await respond({ error: 'internal_error', message: 'x' }, 500);
    expect(text(el.querySelector('.form-alert'))).toBe('Không gửi được câu trả lời, vui lòng thử lại.');
    expect(options().some((o) => o.disabled)).toBe(false);
    expect(main().textContent?.trim()).toBe('Kiểm tra');
  });

  describe('kinds and shuffled options', () => {
    // With Math.random() at 0 the shuffle turns [0, 1, 2, 3] into [1, 2, 3, 0].
    const setupKinded = async (answers: QuizAnswer[] = []) => {
      vi.spyOn(Math, 'random').mockReturnValue(0);
      await setup(kinded(answers));
    };

    it('labels each tab with its number and kind, and shows the Vietnamese under the prompt', async () => {
      await setupKinded();
      expect(tabs().map((t) => text(t))).toEqual(['1 Ý chính', '2 Đ/S/K', '3 Nghĩa từ']);
      expect(tabs()[0].getAttribute('aria-label')).toBe('Câu 1 · Ý chính: chưa trả lời');
      expect(label()).toBe('Câu 1/3 · Ý chính của bài');
      expect(prompt()).toBe('What is the text about?');
      expect(el.querySelector('.prompt')!.getAttribute('lang')).toBe('en');
      expect(text(el.querySelector('.prompt-vi'))).toBe('Bài nói về điều gì?');
    });

    it('shuffles the options but sends the original index, and marks the answer where it is shown', async () => {
      await setupKinded();
      expect(optionTexts()).toEqual(['Sport', 'Food', 'Work', 'Music']);
      await pick(0); // Sport, the second option on the server
      const req = await respond({ answer: answer(0, 1, 0), answered: 1, total: 3, correct: 0 });
      expect(req.request.body).toEqual({ version: 4, questionIndex: 0, choice: 1 });
      const results = Array.from(el.querySelectorAll('.results .result'));
      expect(results.map((r) => text(r.querySelector('.result-text')))).toEqual(['Sport', 'Food', 'Work', 'Music']);
      expect(results[0].classList).toContain('is-wrong');
      expect(results[3].classList).toContain('is-answer');
      expect(text(results[3])).toContain('Đáp án đúng');
    });

    it('sends the original index of a right answer picked at a shuffled place', async () => {
      await setupKinded();
      await pick(3); // Music
      const req = await respond({ answer: answer(0, 0, 0), answered: 1, total: 3, correct: 1 });
      expect(req.request.body.choice).toBe(0);
      expect(text(el.querySelector('.results .is-answer'))).toContain('Music');
      expect(text(el.querySelector('.results .is-answer'))).toContain('Bạn chọn đúng');
    });

    it('keeps the true / false / not given options in their fixed order', async () => {
      await setupKinded();
      await click(tabs()[1]);
      expect(label()).toBe('Câu 2/3 · Đúng, sai hay không có trong bài?');
      expect(optionTexts()).toEqual(['Đúng', 'Sai', 'Không có trong bài']);
      await pick(2);
      const req = await respond({ answer: answer(1, 2, 1), answered: 1, total: 3, correct: 0 });
      expect(req.request.body.choice).toBe(2);
      expect(text(el.querySelectorAll('.results .result')[1])).toContain('Đáp án đúng');
    });

    it('quotes the text over a vocab question asked in Vietnamese', async () => {
      await setupKinded();
      await click(tabs()[2]);
      expect(text(el.querySelector('lu-asker .quote'))).toBe('“…but sometimes it is quiet.”');
      expect(el.querySelector('.prompt')!.getAttribute('lang')).toBe('vi');
      expect(el.querySelector('.prompt-vi')).toBeNull();
      expect(optionTexts()).toEqual(['nhỏ, khẽ', 'vắng vẻ', 'ít nói', 'chậm']);
    });

    it('gives the result by kind', async () => {
      await setupKinded([answer(0, 0, 0), answer(1, 2, 1), answer(2, 1, 1)]);
      expect(text(el.querySelector('.score'))).toBe('Kết quả 2/3 đúng');
      const marks = Array.from(el.querySelectorAll('.kinds [aria-hidden="true"]')).map((m) => text(m));
      expect(marks.join(' ')).toBe('Ý chính ✓ · Đ/S/K ✗ · Nghĩa từ ✓');
      expect(text(el.querySelector('.kinds .visually-hidden'))).toBe('Ý chính: 1/1 đúng,');
    });

    it('keeps the order of questions without a kind, with no result by kind', async () => {
      vi.spyOn(Math, 'random').mockReturnValue(0);
      await setup(quiz(allAnswered()));
      expect(el.querySelector('.kinds')).toBeNull();
      await click(el.querySelector<HTMLButtonElement>('.score-row'));
      expect(text(el.querySelector('.results .result-text'))).toBe('Park');
    });
  });
});

describe('optionOrder', () => {
  const q = (kind?: '' | 'gist' | 'tfn') => ({ kind, prompt: 'p', options: ['a', 'b', 'c', 'd'] });

  it('shuffles gist, detail and vocab options', () => {
    expect(optionOrder(q('gist'), () => 0)).toEqual([1, 2, 3, 0]);
    expect(optionOrder(q('gist'), () => 0.999).sort()).toEqual([0, 1, 2, 3]);
  });

  it('keeps true / false / not given and older questions in order', () => {
    expect(optionOrder(q('tfn'), () => 0)).toEqual([0, 1, 2, 3]);
    expect(optionOrder(q(''), () => 0)).toEqual([0, 1, 2, 3]);
    expect(optionOrder(q(), () => 0)).toEqual([0, 1, 2, 3]);
  });
});
