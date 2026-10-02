import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeTranslation } from '../../../../core/models/practice';
import { TranslateStep } from './translate-step';

const translation: PracticeTranslation = {
  vi: 'Rất vui được gặp bạn.',
  answer: ['Nice', 'to', 'meet', 'you.'],
  tiles: ['Nice', 'to', 'meet', 'you.', 'see', 'glad'],
  audioUrl: '/a/answer/0',
};

describe('TranslateStep', () => {
  let fixture: ComponentFixture<TranslateStep>;
  let el: HTMLElement;
  let played: string[];
  let results: boolean[];

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const bankButtons = () =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('ul[aria-labelledby="tiles-heading"] .tile'));
  const bankButton = (word: string, nth = 0) => bankButtons().filter((b) => text(b) === word)[nth];
  const chosen = () =>
    Array.from(el.querySelectorAll('ul[aria-labelledby="built-heading"] .tile')).map((b) =>
      text(b),
    );
  const named = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;
  const status = () => text(el.querySelector('[role="status"]'));
  const click = async (b: HTMLButtonElement) => {
    b.click();
    await fixture.whenStable();
  };
  const tap = async (...words: string[]) => {
    for (const w of words) {
      await click(bankButtons().find((b) => text(b) === w && !b.disabled)!);
    }
  };

  const render = async (
    t: PracticeTranslation = translation,
    tiles = ['see', 'meet', 'Nice', 'you.', 'glad', 'to'],
  ) => {
    fixture = TestBed.createComponent(TranslateStep);
    fixture.componentRef.setInput('translation', t);
    fixture.componentRef.setInput('tiles', tiles);
    fixture.componentRef.setInput('index', 0);
    fixture.componentRef.setInput('count', 4);
    played = [];
    results = [];
    fixture.componentInstance.playAudio.subscribe((url) => played.push(url));
    fixture.componentInstance.checked.subscribe((r) => results.push(r));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows the sentence, its position, an empty answer and the bank', async () => {
    await render();
    expect(text(el.querySelector('.source-head'))).toBe('Câu 1/4 · Tiếng Việt');
    expect(text(el.querySelector('.vi'))).toBe('"Rất vui được gặp bạn."');
    expect(chosen()).toEqual([]);
    expect(bankButtons().map((b) => text(b))).toEqual([
      'see',
      'meet',
      'Nice',
      'you.',
      'glad',
      'to',
    ]);
    expect(named('Kiểm tra').disabled).toBe(true);
  });

  it('adds tapped tiles in order and fades them in the bank', async () => {
    await render();
    await tap('Nice', 'to');
    expect(chosen()).toEqual(['Nice', 'to']);
    expect(bankButton('Nice').disabled).toBe(true);
    expect(bankButton('see').disabled).toBe(false);
    expect(named('Kiểm tra').disabled).toBe(false);
  });

  it('✕ removes one tile and Làm lại removes all', async () => {
    await render();
    await tap('Nice', 'see', 'to');
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Gỡ từ see"]')!);
    expect(chosen()).toEqual(['Nice', 'to']);
    expect(bankButton('see').disabled).toBe(false);
    await click(named('Làm lại'));
    expect(chosen()).toEqual([]);
    expect(bankButtons().every((b) => !b.disabled)).toBe(true);
    expect(named('Kiểm tra').disabled).toBe(true);
  });

  it('right order: "Chính xác!" and the answer can be heard', async () => {
    await render();
    const speaker = el.querySelector<HTMLButtonElement>(
      'button[aria-label="Nghe câu tiếng Anh đúng"]',
    )!;
    expect(speaker.disabled).toBe(true);
    await tap('Nice', 'to', 'meet', 'you.');
    await click(named('Kiểm tra'));
    expect(status()).toBe('Chính xác!');
    expect(results).toEqual([true]);
    expect(speaker.disabled).toBe(false);
    await click(speaker);
    expect(played).toEqual(['/a/answer/0']);
    // Locked after checking.
    expect(bankButtons().every((b) => b.disabled)).toBe(true);
    expect(named('Kiểm tra').disabled).toBe(true);
  });

  it('wrong order: "Chưa đúng" with the right sentence', async () => {
    await render();
    await tap('to', 'Nice', 'meet', 'you.');
    await click(named('Kiểm tra'));
    expect(Array.from(el.querySelectorAll('[role="status"] p')).map((p) => text(p))).toEqual([
      'Chưa đúng',
      'Câu đúng: Nice to meet you.',
    ]);
    expect(results).toEqual([false]);
  });

  it('tiles with the same text replace each other', async () => {
    const t: PracticeTranslation = {
      vi: 'Con mèo và con chó.',
      answer: ['the', 'cat', 'and', 'the', 'dog'],
      tiles: ['the', 'cat', 'and', 'the', 'dog'],
      audioUrl: null,
    };
    await render(t, ['the', 'dog', 'the', 'and', 'cat']);
    await click(bankButton('the', 1));
    await tap('cat', 'and', 'the', 'dog');
    await click(named('Kiểm tra'));
    expect(status()).toBe('Chính xác!');
    // No audio: no speaker button.
    expect(el.querySelector('button[aria-label="Nghe câu tiếng Anh đúng"]')).toBeNull();
  });

  it('starts empty again for the next sentence', async () => {
    await render();
    await tap('Nice');
    await click(named('Kiểm tra'));
    fixture.componentRef.setInput('translation', { ...translation, vi: 'Câu khác.' });
    fixture.componentRef.setInput('index', 1);
    await fixture.whenStable();
    expect(chosen()).toEqual([]);
    expect(status()).toBe('');
    expect(text(el.querySelector('.source-head'))).toBe('Câu 2/4 · Tiếng Việt');
  });

  it('works with the keyboard: every control is a button', async () => {
    await render();
    await tap('Nice');
    const controls = [
      ...bankButtons(),
      ...el.querySelectorAll<HTMLButtonElement>('ul[aria-labelledby="built-heading"] button'),
    ];
    expect(
      controls.every((b) => b.tagName === 'BUTTON' && b.type === 'button' && b.tabIndex === 0),
    ).toBe(true);
  });
});
