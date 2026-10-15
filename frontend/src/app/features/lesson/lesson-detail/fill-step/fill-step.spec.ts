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
  /** A blank as a line (with a word bank). */
  const line = (n: number) => el.querySelector<HTMLButtonElement>(`#fill-blank-${n}`)!;
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

  /** The word bank of most tests: with one, the blanks are lines to fill from it. */
  const BANK = ['from', 'meet', 'Minh', 'name'];
  /** By default no word bank: the blanks are inputs to type in. */
  const render = async (
    bank: string[] = [],
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

  it('shows one line at a time with its speaker, a text input per blank (no word bank) and the meaning', async () => {
    await render();
    expect(el.querySelectorAll('.turn')).toHaveLength(1);
    // The line is asked by the boy in a bubble, with its number and speaker.
    expect(text(el.querySelector('.turn lu-asker .counter'))).toBe('Câu 1/2 · Minh');
    expect(text(el.querySelector('.meaning'))).toBe('Xin chào, tên mình là Minh.');
    expect(el.querySelectorAll('input.blank')).toHaveLength(2);
    expect(blank(0).type).toBe('text');
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1');
    expect(blank(0).classList).toContain('selected');
    // Same width for every blank: it does not give the answer away.
    expect(blank(0).getAttribute('size')).toBe(blank(1).getAttribute('size'));
    expect(bankButtons()).toEqual([]);

    // The bar has ← to the step before only when the page allows it, then the main button.
    expect(button('Bước trước')).toBeUndefined();
    fixture.componentRef.setInput('canBack', true);
    await fixture.whenStable();
    let backs = 0;
    fixture.componentInstance.back.subscribe(() => backs++);
    await click(el.querySelector<HTMLButtonElement>('.step-nav [aria-label="Bước trước"]')!);
    expect(backs).toBe(1);

    await click(button('Bỏ qua câu này')!);
    expect(text(el.querySelector('.counter'))).toBe('Câu 2/2 · Anna');
    expect(text(el.querySelector('.line'))).toBe('Nice to you!');
  });

  it('reads the whole turn of the line on screen through "Nghe câu"', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 1"]')!);
    await click(button('Bỏ qua câu này')!);
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!);
    expect(played).toEqual(['Hello, my name is Minh.', 'Nice to meet you!']);
  });

  it('with a word bank, the blanks are lines: no input to type in', async () => {
    await render(BANK);
    expect(el.querySelectorAll('input.blank')).toHaveLength(0);
    expect(el.querySelectorAll('button.line-blank')).toHaveLength(2);
    expect(line(0).getAttribute('aria-label')).toBe('Ô trống 1, chưa điền');
    expect(line(0).classList).toContain('selected');
    expect(bankButtons().map((b) => text(b))).toEqual(['from', 'meet', 'Minh', 'name']);
    expect(text(el.querySelector('.step-hint'))).toBe('Chạm ô trống rồi chọn từ bên dưới.');
  });

  it('a word of the bank goes into the chosen line, then the next empty one; tapping a line takes it out', async () => {
    await render(BANK);
    await click(bankButton('name'));
    expect(text(line(0))).toBe('name');
    expect(line(0).getAttribute('aria-label')).toBe('Ô trống 1: name');
    expect(bankButton('name').disabled).toBe(true);
    expect(line(1).classList).toContain('selected');
    await click(bankButton('Minh'));
    expect(text(line(1))).toBe('Minh');
    // Every line filled: a word replaces the chosen one, never another line's.
    await click(line(0));
    expect(text(line(0))).toBe('');
    expect(bankButton('name').disabled).toBe(false);
    await click(bankButton('meet'));
    expect(text(line(0))).toBe('meet');
    await click(checkButton());
    expect(status()).toBe('Đúng 1/2 ô');
    // A checked line cannot be changed any more.
    expect(line(0).disabled).toBe(true);
    await click(button('Câu tiếp theo')!);
    expect(text(line(2))).toBe('');
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
    expect(text(el.querySelector('.progress-line'))).toBe('Đã kiểm tra 1/2 câu có ô trống');

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
    await click(bankButton('Name'));
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

  it('shows a word needed twice as one tile marked ×2, used up once both blanks hold it', async () => {
    const twice: PracticeFill = {
      turns: [fill.turns[0]],
      blanks: [{ answer: 'meet' }, { answer: 'meet' }],
      wordBank: ['meet', 'meet'],
    };
    await render(['meet', 'meet', 'name'], twice);
    const tiles = () => Array.from(el.querySelectorAll<HTMLButtonElement>('.tiles .tile'));
    expect(tiles().map((t) => text(t))).toEqual(['meet ×2 lần', 'name']);
    await click(tiles()[0]);
    expect(text(tiles()[0])).toBe('meet');
    expect(tiles()[0].disabled).toBe(false);
    await click(tiles()[0]);
    expect(tiles().length === 0 || tiles()[0].disabled).toBe(true);
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(button('Hoàn thành')).toBeDefined();
  });

  it('names its last button after what comes next', async () => {
    const one: PracticeFill = { turns: [fill.turns[0]], blanks: fill.blanks, wordBank: fill.wordBank };
    await render(undefined, one);
    fixture.componentRef.setInput('finishLabel', 'Sang bước Viết');
    await type(0, 'name');
    await type(1, 'Mai');
    await click(checkButton());
    expect(button('Sang bước Viết')).toBeDefined();
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
    expect(text(el.querySelector('.counter'))).toBe('Câu 1/2 · Minh');
    expect(blank(0).value).toBe('');
    expect(blank(1).value).toBe('');
    expect(el.querySelector('.mark')).toBeNull();
    expect(text(el.querySelector('.progress-line'))).toBe('Đã kiểm tra 0/2 câu có ô trống');
    expect(el.querySelector('.total')).toBeNull();
    expect(button('Làm lại bước này')).toBeUndefined();
    await click(button('Bỏ qua câu này')!);
    expect(blank(2).value).toBe('');
  });

  it('does not count a line without a blank as checked', async () => {
    const withPlainLine: PracticeFill = {
      ...fill,
      turns: [
        fill.turns[0],
        { speaker: 1, turnIndex: 1, meaningVi: 'Chào.', parts: [{ text: 'Hi.' }] },
        fill.turns[1],
      ],
    };
    await render(undefined, withPlainLine);
    expect(text(el.querySelector('.progress-line'))).toBe('Đã kiểm tra 0/2 câu có ô trống');
    expect(button('Làm lại bước này')).toBeUndefined();
  });
});
