import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { WritingNotifier, WritingToast } from '../../../core/services/writing-notifier.service';
import { Toast, TOAST_DURATION } from './toast';

describe('Toast', () => {
  let fixture: ComponentFixture<Toast>;
  let el: HTMLElement;
  const toast = signal<WritingToast | null>(null);
  const notifier = { toast, dismissToast: () => toast.set(null) };

  beforeEach(async () => {
    vi.useFakeTimers();
    toast.set(null);
    await TestBed.configureTestingModule({
      imports: [Toast],
      providers: [provideRouter([]), { provide: WritingNotifier, useValue: notifier }],
    }).compileComponents();
    fixture = TestBed.createComponent(Toast);
    el = fixture.nativeElement;
    fixture.detectChanges();
  });

  afterEach(() => vi.useRealTimers());

  const show = () => {
    toast.set({ writingId: 'w1', status: 'done', message: 'Bài viết đã có kết quả' });
    fixture.detectChanges();
  };

  it('shows the notice in a live region with a link to the writing', () => {
    expect(el.querySelector('[role="status"]')?.getAttribute('aria-live')).toBe('polite');
    expect(el.querySelector('.toast')).toBeNull();
    show();
    expect(el.querySelector('.message')?.textContent?.trim()).toBe('Bài viết đã có kết quả');
    expect(el.querySelector('a')?.getAttribute('href')).toBe('/writings/w1');
  });

  it('closes on the close button', () => {
    show();
    el.querySelector<HTMLButtonElement>('button[aria-label="Đóng thông báo"]')!.click();
    fixture.detectChanges();
    expect(el.querySelector('.toast')).toBeNull();
  });

  it('hides by itself after a while', () => {
    show();
    vi.advanceTimersByTime(TOAST_DURATION - 1);
    expect(toast()).not.toBeNull();
    vi.advanceTimersByTime(1);
    fixture.detectChanges();
    expect(el.querySelector('.toast')).toBeNull();
  });
});
