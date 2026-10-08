import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs';

import { AuthService } from '../../core/services/auth.service';
import { ThemeService } from '../../core/services/theme.service';
import { ConnectionStatusBar } from '../../shared/components/connection-status/connection-status';
import { Icon, IconName } from '../../shared/components/icon/icon';

interface AdminLink {
  path: string;
  label: string;
  icon: IconName;
  /** Whether a URL path (no query) belongs to this section. */
  matches: (path: string) => boolean;
}

const ADMIN_LINKS: readonly AdminLink[] = [
  { path: '/admin', label: 'Trang chủ', icon: 'home', matches: (p) => p === '/admin' },
  { path: '/admin/lessons', label: 'Bài học', icon: 'book', matches: (p) => p.startsWith('/admin/lessons') },
  { path: '/admin/topics', label: 'Chủ đề', icon: 'layers', matches: (p) => p.startsWith('/admin/topics') },
  { path: '/admin/words', label: 'Từ vựng', icon: 'notebook', matches: (p) => p.startsWith('/admin/words') },
  { path: '/admin/grammar', label: 'Ngữ pháp', icon: 'grammar', matches: (p) => p.startsWith('/admin/grammar') },
  { path: '/admin/roadmap', label: 'Lộ trình', icon: 'route', matches: (p) => p.startsWith('/admin/roadmap') },
  { path: '/admin/accounts', label: 'Tài khoản', icon: 'user', matches: (p) => p.startsWith('/admin/accounts') },
  { path: '/admin/appearance', label: 'Giao diện', icon: 'palette', matches: (p) => p.startsWith('/admin/appearance') },
  { path: '/admin/tts-lab', label: 'Thử giọng đọc', icon: 'volume', matches: (p) => p.startsWith('/admin/tts-lab') },
  { path: '/admin/stt-lab', label: 'Thử nhận dạng giọng nói', icon: 'mic', matches: (p) => p.startsWith('/admin/stt-lab') },
];

/**
 * Admin area: only content management. From 768px a full-height sidebar on the left holds the
 * brand, the admin's name, the menu with icons, and the dark mode switch and logout at the bottom;
 * on narrow screens it becomes a top bar with the menu as a row that scrolls sideways.
 */
@Component({
  selector: 'lu-admin-layout',
  imports: [RouterOutlet, RouterLink, ConnectionStatusBar, Icon],
  templateUrl: './admin-layout.html',
  styleUrl: './admin-layout.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminLayout {
  private readonly router = inject(Router);
  private readonly theme = inject(ThemeService);
  protected readonly auth = inject(AuthService);

  protected readonly links = ADMIN_LINKS;
  protected readonly path = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => this.currentPath()),
    ),
    { initialValue: this.currentPath() },
  );

  /** The part of the email before "@", as the admin has no display name. */
  protected readonly name = computed(() => this.auth.currentUser()?.email.split('@')[0] ?? '');
  protected readonly initial = computed(() => this.name().charAt(0).toUpperCase() || 'A');
  protected readonly dark = computed(() => this.theme.resolved() === 'dark');

  protected toggleDark(): void {
    this.theme.set(this.dark() ? 'light' : 'dark');
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }

  private currentPath(): string {
    return this.router.url.split(/[?#]/)[0];
  }
}
