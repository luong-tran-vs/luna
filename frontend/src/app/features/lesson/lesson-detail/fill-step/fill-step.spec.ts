import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeFill } from '../../../../core/models/practice';
import { Score } from '../practice-logic';
import { FillStep } from './fill-step';

const fill: PracticeFill = {
  turns: [
    {
      speaker: 0,
      turnIndex: 0,
      meaningVi: 'Xin chào, tên mình là Minh.',
      parts: [{ text: 'Hello, my ' }, { blank: 0 }, { text: ' is Minh.' }],
    },
    {
      speaker: 1,
      turnIndex: 2,
      meaningVi: 'Rất vui được gặp bạn.',
      parts: [{ text: 'Nice to ' }, { blank: 1 }, { text: ' you!' }],
    },
  ],
  blanks: [{ answer: 'name' }, { answer: 'meet' }],
  wordBank: ['name', 'meet', 'from'],
};

describe('FillStep', () => {
  let fixture: ComponentFixture<FillStep>;
  let el: HTMLElement;
  let played: string[];
  let results: Score[];

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const blank = (n: number) => el.querySelectorAll<HTMLButtonElement>('.blank')[n];
  const bankButtons = () =>
    Array.from(
      el.querySelectorAll<HTMLButtonElement>('ul[aria-labelledby="fill-bank-heading"] .tile'),
    );
  const bankButton = (word: string, nth = 0) => bankButtons().filter((b) => text(b) === word)[nth];
  const checkButton = () =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find(
      (b) => text(b) === 'Kiểm tra',
    )!;
  const status = () => text(el.querySelector('[role="status"]'));
  const click = async (b: HTMLButtonElement) => {
    b.click();
    await fixture.whenStable();
  };

  const render = async (
    bank = ['from', 'meet', 'name'],
    f: PracticeFill = fill,
    tip = 'Dùng "Nice to meet you".',
  ) => {
    fixture = TestBed.createComponent(FillStep);
    fixture.componentRef.setInput('fill', f);
    fixture.componentRef.setInput('bank', bank);
    fixture.componentRef.setInput('speakers', ['Minh', 'Anna']);
    fixture.componentRef.setInput('turnAudio', ['/a/turn/0', null, '/a/turn/2']);
    fixture.componentRef.setInput('grammarTip', tip);
    played = [];
    results = [];
    fixture.componentInstance.playAudio.subscribe((url) => played.push(url));
    fixture.componentInstance.checked.subscribe((r) => results.push(r));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows each turn with its speaker, blanks and meaning; the first blank is selected', async () => {
    await render();
    const turns = el.querySelectorAll('.turn');
    expect(text(turns[0].querySelector('.speaker'))).toBe('Minh');
    expect(text(turns[0].querySelector('.meaning'))).toBe('Xin chào, tên mình là Minh.');
    expect(text(turns[1].querySelector('.line'))).toBe('Nice to chạm để điền you!');
    expect(blank(0).getAttribute('aria-pressed')).toBe('true');
    expect(blank(1).getAttribute('aria-pressed')).toBe('false');
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1: trống');
    expect(bankButtons().map((b) => text(b))).toEqual(['from', 'meet', 'name']);
  });

  it('plays the turn of a line through "Nghe câu"', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!);
    expect(played).toEqual(['/a/turn/2']);
  });

  it('fills the selected blank, fades the word and selects the next empty blank', async () => {
    await render();
    await click(bankButton('name'));
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1: name');
    expect(text(blank(0))).toBe('name');
    expect(bankButton('name').disabled).toBe(true);
    expect(blank(1).getAttribute('aria-pressed')).toBe('true');
    expect(blank(0).getAttribute('aria-pressed')).toBe('false');
  });

  it('✕ removes the word, gives it back to the bank and selects that blank', async () => {
    await render();
    await click(bankButton('name'));
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Gỡ từ name"]')!);
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1: trống');
    expect(bankButton('name').disabled).toBe(false);
    expect(blank(0).getAttribute('aria-pressed')).toBe('true');
  });

  it('a selected filled blank is replaced by the next word; the old word returns to the bank', async () => {
    await render();
    await click(bankButton('from'));
    await click(blank(0));
    expect(blank(0).getAttribute('aria-pressed')).toBe('true');
    await click(bankButton('name'));
    expect(text(blank(0))).toBe('name');
    expect(bankButton('from').disabled).toBe(false);
    expect(bankButton('name').disabled).toBe(true);
  });

  it('keeps Kiểm tra disabled until every blank is filled', async () => {
    await render();
    expect(checkButton().disabled).toBe(true);
    await click(bankButton('name'));
    expect(checkButton().disabled).toBe(true);
    await click(bankButton('meet'));
    expect(checkButton().disabled).toBe(false);
  });

  it('all correct: "Chính xác!", each blank marked Đúng, then the grammar tip', async () => {
    await render();
    expect(el.querySelector('.tip')).toBeNull();
    await click(bankButton('name'));
    await click(bankButton('meet'));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual(['Đúng', 'Đúng']);
    expect(text(el.querySelector('.tip-title'))).toBe('Mẹo ngữ pháp');
    expect(text(el.querySelector('.tip p'))).toBe('Dùng "Nice to meet you".');
    expect(results).toEqual([{ correct: 2, total: 2 }]);
    // Locked after checking.
    expect(blank(0).disabled).toBe(true);
    expect(bankButton('from').disabled).toBe(true);
    expect(el.querySelector('button[aria-label^="Gỡ từ"]')).toBeNull();
    expect(checkButton().disabled).toBe(true);
  });

  it('a wrong blank says "Sai" with the answer; the result counts right blanks', async () => {
    await render();
    await click(bankButton('from'));
    await click(bankButton('meet'));
    await click(checkButton());
    expect(status()).toBe('Đúng 1/2 ô');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual([
      'Sai · Đáp án: name',
      'Đúng',
    ]);
    expect(results).toEqual([{ correct: 1, total: 2 }]);
  });

  it('checks without case differences and hides an empty grammar tip', async () => {
    await render(['Name', 'MEET'], fill, '');
    await click(bankButton('Name'));
    await click(bankButton('MEET'));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(el.querySelector('.tip')).toBeNull();
  });

  it('treats bank tiles with the same word as interchangeable', async () => {
    const twice: PracticeFill = {
      ...fill,
      blanks: [{ answer: 'meet' }, { answer: 'meet' }],
      wordBank: ['meet', 'meet'],
    };
    await render(['meet', 'meet', 'name'], twice);
    await click(bankButton('meet', 1));
    expect(bankButton('meet', 1).disabled).toBe(true);
    expect(bankButton('meet', 0).disabled).toBe(false);
    await click(bankButton('meet', 0));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
  });

  it('works with the keyboard: blanks, bank words and ✕ are buttons', async () => {
    await render();
    const controls = [blank(0), blank(1), ...bankButtons()];
    expect(controls.every((b) => b.tagName === 'BUTTON' && b.type === 'button')).toBe(true);
    await click(bankButton('meet'));
    const remove = el.querySelector<HTMLButtonElement>('button[aria-label="Gỡ từ meet"]')!;
    expect(remove.tagName).toBe('BUTTON');
    expect(remove.tabIndex).toBe(0);
  });
});
