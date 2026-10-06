import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { Goals } from '../../core/models/study';
import { User } from '../../core/models/user';
import { NaturalVoiceService, NaturalVoiceState } from '../../core/natural-voice/natural-voice.service';
import { AuthService } from '../../core/services/auth.service';
import { PALETTE_STORAGE_KEY } from '../../core/services/palette.service';
import { ThemePreference, ThemeSaveState, ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Account } from './account';

describe('Account', () => {
  let fixture: ComponentFixture<Account>;
  let http: HttpTestingController;
  let el: HTMLElement;
  const preference = signal<ThemePreference>('light');
  const saveState = signal<ThemeSaveState>('idle');
  const setTheme = vi.fn((p: ThemePreference) => preference.set(p));
  const resolved = computed(() => (preference() === 'dark' ? 'dark' : 'light'));
  const unseen = signal(0);
  const voiceState = signal<NaturalVoiceState>('off');
  const voice = {
    supported: true,
    state: voiceState,
    enabled: computed(() => voiceState() !== 'off'),
    percent: signal(40),
    enable: vi.fn(() => voiceState.set('loading')),
    disable: vi.fn(() => voiceState.set('off')),
  };
  const logout = vi.fn(async () => undefined);

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const darkSwitch = () => el.querySelector<HTMLButtonElement>('[role="switch"]')!;

  const open = async (goals: Goals | 'error' = { active: null, others: [] }) => {
    fixture = TestBed.createComponent(Account);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const req = http.expectOne('/api/goals');
    if (goals === 'error') {
      req.flush('down', { status: 500, statusText: 'Error' });
    } else {
      req.flush(goals);
    }
    await fixture.whenStable();
  };

  beforeEach(async () => {
    voiceState.set('off');
    preference.set('light');
    saveState.set('idle');
    setTheme.mockClear();
    logout.mockClear();
    unseen.set(0);
    const user: User = { id: '1', email: 'minh@example.com', role: 'learner', timezone: 'Asia/Ho_Chi_Minh' };
    await TestBed.configureTestingModule({
      imports: [Account],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: { currentUser: signal(user), logout } },
        { provide: ThemeService, useValue: { preference, resolved, saveState, set: setTheme } },
        { provide: WritingNotifier, useValue: { unseen } },
        { provide: NaturalVoiceService, useValue: voice },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('shows who is logged in and the topic being studied', async () => {
    await open({
      active: {
        topicId: 't1',
        topicName: 'Gia đình',
        level: 'A2',
        completedLessons: 1,
        totalLessons: 8,
        status: 'active',
        effectiveFrom: '2026-09-30',
      },
      others: [],
    });
    expect(text(el.querySelector('.avatar'))).toBe('m');
    expect(text(el.querySelector('.email'))).toBe('minh@example.com');
    expect(text(el.querySelector('.level'))).toBe('Đang học: A2 · Gia đình');
  });

  it('still works when the goal cannot be loaded', async () => {
    await open('error');
    expect(text(el.querySelector('.level'))).toBe('Chưa chọn mục tiêu');
    expect(el.querySelectorAll('nav a').length).toBe(4);
  });

  it('links to the stats, writings, notebook and settings, with new writing results', async () => {
    unseen.set(3);
    await open();
    const links = Array.from(el.querySelectorAll('nav a'));
    expect(links.map((a) => a.getAttribute('href'))).toEqual(['/stats', '/writings', '/vocabulary', '/settings']);
    expect(text(links[1])).toContain('3 mới');
  });

  it('switches dark mode on and off', async () => {
    await open();
    expect(darkSwitch().getAttribute('aria-checked')).toBe('false');
    expect(text(darkSwitch())).toBe('Chế độ tối');

    darkSwitch().click();
    await fixture.whenStable();
    expect(setTheme).toHaveBeenCalledWith('dark');
    expect(darkSwitch().getAttribute('aria-checked')).toBe('true');

    darkSwitch().click();
    expect(setTheme).toHaveBeenLastCalledWith('light');
  });

  it('says when the theme could not be saved to the account', async () => {
    await open();
    saveState.set('error');
    await fixture.whenStable();
    expect(text(el.querySelector('.save-status'))).toContain('Chưa lưu được vào tài khoản');
  });

  it('links to the color page with the color in use, without listing the colors here', async () => {
    localStorage.setItem(PALETTE_STORAGE_KEY, 'jade');
    await open();
    const link = el.querySelector('a[href="/colors"]');
    expect(text(link)).toContain('Màu giao diện');
    expect(text(link)).toContain('Đang dùng: Xanh ngọc');
    expect(el.querySelector('input[name="palette"]')).toBeNull();
    localStorage.clear();
    document.documentElement.removeAttribute('data-palette');
  });

  it('turns the natural voice on and off, saying what it costs and how far it got', async () => {
    await open();
    const toggle = () => Array.from(el.querySelectorAll<HTMLButtonElement>('[role="switch"]')).find((b) => text(b).includes('Giọng đọc tự nhiên'))!;
    expect(toggle().getAttribute('aria-checked')).toBe('false');
    expect(text(toggle())).toContain('Tải khoảng 60MB một lần');

    toggle().click();
    await fixture.whenStable();
    expect(voice.enable).toHaveBeenCalled();
    expect(toggle().getAttribute('aria-checked')).toBe('true');
    expect(text(toggle())).toContain('Đang tải giọng đọc… 40%');

    voiceState.set('error');
    await fixture.whenStable();
    expect(text(toggle())).toContain('đang dùng giọng của trình duyệt');

    toggle().click();
    expect(voice.disable).toHaveBeenCalled();
  });

  it('logs out and goes to the login page', async () => {
    await open();
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    const button = Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Đăng xuất')!;
    button.click();
    await fixture.whenStable();
    expect(logout).toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith('/login');
  });
});
