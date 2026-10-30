import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GrammarMistake } from '../../../core/models/grammar-study';
import { MistakeCheck } from './mistake-check';

const mistakes: GrammarMistake[] = [
  { wrong: 'She are a teacher.', right: 'She is a teacher.', noteVi: 'She đi với is.' },
  { wrong: 'I is happy.', right: 'I am happy.', noteVi: '' },
];

describe('MistakeCheck', () => {
  let fixture: ComponentFixture<MistakeCheck>;
  let el: HTMLElement;
  let check: MistakeCheck;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const choices = () => Array.from(el.querySelectorAll<HTMLButtonElement>('.choice'));
  const choice = (sentence: string) => choices().find((b) => text(b).startsWith(sentence))!;
  const settle = () => fixture.whenStable();

  beforeEach(async () => {
    TestBed.configureTestingModule({ imports: [MistakeCheck] });
    fixture = TestBed.createComponent(MistakeCheck);
    fixture.componentRef.setInput('mistakes', mistakes);
    el = fixture.nativeElement;
    check = fixture.componentInstance;
    await settle();
  });

  it('asks which of the two sentences is right, the cat nudging before a pick', () => {
    expect(text(el.querySelector('.count'))).toBe('1 / 2');
    expect(text(el.querySelector('.question'))).toBe('Câu nào đúng?');
    expect(choices().map((b) => text(b)).sort()).toEqual(['She are a teacher.', 'She is a teacher.']);
    expect(text(el.querySelector('.bubble'))).toContain('hay sai');
    expect(el.querySelector('.why')).toBeNull();
    expect(check.answered()).toBe(false);
  });

  it('marks a right pick and shows the note, once', async () => {
    choice('She is a teacher.').click();
    await settle();
    expect(check.answered()).toBe(true);
    expect(choice('She is a teacher.').classList).toContain('is-right');
    expect(choice('She is a teacher.').getAttribute('aria-pressed')).toBe('true');
    expect(text(el.querySelector('.why'))).toBe('Chính xác!She đi với is.');
    expect(el.querySelector('.bubble')).toBeNull();

    choice('She are a teacher.').click();
    await settle();
    expect(choice('She are a teacher.').classList).not.toContain('is-wrong');
  });

  it('marks a wrong pick with the right sentence, and moves on when told', async () => {
    choice('She are a teacher.').click();
    await settle();
    expect(choice('She are a teacher.').classList).toContain('is-wrong');
    expect(choice('She is a teacher.').classList).toContain('is-right');
    expect(text(el.querySelector('.why'))).toBe('Chưa đúng rồiCâu đúng: She is a teacher.She đi với is.');

    expect(check.isLast()).toBe(false);
    check.next();
    await settle();
    expect(text(el.querySelector('.count'))).toBe('2 / 2');
    expect(check.answered()).toBe(false);
    expect(check.isLast()).toBe(true);
    expect(document.activeElement?.className).toContain('question');
  });
});
