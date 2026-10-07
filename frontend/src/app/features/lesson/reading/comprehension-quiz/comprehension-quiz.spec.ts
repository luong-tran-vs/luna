import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { Quiz, QuizAnswer } from '../../../../core/models/reading';
import { ComprehensionQuiz, QuizScore } from './comprehension-quiz';

const quiz = (answers: QuizAnswer[] = []): Quiz => ({
  version: 3,
  questions: [
    { prompt: 'Where did we go?', options: ['Park', 'Home', 'School', 'Work'] },
    { prompt: 'What did he give up?', options: ['Tea', 'Smoking', 'Sport', 'Work'] },
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

describe('ComprehensionQuiz', () => {
  let fixture: ComponentFixture<ComprehensionQuiz>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let scores: QuizScore[];
  let reloads: number;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const options = () => Array.from(el.querySelectorAll<HTMLInputElement>('.question.active .option input'));
  const optionTexts = () =>
    Array.from(el.querySelectorAll('.question.active .option [lang="en"]')).map((o) => text(o));
  /** Picks an option, then Kiểm tra. */
  const pick = async (j: number) => {
    options()[j].click();
    await fixture.whenStable();
    el.querySelector<HTMLButtonElement>('.step-nav .check')!.click();
  };
  const setup = async (q: Quiz, review = false) => {
    await TestBed.configureTestingModule({
      imports: [ComprehensionQuiz],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ComprehensionQuiz);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('quiz', q);
    fixture.componentRef.setInput('review', review);
    scores = [];
    reloads = 0;
    fixture.componentInstance.allAnswered.subscribe((s) => scores.push(s));
    fixture.componentInstance.reload.subscribe(() => reloads++);
    el = fixture.nativeElement;
    await fixture.whenStable();
  };
  const respond = async (body: object, status = 200) => {
    const req = http.expectOne('/api/lessons/l1/answers');
    if (status === 200) {
      req.flush(body);
    } else {
      req.flush(body, { status, statusText: 'Error' });
    }
    await settle();
    return req;
  };

  afterEach(() => http.verify());

  const navButton = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.step-nav button')).find((b) => text(b) === label)!;

  it('shows only the first question with four options', async () => {
    await setup(quiz());
    expect(text(el.querySelector('.counter'))).toBe('Câu 1/2');
    expect(text(el.querySelector('.question.active .prompt'))).toBe('Where did we go?');
    expect(optionTexts()).toEqual(['Park', 'Home', 'School', 'Work']);
    expect(el.textContent).not.toContain('What did he give up?');
    expect(navButton('Kiểm tra').disabled).toBe(true);
    expect(navButton('Câu trước').disabled).toBe(true);
  });

  it('locks the question while sending, then keeps it on screen with the result in words', async () => {
    await setup(quiz());
    await pick(1);
    await settle();
    expect(options().every((o) => o.disabled)).toBe(true);
    const req = await respond({ answer: answer(0, 1, 0), answered: 1, total: 2, correct: 0 });
    expect(req.request.body).toEqual({ version: 3, questionIndex: 0, choice: 1 });

    const done = el.querySelector('.question.answered')!;
    expect(text(done.querySelector('.counter'))).toBe('Câu 1/2');
    expect(text(done.querySelector('.verdict'))).toBe('✗ Sai');
    const results = Array.from(done.querySelectorAll('.result')).map((r) => text(r));
    expect(results[0]).toBe('Park — Đáp án đúng');
    expect(results[1]).toBe('Home (bạn chọn)');
    expect(text(done.querySelector('.explanation'))).toBe('Giải thích 1');
    expect(Array.from(done.querySelectorAll('button')).map((b) => text(b))).toEqual(['Trả lời lại câu này']);
    expect(text(el.querySelector('[aria-live="polite"]'))).toBe('Câu 1: Sai.');

    // Câu tiếp theo shows the next question.
    navButton('Câu tiếp theo').click();
    await fixture.whenStable();
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2');
    expect(text(el.querySelector('.question.active .prompt'))).toBe('What did he give up?');
  });

  it('moves between the questions with ← and →, answered ones showing their result', async () => {
    await setup(quiz([answer(0, 0, 0)]));
    // Opens on the first question not answered.
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2');
    expect(navButton('Câu sau').disabled).toBe(true);
    navButton('Câu trước').click();
    await fixture.whenStable();
    expect(text(el.querySelector('.question.answered .verdict'))).toBe('✓ Đúng');
    expect(navButton('Câu tiếp theo')).toBeDefined();
    navButton('Câu sau').click();
    await fixture.whenStable();
    expect(el.querySelector('.question.active')).not.toBeNull();
  });

  it('reports the score once after the last answer', async () => {
    await setup(quiz([answer(0, 0, 0)]));
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2');
    await pick(1);
    await settle();
    await respond({ answer: answer(1, 1, 1), answered: 2, total: 2, correct: 2 });
    expect(text(el.querySelector('.summary'))).toBe('Đúng 2/2 câu');
    expect(scores).toEqual([{ correct: 2, total: 2 }]);
    expect(text(el.querySelector('.verdict'))).toBe('✓ Đúng');
    expect(navButton('Đã trả lời hết').disabled).toBe(true);
  });

  it('resumes with every answer already given and does not report again', async () => {
    await setup(quiz([answer(0, 0, 0), answer(1, 2, 1)]));
    expect(options()).toEqual([]);
    expect(text(el.querySelector('.summary'))).toBe('Đúng 1/2 câu');
    expect(scores).toEqual([]);
  });

  it('does not report completion in review mode', async () => {
    await setup(quiz([answer(0, 0, 0)]), true);
    await pick(0);
    await settle();
    await respond({ answer: answer(1, 0, 1), answered: 2, total: 2, correct: 1 });
    expect(text(el.querySelector('.summary'))).toBe('Đúng 1/2 câu');
    expect(scores).toEqual([]);
  });

  it('unlocks the options after a network error', async () => {
    await setup(quiz());
    await pick(0);
    await settle();
    http.expectOne('/api/lessons/l1/answers').error(new ProgressEvent('error'));
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Không gửi được câu trả lời, vui lòng thử lại.');
    expect(options().every((o) => !o.disabled)).toBe(true);
  });

  it('shows the stored answer when the question was answered elsewhere', async () => {
    await setup(quiz());
    await pick(3);
    await settle();
    await respond({ error: 'already_answered', message: 'Câu này đã được trả lời', answer: answer(0, 0, 0) }, 409);
    expect(text(el.querySelector('.verdict'))).toBe('✓ Đúng');
    expect(el.querySelector('[role="alert"]')).toBeNull();
  });

  it('answers a question again: the new answer replaces the result', async () => {
    await setup(quiz([answer(0, 1, 0)]));
    navButton('Câu trước').click();
    await fixture.whenStable();
    expect(text(el.querySelector('.question.answered .verdict'))).toBe('✗ Sai');
    Array.from(el.querySelectorAll<HTMLButtonElement>('.question.answered button'))
      .find((b) => text(b) === 'Trả lời lại câu này')!
      .click();
    await fixture.whenStable();
    expect(el.querySelector('.question.answered')).toBeNull();
    expect(optionTexts()).toEqual(['Park', 'Home', 'School', 'Work']);
    await pick(0);
    await settle();
    const req = await respond({ answer: answer(0, 0, 0), answered: 1, total: 2, correct: 1 });
    expect(req.request.body).toEqual({ version: 3, questionIndex: 0, choice: 0 });
    expect(text(el.querySelector('.question.answered .verdict'))).toBe('✓ Đúng');
  });

  it('starts every question again with Làm lại tất cả câu hỏi', async () => {
    await setup(quiz([answer(0, 1, 0), answer(1, 1, 1)]));
    const restart = () =>
      Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === 'Làm lại tất cả câu hỏi');
    restart()!.click();
    await settle();
    const req = http.expectOne('/api/lessons/l1/answers');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expect(text(el.querySelector('.question.active .counter'))).toBe('Câu 1/2');
    expect(optionTexts()).toEqual(['Park', 'Home', 'School', 'Work']);
    expect(restart()).toBeUndefined();
  });

  it('keeps the answers when they cannot be forgotten', async () => {
    await setup(quiz([answer(0, 1, 0)]));
    Array.from(el.querySelectorAll<HTMLButtonElement>('button'))
      .find((b) => text(b) === 'Làm lại tất cả câu hỏi')!
      .click();
    await settle();
    http.expectOne('/api/lessons/l1/answers').flush('down', { status: 500, statusText: 'Error' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Chưa làm lại được, vui lòng thử lại.');
    navButton('Câu trước').click();
    await fixture.whenStable();
    expect(text(el.querySelector('.question.answered .verdict'))).toBe('✗ Sai');
  });

  it('asks to reload when the questions changed', async () => {
    await setup(quiz());
    await pick(0);
    await settle();
    await respond({ error: 'quiz_changed', message: 'Câu hỏi vừa được cập nhật, vui lòng tải lại' }, 409);
    expect(text(el.querySelector('.changed p'))).toBe('Câu hỏi vừa được cập nhật.');
    expect(options().every((o) => o.disabled)).toBe(true);
    el.querySelector<HTMLButtonElement>('.changed button')!.click();
    expect(reloads).toBe(1);
  });
});
