import { TestBed } from '@angular/core/testing';

import { JobStatus } from '../../../core/models/lesson';
import { StatusChip } from './status-chip';

describe('StatusChip', () => {
  const render = async (status: JobStatus) => {
    const fixture = TestBed.createComponent(StatusChip);
    fixture.componentRef.setInput('status', status);
    fixture.componentRef.setInput('label', 'Chú thích');
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  };

  it.each([
    ['running', 'Chú thích: Đang chạy'],
    ['done', 'Chú thích: Xong'],
    ['failed', 'Chú thích: Lỗi'],
  ] as const)('shows %s as text with a hidden icon', async (status, text) => {
    const el = await render(status);
    expect(el.querySelector('.text')?.textContent?.trim()).toBe(text);
    expect(el.querySelector('.icon')?.getAttribute('aria-hidden')).toBe('true');
    expect(el.querySelector('.chip')?.getAttribute('data-status')).toBe(status);
  });
});
