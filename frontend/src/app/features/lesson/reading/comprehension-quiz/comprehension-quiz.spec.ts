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
  const options = () => Array.from(el.querySelectorAll<HTMLButtonElement>('.question.active .option'));
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

  it('shows the first question with four options', async () => {
    await setup(quiz());
    expect(text(el.querySelector('.counter'))).toBe('Câu 1/2');
    expect(text(el.querySelector('.question.active .prompt'))).toBe('Where did we go?');
    expect(options().map((o) => text(o))).toEqual(['Park', 'Home', 'School', 'Work']);
    expect(el.textContent).not.toContain('What did he give up?');
  });

  it('locks the question while sending and shows the result in words', async () => {
    await setup(quiz());
    options()[1].click();
    await settle();
    expect(options().every((o) => o.disabled)).toBe(true);
    const req = await respond({ answer: answer(0, 1, 0), answered: 1, total: 2, correct: 0 });
    expect(req.request.body).toEqual({ version: 3, questionIndex: 0, choice: 1 });

    const done = el.querySelector('.question.answered')!;
    expect(text(done.querySelector('.verdict'))).toBe('✗ Sai');
    const results = Array.from(done.querySelectorAll('.result')).map((r) => text(r));
    expect(results[0]).toBe('Park — Đáp án đúng');
    expect(results[1]).toBe('Home (bạn chọn)');
    expect(text(done.querySelector('.explanation'))).toBe('Giải thích 1');
    expect(done.querySelector('button')).toBeNull();
    expect(text(el.querySelector('[aria-live="polite"]'))).toBe('Câu 1: Sai.');

    // The next question appears.
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2');
  });

  it('reports the score once after the last answer', async () => {
    await setup(quiz([answer(0, 0, 0)]));
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2');
    options()[1].click();
    await settle();
    await respond({ answer: answer(1, 1, 1), answered: 2, total: 2, correct: 2 });
    expect(text(el.querySelector('.summary'))).toBe('Đúng 2/2 câu');
    expect(scores).toEqual([{ correct: 2, total: 2 }]);
    expect(el.querySelectorAll('.verdict').length).toBe(2);
    expect(text(el.querySelectorAll('.verdict')[1])).toBe('✓ Đúng');
  });

  it('resumes with every answer already given and does not report again', async () => {
    await setup(quiz([answer(0, 0, 0), answer(1, 2, 1)]));
    expect(options()).toEqual([]);
    expect(text(el.querySelector('.summary'))).toBe('Đúng 1/2 câu');
    expect(scores).toEqual([]);
  });

  it('does not report completion in review mode', async () => {
    await setup(quiz([answer(0, 0, 0)]), true);
    options()[0].click();
    await settle();
    await respond({ answer: answer(1, 0, 1), answered: 2, total: 2, correct: 1 });
    expect(text(el.querySelector('.summary'))).toBe('Đúng 1/2 câu');
    expect(scores).toEqual([]);
  });

  it('unlocks the options after a network error', async () => {
    await setup(quiz());
    options()[0].click();
    await settle();
    http.expectOne('/api/lessons/l1/answers').error(new ProgressEvent('error'));
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Không gửi được câu trả lời, vui lòng thử lại.');
    expect(options().every((o) => !o.disabled)).toBe(true);
  });

  it('shows the stored answer when the question was answered elsewhere', async () => {
    await setup(quiz());
    options()[3].click();
    await settle();
    await respond({ error: 'already_answered', message: 'Câu này đã được trả lời', answer: answer(0, 0, 0) }, 409);
    expect(text(el.querySelector('.verdict'))).toBe('✓ Đúng');
    expect(el.querySelector('[role="alert"]')).toBeNull();
  });

  it('asks to reload when the questions changed', async () => {
    await setup(quiz());
    options()[0].click();
    await settle();
    await respond({ error: 'quiz_changed', message: 'Câu hỏi vừa được cập nhật, vui lòng tải lại' }, 409);
    expect(text(el.querySelector('.changed p'))).toBe('Câu hỏi vừa được cập nhật.');
    expect(options().every((o) => o.disabled)).toBe(true);
    el.querySelector<HTMLButtonElement>('.changed button')!.click();
    expect(reloads).toBe(1);
  });
});
