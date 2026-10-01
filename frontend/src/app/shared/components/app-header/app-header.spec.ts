import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { User } from '../../../core/models/user';
import { AuthService } from '../../../core/services/auth.service';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';
import { ThemeService } from '../../../core/services/theme.service';
import { AppHeader } from './app-header';

describe('AppHeader', () => {
  let fixture: ComponentFixture<AppHeader>;
  let el: HTMLElement;

  const trigger = () => el.querySelector<HTMLButtonElement>('.theme-trigger')!;
  // The menu renders in the CDK overlay container, outside the component element.
  const items = () => Array.from(document.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]'));
  const openMenu = async () => {
    trigger().click();
    await fixture.whenStable();
  };

  const user = signal<User | null>(null);
  const unseen = signal(0);
  const logout = vi.fn(async () => user.set(null));
  const authStub = {
    currentUser: user,
    isLoggedIn: computed(() => user() !== null),
    isAdmin: computed(() => user()?.role === 'admin'),
    logout,
  };
  const loginAs = async (role: User['role']) => {
    user.set({ id: '1', email: 'rat-dai-ten@example.com', role, timezone: 'Asia/Ho_Chi_Minh' });
    await fixture.whenStable();
  };

  beforeEach(async () => {
    unseen.set(0);
    user.set(null);
    logout.mockClear();
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [AppHeader],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authStub },
        { provide: WritingNotifier, useValue: { unseen } },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AppHeader);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('renders a header with the app name "Luna"', () => {
    const header = el.querySelector('header');
    expect(header).not.toBeNull();
    expect(header?.textContent).toContain('Luna');
  });

  it('shows only the current theme on a menu button', () => {
    const theme = TestBed.inject(ThemeService);
    expect(trigger().textContent?.trim()).toBe(theme.resolved() === 'dark' ? 'Tối' : 'Sáng');
    expect(trigger().getAttribute('aria-haspopup')).toBe('menu');
    expect(items()).toHaveLength(0);
  });

  it('opens a menu with only Sáng and Tối, the current one checked', async () => {
    TestBed.inject(ThemeService).set('dark');
    await openMenu();

    expect(items().map((i) => i.textContent?.trim())).toEqual(['Sáng', 'Tối']);
    expect(items().map((i) => i.getAttribute('aria-checked'))).toEqual(['false', 'true']);
  });

  it('selects a theme from the menu and closes it', async () => {
    const theme = TestBed.inject(ThemeService);
    const spy = vi.spyOn(theme, 'set');
    await openMenu();

    items()[1].click();
    await fixture.whenStable();

    expect(spy).toHaveBeenCalledWith('dark');
    expect(trigger().textContent?.trim()).toBe('Tối');
    expect(items()).toHaveLength(0);
  });

  it('puts the logout button next to the theme button, outside the scrolling nav', async () => {
    await loginAs('learner');
    const logoutButton = Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.includes('Đăng xuất'));
    expect(logoutButton?.closest('.actions')).not.toBeNull();
    expect(logoutButton?.closest('nav')).toBeNull();
  });

  describe('account', () => {
    it('shows no account actions when logged out', () => {
      expect(el.textContent).not.toContain('Đăng xuất');
      expect(el.querySelector('a[href="/admin"]')).toBeNull();
    });

    it('links to the notebook when logged in', async () => {
      expect(el.querySelector('a[href="/vocabulary"]')).toBeNull();
      await loginAs('learner');
      expect(el.querySelector('a[href="/vocabulary"]')?.textContent?.trim()).toBe('Sổ từ');
      expect(el.querySelector('a[href="/settings"]')?.textContent?.trim()).toBe('Cài đặt');
      expect(el.querySelector('a[href="/lessons"]')?.textContent?.trim()).toBe('Bài học');
    });

    it('links to the writings with the number of new results (F8)', async () => {
      await loginAs('learner');
      const link = () => el.querySelector('a[href="/writings"]')!;
      expect(link().textContent?.trim()).toBe('Bài viết');
      expect(link().getAttribute('aria-label')).toBeNull();
      unseen.set(2);
      await fixture.whenStable();
      expect(link().querySelector('.badge')?.textContent?.trim()).toBe('2');
      expect(link().getAttribute('aria-label')).toBe('Bài viết, 2 kết quả mới');
      unseen.set(0);
      await fixture.whenStable();
      expect(link().querySelector('.badge')).toBeNull();
    });

    it('shows the email and a logout button when logged in', async () => {
      await loginAs('learner');
      const email = el.querySelector('.email');
      expect(email?.textContent?.trim()).toBe('rat-dai-ten@example.com');
      expect(email?.getAttribute('title')).toBe('rat-dai-ten@example.com');
      expect(el.textContent).toContain('Đăng xuất');
    });

    it('logs out and goes to /login', async () => {
      const navigate = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
      await loginAs('learner');

      Array.from(el.querySelectorAll('button'))
        .find((b) => b.textContent?.includes('Đăng xuất'))!
        .click();
      await new Promise((resolve) => setTimeout(resolve));
      await fixture.whenStable();

      expect(logout).toHaveBeenCalled();
      expect(navigate).toHaveBeenCalledWith('/login');
    });

    it('shows the admin link only to admins', async () => {
      await loginAs('learner');
      expect(el.querySelector('a[href="/admin"]')).toBeNull();

      await loginAs('admin');
      expect(el.querySelector('a[href="/admin"]')?.textContent?.trim()).toBe('Quản trị');
    });
  });
});
