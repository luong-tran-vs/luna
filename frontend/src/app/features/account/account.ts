import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { GoalView } from '../../core/models/study';
import { AuthService } from '../../core/services/auth.service';
import { PALETTES, PaletteService } from '../../core/services/palette.service';
import { StudyApiService } from '../../core/services/study-api.service';
import { ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Icon, IconName } from '../../shared/components/icon/icon';

interface AccountLink {
  path: string;
  label: string;
  hint: string;
  icon: IconName;
}

const LINKS: readonly AccountLink[] = [
  { path: '/stats', label: 'Thống kê', hint: 'Từ đã học, độ chính xác, bài viết', icon: 'chart' },
  { path: '/writings', label: 'Bài viết', hint: 'Kết quả chấm bài viết', icon: 'pencil' },
  { path: '/vocabulary', label: 'Sổ từ', hint: 'Các từ đã lưu', icon: 'notebook' },
  { path: '/settings', label: 'Cài đặt học tập', hint: 'Số thẻ ôn, múi giờ, xuất dữ liệu', icon: 'sliders' },
];

/**
 * The account tab (client sketch, screen 11): who is logged in, links to the other pages, dark mode,
 * a link to the color page, logout.
 */
@Component({
  selector: 'lu-account',
  imports: [Icon, RouterLink],
  templateUrl: './account.html',
  styleUrl: './account.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Account {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);
  protected readonly notifier = inject(WritingNotifier);
  private readonly palette = inject(PaletteService);
  private readonly router = inject(Router);

  protected readonly links = LINKS;
  protected readonly goal = signal<GoalView | null>(null);
  protected readonly dark = computed(() => this.theme.resolved() === 'dark');
  protected readonly paletteName = computed(() => PALETTES.find((p) => p.id === this.palette.palette())?.name ?? '');
  protected readonly initial = computed(() => this.auth.currentUser()?.email.charAt(0) ?? '?');

  constructor() {
    inject(StudyApiService)
      .goals()
      // The level line is extra information: without it the page still works.
      .subscribe({ next: (g) => this.goal.set(g.active), error: () => undefined });
  }

  protected toggleDark(): void {
    this.theme.set(this.dark() ? 'light' : 'dark');
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }
}
