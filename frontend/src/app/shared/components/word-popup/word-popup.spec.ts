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
  const events = { closed: vi.fn(), save: vi.fn(), play: vi.fn() };

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
});
