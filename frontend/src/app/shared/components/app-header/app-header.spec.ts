import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { User } from '../../../core/models/user';
import { AuthService } from '../../../core/services/auth.service';
import { ThemeService } from '../../../core/services/theme.service';
import { AppHeader, NavLink } from './app-header';

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
    user.set(null);
    logout.mockClear();
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [AppHeader],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authStub },
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

  it('publishes its height as --header-h for bars that stick under it', async () => {
    let notify: (() => void) | undefined;
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(cb: () => void) {
          notify = cb;
        }
        observe = vi.fn();
        disconnect = vi.fn();
      },
    );
    const header = TestBed.createComponent(AppHeader);
    await header.whenStable();
    vi.spyOn(header.nativeElement as HTMLElement, 'offsetHeight', 'get').mockReturnValue(72);
    notify!();
    expect(document.documentElement.style.getPropertyValue('--header-h')).toBe('72px');
    header.destroy();
    vi.unstubAllGlobals();
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
    await loginAs('member');
    const logoutButton = Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.includes('Đăng xuất'));
    expect(logoutButton?.closest('.actions')).not.toBeNull();
    expect(logoutButton?.closest('nav')).toBeNull();
  });

  describe('account', () => {
    const learnerLinks: NavLink[] = [
      { path: '/lessons', label: 'Bài học' },
      { path: '/writings', label: 'Bài viết', badge: 0, badgeLabel: 'kết quả mới' },
      { path: '/vocabulary', label: 'Sổ từ' },
    ];

    it('shows no account actions when logged out', () => {
      fixture.componentRef.setInput('links', learnerLinks);
      expect(el.textContent).not.toContain('Đăng xuất');
      expect(el.querySelector('a[href="/lessons"]')).toBeNull();
    });

    it('shows the links given by the layout when logged in', async () => {
      fixture.componentRef.setInput('links', learnerLinks);
      await loginAs('member');
      const links = Array.from(el.querySelectorAll('nav a')).map((a) => [a.getAttribute('href'), a.textContent?.trim()]);
      expect(links).toEqual([
        ['/lessons', 'Bài học'],
        ['/writings', 'Bài viết'],
        ['/vocabulary', 'Sổ từ'],
      ]);
    });

    it('shows a badge with its count in the accessible name', async () => {
      await loginAs('member');
      const link = () => el.querySelector('a[href="/writings"]')!;
      fixture.componentRef.setInput('links', learnerLinks);
      await fixture.whenStable();
      expect(link().querySelector('.badge')).toBeNull();
      expect(link().getAttribute('aria-label')).toBeNull();

      fixture.componentRef.setInput('links', [{ ...learnerLinks[1], badge: 2 }]);
      await fixture.whenStable();
      expect(link().querySelector('.badge')?.textContent?.trim()).toBe('2');
      expect(link().getAttribute('aria-label')).toBe('Bài viết, 2 kết quả mới');
    });

    it('links the brand to the area home and names the area', async () => {
      expect(el.querySelector('.brand')?.getAttribute('href')).toBe('/');
      expect(el.querySelector('.area')).toBeNull();

      fixture.componentRef.setInput('home', '/admin');
      fixture.componentRef.setInput('area', 'Quản trị');
      await loginAs('admin');
      expect(el.querySelector('.brand')?.getAttribute('href')).toBe('/admin');
      expect(el.querySelector('.area')?.textContent?.trim()).toBe('Quản trị');
      expect(el.querySelector('nav')?.getAttribute('aria-label')).toBe('Quản trị');
    });

    it('shows the email and a logout button when logged in', async () => {
      await loginAs('member');
      const email = el.querySelector('.email');
      expect(email?.textContent?.trim()).toBe('rat-dai-ten@example.com');
      expect(email?.getAttribute('title')).toBe('rat-dai-ten@example.com');
      expect(el.textContent).toContain('Đăng xuất');
    });

    it('logs out and goes to /login', async () => {
      const navigate = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
      await loginAs('member');

      Array.from(el.querySelectorAll('button'))
        .find((b) => b.textContent?.includes('Đăng xuất'))!
        .click();
      await new Promise((resolve) => setTimeout(resolve));
      await fixture.whenStable();

      expect(logout).toHaveBeenCalled();
      expect(navigate).toHaveBeenCalledWith('/login');
    });
  });
});
