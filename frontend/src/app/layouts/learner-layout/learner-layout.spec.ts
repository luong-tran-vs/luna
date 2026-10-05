import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { ThemePreference, ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { LearnerLayout } from './learner-layout';

describe('LearnerLayout', () => {
  let fixture: ComponentFixture<LearnerLayout>;
  let el: HTMLElement;
  const preference = signal<ThemePreference>('light');
  const resolved = computed(() => (preference() === 'dark' ? 'dark' : 'light'));
  const setTheme = vi.fn((p: ThemePreference) => preference.set(p));

  const toggle = () => el.querySelector<HTMLButtonElement>('.theme-toggle')!;

  beforeEach(async () => {
    preference.set('light');
    setTheme.mockClear();
    await TestBed.configureTestingModule({
      imports: [LearnerLayout],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ThemeService, useValue: { preference, resolved, set: setTheme } },
        { provide: WritingNotifier, useValue: { unseen: signal(0), toast: signal(null), latest: signal(null) } },
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(LearnerLayout);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('puts Tài khoản last, marked to sit at the end of the bar', () => {
    const tabs = Array.from(el.querySelectorAll('.tab'));
    expect(tabs.map((t) => t.textContent?.trim())).toEqual(['Trang chủ', 'Khóa học', 'Ngữ pháp', 'Ôn tập', 'Tài khoản']);
    expect(tabs.filter((t) => t.classList.contains('tab-end')).map((t) => t.getAttribute('href'))).toEqual([
      '/account',
    ]);
  });

  it('switches dark mode from the header and shows the state without color', async () => {
    const button = toggle();
    expect(button.getAttribute('aria-label')).toBe('Chế độ tối');
    expect(button.getAttribute('aria-pressed')).toBe('false');

    button.click();
    await fixture.whenStable();
    expect(setTheme).toHaveBeenCalledWith('dark');
    expect(toggle().getAttribute('aria-pressed')).toBe('true');
    expect(toggle().title).toBe('Chuyển sang giao diện sáng');

    toggle().click();
    await fixture.whenStable();
    expect(setTheme).toHaveBeenLastCalledWith('light');
    expect(toggle().getAttribute('aria-pressed')).toBe('false');
  });
});
