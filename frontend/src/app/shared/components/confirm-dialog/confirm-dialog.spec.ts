import { TestBed } from '@angular/core/testing';

import { ConfirmDialog } from './confirm-dialog';

describe('ConfirmDialog', () => {
  const setup = async (open: boolean) => {
    const fixture = TestBed.createComponent(ConfirmDialog);
    fixture.componentRef.setInput('open', open);
    fixture.componentRef.setInput('title', 'Xoá bài?');
    fixture.componentRef.setInput('message', 'Không hoàn tác được.');
    fixture.componentRef.setInput('confirmLabel', 'Xoá');
    await fixture.whenStable();
    const confirmed = vi.fn();
    const cancelled = vi.fn();
    fixture.componentInstance.confirmed.subscribe(confirmed);
    fixture.componentInstance.cancelled.subscribe(cancelled);
    return { el: fixture.nativeElement as HTMLElement, confirmed, cancelled };
  };

  it('renders nothing when closed', async () => {
    const { el } = await setup(false);
    expect(el.querySelector('[role="alertdialog"]')).toBeNull();
  });

  it('shows the title, message and focuses cancel when open', async () => {
    const { el } = await setup(true);
    const dialog = el.querySelector('[role="alertdialog"]');
    expect(dialog?.getAttribute('aria-modal')).toBe('true');
    expect(el.textContent).toContain('Xoá bài?');
    expect(el.textContent).toContain('Không hoàn tác được.');
    expect(document.activeElement?.textContent?.trim()).toBe('Huỷ');
  });

  it('emits confirmed and cancelled', async () => {
    const { el, confirmed, cancelled } = await setup(true);
    const buttons = el.querySelectorAll('button');
    buttons[1].click();
    buttons[0].click();
    expect(confirmed).toHaveBeenCalledTimes(1);
    expect(cancelled).toHaveBeenCalledTimes(1);
  });

  it('cancels on Escape', async () => {
    const { cancelled } = await setup(true);
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(cancelled).toHaveBeenCalled();
  });
});
