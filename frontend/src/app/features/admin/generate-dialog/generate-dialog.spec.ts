import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GenerateInput } from '../../../core/models/generate';
import { GenerateDialog } from './generate-dialog';

describe('GenerateDialog', () => {
  let fixture: ComponentFixture<GenerateDialog>;
  let el: HTMLElement;
  let emitted: GenerateInput[];
  let closed: number;

  const options: GenerateInput = { count: 3, words: 120, kind: 'reading', idea: '' };

  beforeEach(async () => {
    // jsdom has no showModal/close.
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    });
    HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
      this.removeAttribute('open');
    });

    await TestBed.configureTestingModule({ imports: [GenerateDialog] }).compileComponents();
    fixture = TestBed.createComponent(GenerateDialog);
    fixture.componentRef.setInput('topicLabel', 'A1 · Gia đình');
    fixture.componentRef.setInput('options', options);
    fixture.componentRef.setInput('open', true);
    emitted = [];
    closed = 0;
    fixture.componentInstance.generate.subscribe((v) => emitted.push(v));
    fixture.componentInstance.closed.subscribe(() => closed++);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  const input = (id: string) => el.querySelector<HTMLInputElement | HTMLTextAreaElement>(`#${id}`)!;
  const type = async (id: string, value: string) => {
    const field = input(id);
    field.value = value;
    field.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const submit = async () => {
    el.querySelector<HTMLButtonElement>('button[type="submit"]')!.click();
    await fixture.whenStable();
  };
  const errorOf = (field: string) => el.querySelector(`#generate-${field}-error`)?.textContent?.trim();

  it('opens as a modal with the topic and the given options', () => {
    const dialog = el.querySelector('dialog')!;
    expect(dialog.showModal).toHaveBeenCalled();
    expect(dialog.hasAttribute('open')).toBe(true);
    expect(el.querySelector('#generate-title')?.textContent).toContain('Sinh bài bằng AI');
    expect(el.textContent).toContain('A1 · Gia đình');
    expect(input('generate-count').value).toBe('3');
    expect(input('generate-words').value).toBe('120');
    expect(el.querySelector<HTMLInputElement>('input[value="reading"]')!.checked).toBe(true);
    expect(input('generate-idea').value).toBe('');
    expect(el.textContent).toContain('96–144 từ');
  });

  it('emits trimmed valid values', async () => {
    await type('generate-count', '2');
    await type('generate-words', '200');
    el.querySelector<HTMLInputElement>('input[value="dialogue"]')!.click();
    await type('generate-idea', '  một bữa tiệc  ');
    await submit();
    expect(emitted).toEqual([{ count: 2, words: 200, kind: 'dialogue', idea: 'một bữa tiệc' }]);
  });

  for (const [field, value, message] of [
    ['count', '0', 'Số bài từ 1 đến 5'],
    ['count', '6', 'Số bài từ 1 đến 5'],
    ['count', '2.5', 'Số bài từ 1 đến 5'],
    ['words', '49', 'Độ dài từ 50 đến 800 từ'],
    ['words', '801', 'Độ dài từ 50 đến 800 từ'],
    ['idea', 'x'.repeat(501), 'Ý chính tối đa 500 ký tự'],
  ]) {
    it(`rejects ${field} = ${value.slice(0, 10)}`, async () => {
      await type(`generate-${field}`, value);
      await submit();
      expect(emitted).toEqual([]);
      expect(errorOf(field)).toBe(message);
      expect(input(`generate-${field}`).getAttribute('aria-invalid')).toBe('true');
      expect(input(`generate-${field}`).getAttribute('aria-describedby')).toContain(`generate-${field}-error`);
    });
  }

  it('locks everything while generating and ignores Escape', async () => {
    fixture.componentRef.setInput('busy', true);
    await fixture.whenStable();
    const button = el.querySelector<HTMLButtonElement>('button[type="submit"]')!;
    expect(button.textContent?.trim()).toBe('Đang sinh…');
    expect(button.disabled).toBe(true);
    expect(button.getAttribute('aria-busy')).toBe('true');
    expect(input('generate-count').disabled).toBe(true);
    expect(input('generate-idea').disabled).toBe(true);

    const cancel = new Event('cancel', { cancelable: true });
    el.querySelector('dialog')!.dispatchEvent(cancel);
    expect(cancel.defaultPrevented).toBe(true);
    expect(closed).toBe(0);
  });

  it('shows the error and keeps the values', async () => {
    await type('generate-idea', 'picnic');
    fixture.componentRef.setInput('busy', true);
    await fixture.whenStable();
    fixture.componentRef.setInput('busy', false);
    fixture.componentRef.setInput('error', 'Đã hết lượt AI, vui lòng thử lại sau.');
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')?.textContent?.trim()).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
    expect(input('generate-idea').value).toBe('picnic');
    expect(input('generate-idea').disabled).toBe(false);
  });

  it('closes on Huỷ and on Escape', async () => {
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === 'Huỷ')!.click();
    const cancel = new Event('cancel', { cancelable: true });
    el.querySelector('dialog')!.dispatchEvent(cancel);
    expect(closed).toBe(2);

    fixture.componentRef.setInput('open', false);
    await fixture.whenStable();
    expect(el.querySelector('dialog')!.hasAttribute('open')).toBe(false);
  });

  it('resets to the options when opened again', async () => {
    await type('generate-count', '5');
    fixture.componentRef.setInput('open', false);
    await fixture.whenStable();
    fixture.componentRef.setInput('options', { count: 1, words: 300, kind: 'dialogue', idea: 'x' });
    fixture.componentRef.setInput('open', true);
    await fixture.whenStable();
    expect(input('generate-count').value).toBe('1');
    expect(input('generate-words').value).toBe('300');
    expect(el.querySelector<HTMLInputElement>('input[value="dialogue"]')!.checked).toBe(true);
  });
});
