import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GrammarExercise } from '../../../core/models/grammar-study';
import { GrammarExerciseView } from './grammar-exercise';

const choice: GrammarExercise = {
  id: 'p1', kind: 'choice', promptVi: 'Chọn đáp án đúng', text: 'She ___ a teacher.',
  options: ['am', 'is', 'are', 'be'], answerIndex: 1, explanationVi: 'She đi với is.',
};
const fill: GrammarExercise = {
  id: 'p2', kind: 'fill', text: 'They ___ happy.', answers: ['are'], explanationVi: 'They đi với are.',
};
const reorder: GrammarExercise = {
  id: 'p3', kind: 'reorder', text: 'Cô ấy là giáo viên.', sentence: 'She is a teacher',
  words: ['She', 'is', 'a', 'teacher', 'are'], explanationVi: 'Chủ ngữ rồi động từ.',
};

describe('GrammarExerciseView', () => {
  let fixture: ComponentFixture<GrammarExerciseView>;
  let el: HTMLElement;
  let answers: boolean[];

  const setup = async (exercise: GrammarExercise, reveal = true) => {
    fixture = TestBed.createComponent(GrammarExerciseView);
    fixture.componentRef.setInput('exercise', exercise);
    fixture.componentRef.setInput('reveal', reveal);
    answers = [];
    fixture.componentInstance.answered.subscribe((a) => answers.push(a));
    el = fixture.nativeElement;
    await fixture.whenStable();
  };
  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const feedback = () => text(el.querySelector('.feedback'));
  const buttons = () => Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
  const byText = (t: string) => buttons().find((b) => text(b) === t)!;

  beforeEach(() => TestBed.configureTestingModule({ imports: [GrammarExerciseView] }));

  it('grades a choice at once and says Đúng with the explanation', async () => {
    await setup(choice);
    expect(text(el.querySelector('.prompt'))).toBe('Chọn đáp án đúng');
    expect(el.querySelectorAll('.option').length).toBe(4);
    byText('is').click();
    await fixture.whenStable();
    expect(answers).toEqual([true]);
    expect(feedback()).toContain('Đúng');
    expect(feedback()).toContain('She đi với is.');
    expect(buttons().every((b) => b.disabled)).toBe(true);
  });

  it('says Sai in words and marks the right option for a wrong choice', async () => {
    await setup(choice);
    byText('are').click();
    await fixture.whenStable();
    expect(answers).toEqual([false]);
    expect(feedback()).toContain('Sai');
    expect(text(el.querySelector('.is-wrong'))).toContain('Bạn chọn');
    expect(text(el.querySelector('.is-right'))).toContain('Đáp án đúng');
  });

  it('checks a fill ignoring case and spaces, with Enter', async () => {
    await setup(fill);
    const input = el.querySelector<HTMLInputElement>('input.blank')!;
    input.value = '  ARE ';
    input.dispatchEvent(new Event('input'));
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    await fixture.whenStable();
    expect(answers).toEqual([true]);
    expect(feedback()).toContain('Đúng');
    expect(input.disabled).toBe(true);
  });

  it('shows the right answer after a wrong fill and ignores an empty one', async () => {
    await setup(fill);
    expect(byText('Kiểm tra').disabled).toBe(true);
    const input = el.querySelector<HTMLInputElement>('input.blank')!;
    input.value = 'is';
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    byText('Kiểm tra').click();
    await fixture.whenStable();
    expect(answers).toEqual([false]);
    expect(feedback()).toContain('Sai');
    expect(feedback()).toContain('Đáp án đúng: are');
  });

  it('builds a reorder from tiles, can undo, and grades on Kiểm tra', async () => {
    await setup(reorder);
    const tile = (w: string) => Array.from(el.querySelectorAll<HTMLButtonElement>('ul.tiles:not(.answer-area) .tile')).find((b) => text(b) === w)!;
    expect(el.querySelectorAll('ul.tiles:not(.answer-area) .tile').length).toBe(5);
    for (const w of ['is', 'She']) {
      tile(w).click();
      await fixture.whenStable();
    }
    expect(tile('She').disabled).toBe(true);
    byText('Làm lại').click();
    await fixture.whenStable();
    expect(el.querySelectorAll('.answer-area .tile').length).toBe(0);
    for (const w of ['She', 'is', 'a', 'teacher']) {
      tile(w).click();
      await fixture.whenStable();
    }
    byText('Kiểm tra').click();
    await fixture.whenStable();
    expect(answers).toEqual([true]);
    expect(feedback()).toContain('Đúng');
  });

  it('grades a wrong reorder and shows the sentence', async () => {
    await setup(reorder);
    const tile = (w: string) => Array.from(el.querySelectorAll<HTMLButtonElement>('ul.tiles:not(.answer-area) .tile')).find((b) => text(b) === w)!;
    for (const w of ['is', 'She']) {
      tile(w).click();
      await fixture.whenStable();
    }
    byText('Kiểm tra').click();
    await fixture.whenStable();
    expect(answers).toEqual([false]);
    expect(feedback()).toContain('Sai');
    expect(feedback()).toContain('She is a teacher');
  });

  it('without reveal only records the answer', async () => {
    await setup(choice, false);
    byText('are').click();
    await fixture.whenStable();
    expect(answers).toEqual([false]);
    expect(feedback()).toBe('Đã ghi nhận câu trả lời.');
    expect(el.querySelector('.is-right')).toBeNull();
  });

  it('offers "Báo lỗi câu này" after the answer, only with a point and reveal', async () => {
    await setup(choice);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    byText('are').click();
    await fixture.whenStable();
    expect(el.querySelector('lu-report-exercise')).not.toBeNull();
    expect(byText('Báo lỗi câu này')).toBeDefined();
  });

  it('shows no report button before the answer, without a point, or in a test', async () => {
    await setup(choice);
    byText('are').click();
    await fixture.whenStable();
    expect(el.querySelector('lu-report-exercise')).toBeNull();

    await setup(choice, false);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    expect(el.querySelector('lu-report-exercise')).toBeNull();
    byText('are').click();
    await fixture.whenStable();
    expect(el.querySelector('lu-report-exercise')).toBeNull();
  });

  it('tells what the learner answered before answered', async () => {
    await setup(fill);
    const given: string[] = [];
    fixture.componentInstance.given.subscribe((g) => given.push(g));
    const input = el.querySelector<HTMLInputElement>('input.blank')!;
    input.value = ' is ';
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    byText('Kiểm tra').click();
    await fixture.whenStable();
    expect(given).toEqual(['is']);
  });

  it('starts afresh when the exercise changes', async () => {
    await setup(choice);
    byText('is').click();
    await fixture.whenStable();
    fixture.componentRef.setInput('exercise', { ...choice, id: 'p9' });
    await fixture.whenStable();
    expect(feedback()).toBe('');
    expect(buttons().some((b) => b.disabled)).toBe(false);
  });
});
