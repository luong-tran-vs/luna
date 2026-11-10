import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs';

import { FocusMode } from '../../core/services/focus-mode.service';
import { ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { ConnectionStatusBar } from '../../shared/components/connection-status/connection-status';
import { Icon, IconName } from '../../shared/components/icon/icon';
import { Toast } from '../../shared/components/toast/toast';
import { PracticeMenu } from './practice-menu/practice-menu';

interface Tab {
  path: string;
  label: string;
  icon: IconName;
  /** Whether a URL path (no query) belongs to this tab. */
  matches: (path: string) => boolean;
  /** Luyện: the raised middle button opening the practice card (PracticeMenu), not a link. */
  practice?: boolean;
}

const TABS: readonly Tab[] = [
  { path: '/', label: 'Trang chủ', icon: 'home', matches: (p) => p === '/' },
  {
    path: '/lessons',
    label: 'Học',
    icon: 'book',
    matches: (p) => p.startsWith('/lessons') || p === '/goal',
  },
  // Luyện: the raised middle button (design/de-xuat-nut-giua-noi-len.png, option 3, 2026-11-10).
  {
    path: '',
    label: 'Luyện',
    icon: 'sparkles',
    practice: true,
    matches: isPracticePath,
  },
  {
    path: '/vocabulary/review',
    label: 'Ôn tập',
    icon: 'review',
    matches: (p) => p.startsWith('/vocabulary') && !isPracticePath(p),
  },
  {
    path: '/account',
    label: 'Tôi',
    icon: 'user',
    matches: (p) =>
      ['/account', '/settings', '/stats', '/writings'].some(
        (s) => p === s || p.startsWith(s + '/'),
      ),
  },
];

/** The pages opened from Luyện: grammar, the irregular verbs and talking with the AI. */
function isPracticePath(path: string): boolean {
  return ['/grammar', '/irregular-verbs', '/vocabulary/talk'].some(
    (s) => path === s || path.startsWith(s + '/'),
  );
}

/**
 * Studying a lesson, a grammar point, a verb lesson or a verb review, and talking with the AI, take
 * the whole phone screen, like the sketch: no tab bar there, so the step buttons and the answer
 * fields are not squeezed under it. Each of these pages has its own close button.
 */
export function isFocusPage(path: string): boolean {
  return (
    /^\/lessons\/[^/]+(\/(read|listen|write))?$/.test(path) ||
    /^\/grammar\/[^/]+$/.test(path) ||
    /^\/irregular-verbs\/(lessons\/[^/]+|review|listen)$/.test(path) ||
    /^\/vocabulary\/talk\/[^/]+$/.test(path)
  );
}

/**
 * Learning area (layout of the client sketch): five tabs (Trang chủ first, Tôi last), as a bar at the bottom of phones and
 * tablets and at the top from 1024px, where Tôi sits on the right next to a dark mode switch.
 * Admins never get here (learnerGuard).
 */
@Component({
  selector: 'lu-learner-layout',
  imports: [RouterOutlet, RouterLink, ConnectionStatusBar, Icon, Toast, PracticeMenu],
  templateUrl: './learner-layout.html',
  styleUrl: './learner-layout.css',
  host: { '[class.focus]': 'focus()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LearnerLayout {
  private readonly router = inject(Router);
  private readonly theme = inject(ThemeService);
  private readonly focusMode = inject(FocusMode);
  protected readonly notifier = inject(WritingNotifier);

  protected readonly tabs = TABS;
  protected readonly path = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => this.currentPath()),
    ),
    { initialValue: this.currentPath() },
  );
  /** The place (0–4) of the current link tab, -1 when none or Luyện: where the droplet sits. */
  protected readonly current = computed(() =>
    TABS.findIndex((t) => !t.practice && t.matches(this.path())),
  );
  protected readonly focus = computed(() => isFocusPage(this.path()) || this.focusMode.on());
  protected readonly dark = computed(() => this.theme.resolved() === 'dark');

  /** Same switch as "Chế độ tối" in Tài khoản: saved to the account when logged in. */
  protected toggleDark(): void {
    this.theme.set(this.dark() ? 'light' : 'dark');
  }

  private currentPath(): string {
    return this.router.url.split(/[?#]/)[0];
  }
}
