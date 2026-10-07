import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeFill } from '../../../../core/models/practice';
import { Score } from '../practice-logic';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { FillStep } from './fill-step';

const fill: PracticeFill = {
  turns: [
    {
      speaker: 0,
      turnIndex: 0,
      meaningVi: 'Xin chào, tên mình là Minh.',
      parts: [{ text: 'Hello, my ' }, { blank: 0 }, { text: ' is ' }, { blank: 1 }, { text: '.' }],
    },
    {
      speaker: 1,
      turnIndex: 2,
      meaningVi: 'Rất vui được gặp bạn.',
      parts: [{ text: 'Nice to ' }, { blank: 2 }, { text: ' you!' }],
    },
  ],
  blanks: [{ answer: 'name' }, { answer: 'Minh' }, { answer: 'meet' }],
  wordBank: ['name', 'Minh', 'meet', 'from'],
};

describe('FillStep', () => {
  let fixture: ComponentFixture<FillStep>;
  let el: HTMLElement;
  let played: string[];
  let results: Score[];
  let missed: string[][];
  let done: number;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const blank = (n: number) => el.querySelector<HTMLInputElement>(`#fill-blank-${n}`)!;
  const bankButtons = () =>
    Array.from(
      el.querySelectorAll<HTMLButtonElement>('ul[aria-labelledby="fill-bank-heading"] .tile'),
    );
  const bankButton = (word: string, nth = 0) => bankButtons().filter((b) => text(b) === word)[nth];
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
  const checkButton = () => button('Kiểm tra')!;
  const status = () => text(el.querySelector('[role="status"]'));
  const click = async (b: HTMLButtonElement) => {
    b.click();
    await fixture.whenStable();
  };
  const type = async (n: number, value: string) => {
    const input = blank(n);
    input.dispatchEvent(new Event('focus'));
    input.value = value;
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const enter = async (n: number) => {
    blank(n).dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    await fixture.whenStable();
  };

  const render = async (
    bank = ['from', 'meet', 'Minh', 'name'],
    f: PracticeFill = fill,
    tip = 'Dùng "Nice to meet you".',
  ) => {
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(new FakeSpeech())] });
    fixture = TestBed.createComponent(FillStep);
    fixture.componentRef.setInput('fill', f);
    fixture.componentRef.setInput('bank', bank);
    fixture.componentRef.setInput('speakers', ['Minh', 'Anna']);
    fixture.componentRef.setInput('turnTexts', [
      'Hello, my name is Minh.',
      'Hi.',
      'Nice to meet you!',
    ]);
    fixture.componentRef.setInput('grammarTip', tip);
    played = [];
    results = [];
    missed = [];
    done = 0;
    fixture.componentInstance.readAloud.subscribe((t) => played.push(t));
    fixture.componentInstance.checked.subscribe((r) => results.push(r));
    fixture.componentInstance.missed.subscribe((w) => missed.push(w));
    fixture.componentInstance.done.subscribe(() => done++);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows one line at a time with its speaker, a text input per blank and the meaning', async () => {
    await render();
    expect(el.querySelectorAll('.turn')).toHaveLength(1);
    expect(text(el.querySelector('.turn .speaker'))).toBe('Minh');
    expect(text(el.querySelector('.step-card-head .muted'))).toBe('Câu 1/2');
    expect(text(el.querySelector('.meaning'))).toBe('Xin chào, tên mình là Minh.');
    expect(el.querySelectorAll('input.blank')).toHaveLength(2);
    expect(blank(0).type).toBe('text');
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1');
    expect(blank(0).classList).toContain('selected');
    // Same width for every blank: it does not give the answer away.
    expect(blank(0).getAttribute('size')).toBe(blank(1).getAttribute('size'));
    expect(bankButtons().map((b) => text(b))).toEqual(['from', 'meet', 'Minh', 'name']);

    await click(button('Câu sau')!);
    expect(text(el.querySelector('.step-card-head .muted'))).toBe('Câu 2/2');
    expect(text(el.querySelector('.line'))).toBe('Nice to you!');
    expect(button('Câu sau')!.disabled).toBe(true);
    await click(button('Câu trước')!);
    expect(text(el.querySelector('.step-card-head .muted'))).toBe('Câu 1/2');
    expect(button('Câu trước')!.disabled).toBe(true);
  });

  it('reads the whole turn of the line on screen through "Nghe câu"', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 1"]')!);
    await click(button('Câu sau')!);
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!);
    expect(played).toEqual(['Hello, my name is Minh.', 'Nice to meet you!']);
  });

  it('typing fills a blank and fades its word in the bank', async () => {
    await render();
    await type(0, 'Name ');
    expect(blank(0).classList).toContain('filled');
    expect(bankButton('name').disabled).toBe(true);
    await type(0, '');
    expect(bankButton('name').disabled).toBe(false);
  });

  it('a word of the bank goes into the selected blank of the line, then its next empty one', async () => {
    await render();
    await click(bankButton('name'));
    expect(blank(0).value).toBe('name');
    expect(bankButton('name').disabled).toBe(true);
    expect(blank(1).classList).toContain('selected');
    // The focused blank is filled: the next word goes to the line's next empty one.
    await type(0, 'name');
    await click(bankButton('Minh'));
    expect(blank(1).value).toBe('Minh');
    // Every blank of the line filled: the word replaces the focused one, never another line's.
    await type(0, 'name');
    await click(bankButton('meet'));
    expect(blank(0).value).toBe('meet');
    expect(bankButton('name').disabled).toBe(false);
    await click(button('Câu sau')!);
    expect(blank(2).value).toBe('');
  });

  it('Enter goes to the next blank of the line, and checks the line on its last one', async () => {
    await render();
    await type(0, 'name');
    await enter(0);
    expect(blank(1).classList).toContain('selected');
    await type(1, 'Minh');
    await enter(1);
    expect(status()).toBe('Chính xác!');
    // Only the first line is checked: no score yet.
    expect(results).toEqual([]);
  });

  it('keeps Kiểm tra disabled until every blank of the line is filled', async () => {
    await render();
    expect(checkButton().disabled).toBe(true);
    await type(0, 'name');
    expect(checkButton().disabled).toBe(true);
    await type(1, '   ');
    expect(checkButton().disabled).toBe(true);
    await type(1, 'Minh');
    // The other line's blank is still empty: it does not matter here.
    expect(checkButton().disabled).toBe(false);
  });

  it('checks line by line, tells the wrong words of each, then the score and the grammar tip', async () => {
    await render();
    await type(0, 'name');
    await type(1, 'Minh');
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual(['Đúng', 'Đúng']);
    // Locked after checking, the bank gone, the next line offered.
    expect(blank(0).readOnly).toBe(true);
    expect(bankButtons()).toEqual([]);
    expect(button('Kiểm tra')).toBeUndefined();
    expect(el.querySelector('.tip')).toBeNull();
    expect(results).toEqual([]);
    expect(text(el.querySelector('.progress-line'))).toBe('Đã kiểm tra 1/2 câu');

    await click(button('Câu tiếp theo')!);
    expect(text(el.querySelector('.line'))).toBe('Nice to you!');
    await type(2, 'see');
    await click(checkButton());
    expect(status()).toBe('Đúng 0/1 ô');
    expect(text(el.querySelector('.mark'))).toBe('Sai · Đáp án: meet');
    expect(missed).toEqual([[], ['meet']]);
    expect(results).toEqual([{ correct: 2, total: 3 }]);
    expect(text(el.querySelector('.result.total'))).toBe('Cả bài: đúng 2/3 ô');
    expect(text(el.querySelector('.tip-title'))).toBe('Mẹo ngữ pháp');
    expect(text(el.querySelector('.tip p'))).toBe('Dùng "Nice to meet you".');

    // A checked line keeps its result when shown again.
    await click(button('Câu trước')!);
    expect(status()).toBe('Chính xác!');
    await click(button('Câu sau')!);
    await click(button('Hoàn thành')!);
    expect(done).toBe(1);
  });

  it('a wrong blank says "Sai" with the answer; the line result counts right blanks', async () => {
    await render();
    await type(0, 'names');
    await type(1, 'Minh');
    await click(checkButton());
    expect(status()).toBe('Đúng 1/2 ô');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual([
      'Sai · Đáp án: name',
      'Đúng',
    ]);
    expect(missed).toEqual([['name']]);
  });

  it('checks without case or spaces and hides an empty grammar tip', async () => {
    await render(['Name', 'MINH', 'MEET'], fill, '');
    await type(0, '  NAME');
    await click(bankButton('MINH'));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    await click(button('Câu tiếp theo')!);
    await click(bankButton('MEET'));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(results).toEqual([{ correct: 3, total: 3 }]);
    expect(el.querySelector('.tip')).toBeNull();
  });

  it('fades one bank tile per blank holding the same word', async () => {
    const twice: PracticeFill = {
      turns: [fill.turns[0]],
      blanks: [{ answer: 'meet' }, { answer: 'meet' }],
      wordBank: ['meet', 'meet'],
    };
    await render(['meet', 'meet', 'name'], twice);
    await type(0, 'meet');
    expect(bankButton('meet', 0).disabled).toBe(true);
    expect(bankButton('meet', 1).disabled).toBe(false);
    await click(bankButton('meet', 1));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(button('Hoàn thành')).toBeDefined();
  });

  it('Làm lại câu này clears only the line on screen; the new check counts', async () => {
    let restarts = 0;
    await render();
    fixture.componentInstance.restarted.subscribe(() => restarts++);
    expect(button('Làm lại bước này')).toBeUndefined();
    await type(0, 'name');
    await type(1, 'Tom');
    await click(checkButton());
    expect(status()).toBe('Đúng 1/2 ô');
    expect(button('Làm lại bước này')).toBeDefined();

    await click(button('Làm lại câu này')!);
    expect(blank(0).value).toBe('');
    expect(blank(1).value).toBe('');
    expect(blank(0).readOnly).toBe(false);
    expect(el.querySelector('.mark')).toBeNull();
    expect(checkButton().disabled).toBe(true);
    expect(button('Làm lại câu này')).toBeUndefined();

    await type(0, 'name');
    await type(1, 'Minh');
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    await click(button('Câu tiếp theo')!);
    await type(2, 'meet');
    await click(checkButton());
    expect(results).toEqual([{ correct: 3, total: 3 }]);
    expect(restarts).toBe(0);
  });

  it('Làm lại bước này clears every line and goes back to the first one', async () => {
    let restarts = 0;
    await render();
    fixture.componentInstance.restarted.subscribe(() => restarts++);
    await type(0, 'name');
    await type(1, 'Minh');
    await click(checkButton());
    await click(button('Câu tiếp theo')!);
    await type(2, 'eat');
    await click(checkButton());
    expect(results).toEqual([{ correct: 2, total: 3 }]);

    await click(button('Làm lại bước này')!);
    expect(restarts).toBe(1);
    expect(text(el.querySelector('.step-card-head .muted'))).toBe('Câu 1/2');
    expect(blank(0).value).toBe('');
    expect(blank(1).value).toBe('');
    expect(el.querySelector('.mark')).toBeNull();
    expect(text(el.querySelector('.progress-line'))).toBe('Đã kiểm tra 0/2 câu');
    expect(el.querySelector('.total')).toBeNull();
    expect(button('Làm lại bước này')).toBeUndefined();
    await click(el.querySelectorAll<HTMLButtonElement>('.step-nav .nav-btn')[1]);
    expect(blank(2).value).toBe('');
  });
});
