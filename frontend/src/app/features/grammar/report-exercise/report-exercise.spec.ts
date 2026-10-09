import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ReportExercise } from './report-exercise';

describe('ReportExercise', () => {
  let fixture: ComponentFixture<ReportExercise>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const trigger = () => el.querySelector<HTMLButtonElement>('.report-btn')!;
  const dialog = () => el.querySelector('dialog')!;
  const submitBtn = () => el.querySelector<HTMLButtonElement>('button[type="submit"]')!;
  const pick = async (label: string) => {
    const input = Array.from(el.querySelectorAll<HTMLLabelElement>('label.reason'))
      .find((l) => text(l) === label)!
      .querySelector('input')!;
    input.click();
    await fixture.whenStable();
  };
  const type = async (value: string) => {
    const area = el.querySelector<HTMLTextAreaElement>('textarea')!;
    area.value = value;
    area.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };

  beforeEach(async () => {
    // jsdom has no showModal/close.
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    });
    HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
      this.removeAttribute('open');
      this.dispatchEvent(new Event('close'));
    });
    TestBed.configureTestingModule({
      imports: [ReportExercise],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ReportExercise);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    fixture.componentRef.setInput('exerciseId', 'p3');
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('opens a titled modal with four reasons and a counted note', async () => {
    trigger().click();
    await fixture.whenStable();
    expect(dialog().showModal).toHaveBeenCalled();
    const labelled = dialog().getAttribute('aria-labelledby')!;
    expect(text(el.querySelector(`#${labelled}`))).toBe('Báo lỗi câu này');
    expect(Array.from(el.querySelectorAll('label.reason')).map((l) => text(l))).toEqual([
      'Đáp án sai', 'Câu mơ hồ', 'Lỗi chính tả', 'Khác',
    ]);
    await type('abc');
    expect(text(el.querySelector('.count'))).toBe('3/300');
    expect(el.querySelector('textarea')!.getAttribute('maxlength')).toBe('300');
  });

  it('asks for a reason before sending', async () => {
    trigger().click();
    submitBtn().click();
    await fixture.whenStable();
    expect(text(el.querySelector('.field-error'))).toBe('Hãy chọn một lý do.');
    http.expectNone('/grammar/a1-to-be/reports');
  });

  it('sends the report, thanks the learner and shows the button as reported', async () => {
    trigger().click();
    await pick('Lỗi chính tả');
    await type('  thiếu dấu  ');
    submitBtn().click();
    const req = http.expectOne('/grammar/a1-to-be/reports');
    expect(req.request.body).toEqual({ exerciseId: 'p3', reason: 'typo', note: 'thiếu dấu' });
    req.flush({ ok: true });
    await fixture.whenStable();
    expect(dialog().hasAttribute('open')).toBe(false);
    expect(text(trigger())).toBe('Đã báo lỗi câu này');
    expect(trigger().getAttribute('aria-disabled')).toBe('true');
    expect(text(el.querySelector('[role="status"]'))).toBe('Đã ghi nhận, cảm ơn bạn.');

    trigger().click();
    expect(dialog().hasAttribute('open')).toBe(false);
  });

  it('keeps what was typed and offers a retry when sending fails', async () => {
    trigger().click();
    await pick('Đáp án sai');
    await type('đáp án là is');
    submitBtn().click();
    http.expectOne('/grammar/a1-to-be/reports').flush({ error: 'x' }, { status: 500, statusText: 'x' });
    await fixture.whenStable();
    expect(text(el.querySelector('.form-alert'))).toContain('Không gửi được');
    expect(el.querySelector<HTMLTextAreaElement>('textarea')!.value).toBe('đáp án là is');
    expect(text(submitBtn())).toBe('Thử lại');
    expect(dialog().hasAttribute('open')).toBe(true);

    submitBtn().click();
    http.expectOne('/grammar/a1-to-be/reports').flush({ ok: true });
    await fixture.whenStable();
    expect(text(trigger())).toBe('Đã báo lỗi câu này');
  });

  it('closes with Huỷ and returns focus to the button', async () => {
    trigger().click();
    await fixture.whenStable();
    const cancel = Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === 'Huỷ')!;
    cancel.click();
    expect(dialog().hasAttribute('open')).toBe(false);
  });
});
