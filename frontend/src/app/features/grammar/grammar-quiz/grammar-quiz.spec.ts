import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AttemptResult, GrammarExercise, GrammarProgress } from '../../../core/models/grammar-study';
import { GrammarQuiz } from './grammar-quiz';

const choice = (id: string): GrammarExercise => ({
  id, kind: 'choice', text: 'x ___', options: ['a', 'b', 'c', 'd'], answerIndex: 0, explanationVi: `vì ${id}`,
});
const progress = (over: Partial<GrammarProgress> = {}): GrammarProgress => ({
  status: 'learning', practiceAttempts: 1, lastPractice: 50, masteryAttempts: 0, bestMastery: 0, mastered: false,
  weak: [], ...over,
});

describe('GrammarQuiz', () => {
  let fixture: ComponentFixture<GrammarQuiz>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let recorded: AttemptResult[];

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
  const settle = () => fixture.whenStable();
  const answer = async (option: string, then: string) => {
    button(option).click();
    await settle();
    button(then).click();
    await settle();
  };

  const setup = async (kind: 'practice' | 'mastery', ids: string[], weak: string[] = []) => {
    TestBed.configureTestingModule({
      imports: [GrammarQuiz],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarQuiz);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    fixture.componentRef.setInput('kind', kind);
    fixture.componentRef.setInput('exercises', ids.map(choice));
    fixture.componentRef.setInput('weak', weak);
    recorded = [];
    fixture.componentInstance.recorded.subscribe((r) => recorded.push(r));
    el = fixture.nativeElement;
    await settle();
  };

  afterEach(() => http.verify());

  it('runs a practice round with feedback, then posts the score and offers the wrong ones again', async () => {
    await setup('practice', ['p1', 'p2']);
    expect(text(el.querySelector('.count'))).toBe('Câu 1/2');
    await answer('a', 'Câu tiếp theo');
    expect(text(el.querySelector('.count'))).toBe('Câu 2/2');
    button('b').click();
    await settle();
    expect(text(el.querySelector('.feedback'))).toContain('Sai');
    button('Xem kết quả').click();
    await settle();

    const req = http.expectOne('/grammar/a1-to-be/attempts');
    expect(req.request.body).toEqual({ kind: 'practice', correct: 1, total: 2, wrong: ['p2'] });
    req.flush({ progress: progress({ weak: ['p2'] }), passed: false });
    await settle();
    expect(recorded.length).toBe(1);
    expect(text(el.querySelector('.score'))).toBe('Đúng 1/2 câu (50%)');

    button('Luyện lại các câu sai (1)').click();
    await settle();
    expect(text(el.querySelector('.count'))).toBe('Câu 1/1');
    expect(text(el)).toContain('không tính vào điểm');
    await answer('a', 'Xem kết quả');
    http.expectNone('/grammar/a1-to-be/attempts');
    expect(text(el.querySelector('.score'))).toBe('Đúng 1/1 câu (100%)');
  });

  it('offers to redo the weak exercises before a round starts', async () => {
    await setup('practice', ['p1', 'p2', 'p3'], ['p3', 'zz']);
    button('Luyện lại các câu sai (1)').click();
    await settle();
    expect(text(el.querySelector('.count'))).toBe('Câu 1/1');
    expect(el.querySelector('.weak')).toBeNull();
  });

  it('hides the right and wrong answers in a mastery test and shows Đã nắm vững when passed', async () => {
    await setup('mastery', ['m1', 'm2']);
    button('b').click();
    await settle();
    expect(text(el.querySelector('.feedback'))).toBe('Đã ghi nhận câu trả lời.');
    button('Câu tiếp theo').click();
    await settle();
    await answer('a', 'Xem kết quả');
    const req = http.expectOne('/grammar/a1-to-be/attempts');
    expect(req.request.body.kind).toBe('mastery');
    req.flush({ progress: progress({ status: 'mastered', mastered: true, bestMastery: 100 }), passed: true });
    await settle();
    expect(text(el.querySelector('.verdict'))).toContain('Đã nắm vững ✓');
  });

  it('reviews every question after a test, wrong ones first, with a report button each', async () => {
    await setup('mastery', ['m1', 'm2']);
    await answer('a', 'Câu tiếp theo');
    expect(el.querySelector('lu-report-exercise')).toBeNull();
    button('b').click();
    await settle();
    expect(el.querySelector('lu-report-exercise')).toBeNull();
    button('Xem kết quả').click();
    await settle();
    http.expectOne('/grammar/a1-to-be/attempts').flush({ progress: progress(), passed: false });
    await settle();

    const items = Array.from(el.querySelectorAll('.review-item'));
    expect(items.length).toBe(2);
    expect(text(items[0])).toContain('Câu 2: Sai');
    expect(text(items[0])).toContain('Bạn trả lời: b');
    expect(text(items[0])).toContain('Đáp án đúng: a');
    expect(text(items[0])).toContain('vì m2');
    expect(text(items[1])).toContain('Câu 1: Đúng');
    expect(text(items[1])).not.toContain('Đáp án đúng');
    expect(el.querySelectorAll('.review-item lu-report-exercise').length).toBe(2);
  });

  it('says how many answers are missing when the test is not passed, and can be retried', async () => {
    await setup('mastery', ['m1', 'm2', 'm3', 'm4', 'm5']);
    await answer('a', 'Câu tiếp theo');
    await answer('a', 'Câu tiếp theo');
    await answer('a', 'Câu tiếp theo');
    await answer('b', 'Câu tiếp theo');
    await answer('b', 'Xem kết quả');
    http.expectOne('/grammar/a1-to-be/attempts').flush({ progress: progress(), passed: false });
    await settle();
    expect(text(el.querySelector('.score'))).toBe('Đúng 3/5 câu (60%)');
    expect(text(el.querySelector('.verdict'))).toContain('còn thiếu 1 câu đúng');
    button('Làm bài kiểm tra lại').click();
    await settle();
    expect(text(el.querySelector('.count'))).toBe('Câu 1/5');
  });

  it('keeps the local score and lets the learner send it again when saving fails', async () => {
    await setup('practice', ['p1']);
    await answer('a', 'Xem kết quả');
    http.expectOne('/grammar/a1-to-be/attempts').flush({ error: 'x' }, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không lưu được điểm');
    button('Gửi lại điểm').click();
    http.expectOne('/grammar/a1-to-be/attempts').flush({ progress: progress(), passed: false });
    await settle();
    expect(el.querySelector('[role="alert"]')).toBeNull();
    expect(recorded.length).toBe(1);
  });
});
