import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs';

import { ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { ConnectionStatusBar } from '../../shared/components/connection-status/connection-status';
import { Icon, IconName } from '../../shared/components/icon/icon';
import { Toast } from '../../shared/components/toast/toast';

interface Tab {
  path: string;
  label: string;
  icon: IconName;
  /** Whether a URL path (no query) belongs to this tab. */
  matches: (path: string) => boolean;
}

const TABS: readonly Tab[] = [
  { path: '/', label: 'Trang chủ', icon: 'home', matches: (p) => p === '/' },
  {
    path: '/goal',
    label: 'Khóa học',
    icon: 'book',
    matches: (p) => p.startsWith('/lessons') || p === '/goal',
  },
  { path: '/vocabulary/review', label: 'Ôn tập', icon: 'review', matches: (p) => p.startsWith('/vocabulary') },
  {
    path: '/account',
    label: 'Tài khoản',
    icon: 'user',
    matches: (p) => ['/account', '/settings', '/stats', '/writings'].some((s) => p === s || p.startsWith(s + '/')),
  },
];

/** Studying a lesson takes the whole phone screen, like the sketch: no tab bar there. */
function isFocusPage(path: string): boolean {
  return /^\/lessons\/[^/]+\/(read|listen|write)$/.test(path);
}

/**
 * Learning area (layout of the client sketch): four tabs, as a bar at the bottom of a phone and at
 * the top from 768px, where Tài khoản sits on the right next to a dark mode switch. Admins never
 * get here (learnerGuard).
 */
@Component({
  selector: 'lu-learner-layout',
  imports: [RouterOutlet, RouterLink, ConnectionStatusBar, Icon, Toast],
  templateUrl: './learner-layout.html',
  styleUrl: './learner-layout.css',
  host: { '[class.focus]': 'focus()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LearnerLayout {
  private readonly router = inject(Router);
  private readonly theme = inject(ThemeService);
  protected readonly notifier = inject(WritingNotifier);

  protected readonly tabs = TABS;
  protected readonly path = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => this.currentPath()),
    ),
    { initialValue: this.currentPath() },
  );
  protected readonly focus = computed(() => isFocusPage(this.path()));
  protected readonly dark = computed(() => this.theme.resolved() === 'dark');

  /** Same switch as "Chế độ tối" in Tài khoản: saved to the account when logged in. */
  protected toggleDark(): void {
    this.theme.set(this.dark() ? 'light' : 'dark');
  }

  private currentPath(): string {
    return this.router.url.split(/[?#]/)[0];
  }
}
