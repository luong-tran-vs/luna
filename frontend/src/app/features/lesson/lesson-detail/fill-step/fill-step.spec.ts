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
  const blank = (n: number) => el.querySelectorAll<HTMLInputElement>('input.blank')[n];
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
  const type = async (n: number, value: string) => {
    const input = blank(n);
    input.dispatchEvent(new Event('focus'));
    input.value = value;
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };

  const render = async (
    bank = ['from', 'meet', 'name'],
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
    fixture.componentInstance.readAloud.subscribe((t) => played.push(t));
    fixture.componentInstance.checked.subscribe((r) => results.push(r));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows each turn with its speaker, a text input per blank and the meaning', async () => {
    await render();
    const turns = el.querySelectorAll('.turn');
    expect(text(turns[0].querySelector('.speaker'))).toBe('Minh');
    expect(text(turns[0].querySelector('.meaning'))).toBe('Xin chào, tên mình là Minh.');
    expect(text(turns[1].querySelector('.line'))).toBe('Nice to you!');
    expect(blank(0).type).toBe('text');
    expect(blank(0).getAttribute('aria-label')).toBe('Ô trống 1');
    expect(blank(0).classList).toContain('selected');
    // Same width for every blank: it does not give the answer away.
    expect(blank(0).getAttribute('size')).toBe(blank(1).getAttribute('size'));
    expect(bankButtons().map((b) => text(b))).toEqual(['from', 'meet', 'name']);
  });

  it('reads the whole turn of a line through "Nghe câu"', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!);
    expect(played).toEqual(['Nice to meet you!']);
  });

  it('typing fills a blank and fades its word in the bank', async () => {
    await render();
    await type(0, 'Name ');
    expect(blank(0).classList).toContain('filled');
    expect(bankButton('name').disabled).toBe(true);
    await type(0, '');
    expect(bankButton('name').disabled).toBe(false);
  });

  it('a word of the bank goes into the selected blank, then the next empty one is selected', async () => {
    await render();
    await click(bankButton('name'));
    expect(blank(0).value).toBe('name');
    expect(bankButton('name').disabled).toBe(true);
    expect(blank(1).classList).toContain('selected');
    // The focused blank is filled: the next word goes to the next empty one.
    await type(0, 'name');
    await click(bankButton('meet'));
    expect(blank(1).value).toBe('meet');
    // Every blank filled: the word replaces the focused one.
    await type(0, 'name');
    await click(bankButton('from'));
    expect(blank(0).value).toBe('from');
    expect(bankButton('name').disabled).toBe(false);
  });

  it('Enter goes to the next blank, and checks on the last one', async () => {
    await render();
    await type(0, 'name');
    blank(0).dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    await fixture.whenStable();
    expect(blank(1).classList).toContain('selected');
    await type(1, 'meet');
    blank(1).dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    await fixture.whenStable();
    expect(status()).toBe('Chính xác!');
  });

  it('keeps Kiểm tra disabled until every blank is filled', async () => {
    await render();
    expect(checkButton().disabled).toBe(true);
    await type(0, 'name');
    expect(checkButton().disabled).toBe(true);
    await type(1, '   ');
    expect(checkButton().disabled).toBe(true);
    await type(1, 'meet');
    expect(checkButton().disabled).toBe(false);
  });

  it('all correct: "Chính xác!", each blank marked Đúng, then the grammar tip', async () => {
    await render();
    expect(el.querySelector('.tip')).toBeNull();
    await type(0, 'name');
    await type(1, 'meet');
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual(['Đúng', 'Đúng']);
    expect(text(el.querySelector('.tip-title'))).toBe('Mẹo ngữ pháp');
    expect(text(el.querySelector('.tip p'))).toBe('Dùng "Nice to meet you".');
    expect(results).toEqual([{ correct: 2, total: 2 }]);
    // Locked after checking.
    expect(blank(0).readOnly).toBe(true);
    expect(bankButton('from').disabled).toBe(true);
    expect(checkButton().disabled).toBe(true);
  });

  it('a wrong blank says "Sai" with the answer; the result counts right blanks', async () => {
    await render();
    await type(0, 'names');
    await type(1, 'meet');
    await click(checkButton());
    expect(status()).toBe('Đúng 1/2 ô');
    expect(Array.from(el.querySelectorAll('.mark')).map((m) => text(m))).toEqual([
      'Sai · Đáp án: name',
      'Đúng',
    ]);
    expect(results).toEqual([{ correct: 1, total: 2 }]);
  });

  it('checks without case or spaces and hides an empty grammar tip', async () => {
    await render(['Name', 'MEET'], fill, '');
    await type(0, '  NAME');
    await click(bankButton('MEET'));
    await click(checkButton());
    expect(status()).toBe('Chính xác!');
    expect(el.querySelector('.tip')).toBeNull();
  });

  it('fades one bank tile per blank holding the same word', async () => {
    const twice: PracticeFill = {
      ...fill,
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
  });
});
