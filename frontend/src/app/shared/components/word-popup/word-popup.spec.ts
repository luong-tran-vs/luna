import { ComponentFixture, TestBed } from '@angular/core/testing';

import { LookupResult } from '../../../core/models/reading';
import { PopupState, WordPopup } from './word-popup';

const aiResult: LookupResult = {
  source: 'ai', text: 'went', lemma: 'go', ipa: '/ɡəʊ/', meanings: [{ pos: '', text: 'đã đi' }],
};
const dictResult: LookupResult = {
  source: 'dictionary', text: 'park', lemma: 'park', ipa: '/pɑːk/',
  meanings: [{ pos: 'N', text: 'Công viên.' }, { pos: 'V', text: 'Đỗ xe.' }, { pos: 'X', text: 'Khác.' }],
};

describe('WordPopup', () => {
  let fixture: ComponentFixture<WordPopup>;
  let el: HTMLElement;
  const events = { closed: vi.fn(), save: vi.fn(), play: vi.fn(), ask: vi.fn() };

  const render = async (state: PopupState, extra: Record<string, unknown> = {}) => {
    fixture = TestBed.createComponent(WordPopup);
    fixture.componentRef.setInput('text', state.kind === 'result' ? state.result.text : 'xyz');
    fixture.componentRef.setInput('state', state);
    for (const [k, v] of Object.entries(extra)) {
      fixture.componentRef.setInput(k, v);
    }
    fixture.componentInstance.closed.subscribe(events.closed);
    fixture.componentInstance.save.subscribe(events.save);
    fixture.componentInstance.listen.subscribe(events.play);
    fixture.componentInstance.ask.subscribe(events.ask);
    await fixture.whenStable();
    el = fixture.nativeElement as HTMLElement;
  };
  const button = (text: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === text);

  beforeEach(() => Object.values(events).forEach((f) => f.mockClear()));

  // --- US1 ---

  it('shows an AI meaning with the base form, IPA and source label', async () => {
    await render({ kind: 'result', result: aiResult });
    const dialog = el.querySelector('[role="dialog"]')!;
    expect(dialog.getAttribute('aria-labelledby')).toBe(el.querySelector('.word')!.id);
    expect(el.querySelector('.word')?.textContent?.trim()).toBe('went');
    expect(el.querySelector('.lemma')?.textContent).toContain('go');
    expect(el.querySelector('.ipa')?.textContent?.trim()).toBe('/ɡəʊ/');
    expect(el.querySelector('.source')?.textContent?.trim()).toBe('AI · theo ngữ cảnh');
    expect(el.textContent).toContain('đã đi');
  });

  it('shows up to 3 dictionary meanings with Vietnamese parts of speech', async () => {
    await render({ kind: 'result', result: dictResult });
    expect(el.querySelector('.lemma')).toBeNull(); // same as the word
    expect(el.querySelector('.source')?.textContent?.trim()).toBe('Từ điển');
    const items = Array.from(el.querySelectorAll('.meanings li')).map((li) => li.textContent?.replace(/\s+/g, ' ').trim());
    expect(items).toEqual(['danh từ Công viên.', 'động từ Đỗ xe.', 'X Khác.']);
  });

  it('shows a loading state', async () => {
    await render({ kind: 'loading' });
    expect(el.querySelector('[role="status"]')?.textContent).toContain('Đang tra');
  });

  it('takes focus when it opens', async () => {
    await render({ kind: 'loading' });
    expect(document.activeElement).toBe(el.querySelector('.popup'));
  });

  it('closes with the close button', async () => {
    await render({ kind: 'result', result: aiResult });
    el.querySelector<HTMLButtonElement>('button[aria-label="Đóng"]')!.click();
    expect(events.closed).toHaveBeenCalled();
  });

  // --- US2 ---

  it('emits save, and shows the saved state instead of the button', async () => {
    await render({ kind: 'result', result: aiResult });
    button('Lưu vào sổ từ')!.click();
    expect(events.save).toHaveBeenCalledWith(undefined);

    await render({ kind: 'result', result: aiResult }, { saved: true });
    expect(button('Lưu vào sổ từ')).toBeUndefined();
    expect(el.textContent).toContain('✓ Đã có trong sổ');
  });

  it('asks for a meaning when nothing was found and validates it', async () => {
    await render({ kind: 'not-found' });
    expect(el.textContent).toContain('Chưa có nghĩa');
    const input = el.querySelector<HTMLInputElement>('input[name="meaning"]')!;

    button('Lưu')!.click();
    await fixture.whenStable();
    expect(events.save).not.toHaveBeenCalled();
    expect(el.querySelector('.field-error')?.textContent).toContain('Vui lòng nhập nghĩa');

    input.value = 'x'.repeat(201);
    input.dispatchEvent(new Event('input'));
    button('Lưu')!.click();
    await fixture.whenStable();
    expect(events.save).not.toHaveBeenCalled();

    input.value = '  một cái tên  ';
    input.dispatchEvent(new Event('input'));
    button('Lưu')!.click();
    expect(events.save).toHaveBeenCalledWith('một cái tên');
  });

  it('shows a save error as an alert', async () => {
    await render({ kind: 'result', result: aiResult }, { error: 'Không lưu được, vui lòng thử lại.' });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Không lưu được');
  });

  // --- US4 ---

  it('emits play from the listen button', async () => {
    await render({ kind: 'result', result: aiResult });
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe phát âm"]')!.click();
    expect(events.play).toHaveBeenCalled();
  });

  // --- F9: Hỏi AI ---

  it('offers "Hỏi AI" when nothing was found or the meaning is from the dictionary', async () => {
    await render({ kind: 'not-found' });
    button('Hỏi AI')!.click();
    expect(events.ask).toHaveBeenCalledTimes(1);

    await render({ kind: 'result', result: dictResult });
    expect(button('Hỏi AI')).toBeTruthy();
  });

  it('does not offer "Hỏi AI" for an AI meaning or while looking up', async () => {
    await render({ kind: 'result', result: aiResult });
    expect(button('Hỏi AI')).toBeUndefined();
    await render({ kind: 'loading' });
    expect(button('Hỏi AI')).toBeUndefined();
  });

  it('shows the asking state and the error', async () => {
    await render({ kind: 'not-found' }, { asking: true });
    const asking = button('Đang hỏi AI…')!;
    expect(asking.disabled).toBe(true);
    expect(el.querySelector('[role="status"]')?.textContent).toContain('Đang hỏi AI');

    await render({ kind: 'result', result: dictResult }, { askError: 'Đã hết lượt AI, vui lòng thử lại sau.' });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Đã hết lượt AI');
    // The dictionary meaning stays.
    expect(el.textContent).toContain('Công viên.');
  });

  it('shows the note of an asked meaning', async () => {
    await render({ kind: 'result', result: { ...aiResult, note: 'Quá khứ của go.' } });
    expect(el.querySelector('.note')?.textContent?.trim()).toBe('Quá khứ của go.');
    await render({ kind: 'result', result: dictResult });
    expect(el.querySelector('.note')).toBeNull();
  });
});
