import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AppUpdateService } from '../../../core/services/app-update.service';
import { UpdateBanner } from './update-banner';

describe('UpdateBanner', () => {
  let fixture: ComponentFixture<UpdateBanner>;
  let el: HTMLElement;
  const ready = signal(false);
  const downloading = signal(false);
  const update = vi.fn();

  const dialog = () => el.querySelector('dialog') as HTMLDialogElement;

  beforeEach(async () => {
    // jsdom has no <dialog> methods.
    HTMLDialogElement.prototype.showModal ??= function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    };
    ready.set(false);
    downloading.set(false);
    update.mockClear();
    TestBed.configureTestingModule({
      providers: [{ provide: AppUpdateService, useValue: { ready, downloading, update } }],
    });
    fixture = TestBed.createComponent(UpdateBanner);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('says a new version is downloading, without blocking the page', async () => {
    const status = el.querySelector('[role="status"]') as HTMLElement;
    expect(status.textContent?.trim()).toBe('');
    downloading.set(true);
    await fixture.whenStable();
    expect(status.textContent).toContain('Đang tải bản mới…');
    expect(dialog().open).toBe(false);
    downloading.set(false);
    await fixture.whenStable();
    expect(status.textContent?.trim()).toBe('');
  });

  it('opens the dialog only once a new version is ready', async () => {
    expect(dialog().open).toBe(false);
    ready.set(true);
    await fixture.whenStable();
    expect(dialog().open).toBe(true);
    expect(dialog().textContent).toContain('Đã có phiên bản mới');
  });

  it('offers only Cập nhật, and Escape does not close it', async () => {
    ready.set(true);
    await fixture.whenStable();
    const buttons = Array.from(dialog().querySelectorAll('button'));
    expect(buttons.map((b) => b.textContent?.trim())).toEqual(['Cập nhật']);
    buttons[0].click();
    expect(update).toHaveBeenCalledOnce();

    const cancel = new Event('cancel', { cancelable: true });
    dialog().dispatchEvent(cancel);
    expect(cancel.defaultPrevented).toBe(true);
    expect(dialog().open).toBe(true);
  });
});
