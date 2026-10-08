import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeTranslation } from '../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { TranslateStep } from './translate-step';

const translation: PracticeTranslation = {
  vi: 'Rất vui được gặp bạn.',
  answer: ['Nice', 'to', 'meet', 'you.'],
  tiles: ['Nice', 'to', 'meet', 'you.', 'see', 'glad'],
};

describe('TranslateStep', () => {
  let fixture: ComponentFixture<TranslateStep>;
  let el: HTMLElement;
  let played: string[];
  let results: boolean[];
  let moves: number[];
  let done: number;

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
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(new FakeSpeech())] });
    fixture = TestBed.createComponent(TranslateStep);
    fixture.componentRef.setInput('translation', t);
    fixture.componentRef.setInput('tiles', tiles);
    fixture.componentRef.setInput('index', 0);
    fixture.componentRef.setInput('count', 4);
    played = [];
    results = [];
    fixture.componentInstance.readAloud.subscribe((t) => played.push(t));
    fixture.componentInstance.checked.subscribe((r) => results.push(r));
    moves = [];
    done = 0;
    fixture.componentInstance.go.subscribe((i) => moves.push(i));
    fixture.componentInstance.done.subscribe(() => done++);
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

  it('right order: "Chính xác!" and only then the answer can be heard', async () => {
    await render();
    const speaker = () => el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu tiếng Anh đúng"]');
    // Hidden before checking: hearing it would give the answer away.
    expect(speaker()).toBeNull();
    await tap('Nice', 'to', 'meet', 'you.');
    expect(speaker()).toBeNull();
    await click(named('Kiểm tra'));
    expect(status()).toBe('Chính xác!');
    expect(results).toEqual([true]);
    await click(speaker()!);
    expect(played).toEqual(['Nice to meet you.']);
    // Locked after checking; the main button now moves on.
    expect(bankButtons().every((b) => b.disabled)).toBe(true);
    expect(named('Kiểm tra')).toBeUndefined();
    expect(named('Câu tiếp theo')).toBeDefined();
  });

  it('moves between the sentences with ← Câu tiếp theo →, and Hoàn thành after the last one', async () => {
    await render();
    expect(named('Câu trước').disabled).toBe(true);
    await click(named('Câu sau'));
    expect(moves).toEqual([1]);

    await tap('Nice');
    await click(named('Kiểm tra'));
    await click(named('Câu tiếp theo'));
    expect(moves).toEqual([1, 1]);

    fixture.componentRef.setInput('index', 3);
    await fixture.whenStable();
    expect(named('Câu sau').disabled).toBe(true);
    await click(named('Câu trước'));
    expect(moves).toEqual([1, 1, 2]);
    expect(done).toBe(0);
    fixture.componentRef.setInput('translation', { ...translation, vi: 'Câu cuối.' });
    await fixture.whenStable();
    await tap('Nice');
    await click(named('Kiểm tra'));
    expect(named('Câu tiếp theo')).toBeUndefined();
    await click(named('Hoàn thành'));
    expect(done).toBe(1);
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
    // A wrong answer does not unlock the sound.
    expect(el.querySelector('button[aria-label="Nghe câu tiếng Anh đúng"]')).toBeNull();
  });

  it('tiles with the same text replace each other', async () => {
    const t: PracticeTranslation = {
      vi: 'Con mèo và con chó.',
      answer: ['the', 'cat', 'and', 'the', 'dog'],
      tiles: ['the', 'cat', 'and', 'the', 'dog'],
    };
    await render(t, ['the', 'dog', 'the', 'and', 'cat']);
    await click(bankButton('the', 1));
    await tap('cat', 'and', 'the', 'dog');
    await click(named('Kiểm tra'));
    expect(status()).toBe('Chính xác!');
    // Read by the browser, so every sentence built correctly has a speaker button.
    expect(el.querySelector('button[aria-label="Nghe câu tiếng Anh đúng"]')).not.toBeNull();
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

  it('Làm lại câu này, after a check, clears the tiles and the result', async () => {
    let redone = 0;
    await render();
    fixture.componentInstance.redone.subscribe(() => redone++);
    await tap('see', 'to', 'meet', 'you.');
    await click(named('Kiểm tra'));
    expect(status()).toContain('Chưa đúng');
    const redo = named('Làm lại câu này');
    // The button is not part of the result read out by screen readers.
    expect(el.querySelector('[role="status"]')!.contains(redo)).toBe(false);

    await click(redo);
    expect(redone).toBe(1);
    expect(chosen()).toEqual([]);
    expect(status()).toBe('');
    expect(bankButtons().every((b) => !b.disabled)).toBe(true);
    await tap('Nice', 'to', 'meet', 'you.');
    await click(named('Kiểm tra'));
    expect(results).toEqual([false, true]);
  });

  it('Làm lại bước này clears the sentence and asks the page to start again', async () => {
    let restarts = 0;
    await render();
    fixture.componentInstance.restart.subscribe(() => restarts++);
    expect(Array.from(el.querySelectorAll('button')).some((b) => text(b) === 'Làm lại bước này')).toBe(false);
    await tap('Nice', 'to', 'meet', 'you.');
    await click(named('Kiểm tra'));

    await click(named('Làm lại bước này'));
    expect(restarts).toBe(1);
    expect(chosen()).toEqual([]);
    expect(status()).toBe('');
    expect(named('Kiểm tra').disabled).toBe(true);
  });
});
