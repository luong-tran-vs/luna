import { CdkMenu, CdkMenuItemRadio, CdkMenuTrigger } from '@angular/cdk/menu';
import { ConnectedPosition } from '@angular/cdk/overlay';
import { NgTemplateOutlet } from '@angular/common';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  input,
} from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../core/services/auth.service';
import { ResolvedTheme, ThemeService } from '../../../core/services/theme.service';

/** A link of the header menu; badge is a count shown next to it (e.g. new writing results). */
export interface NavLink {
  path: string;
  label: string;
  badge?: number;
  /** Read after the count by screen readers, e.g. "kết quả mới". */
  badgeLabel?: string;
}

interface ThemeOption {
  value: ResolvedTheme;
  label: string;
}

/** Only light and dark are offered; until the user picks one, the device setting applies. */
const OPTIONS: readonly ThemeOption[] = [
  { value: 'light', label: 'Sáng' },
  { value: 'dark', label: 'Tối' },
];

/** Menu opens below the button, right-aligned; flips above when there is no room. */
const MENU_POSITIONS: ConnectedPosition[] = [
  { originX: 'end', originY: 'bottom', overlayX: 'end', overlayY: 'top', offsetY: 4 },
  { originX: 'end', originY: 'top', overlayX: 'end', overlayY: 'bottom', offsetY: -4 },
];

/**
 * Header shell shared by the learner, admin and public layouts: brand, theme menu and logout. Each
 * layout passes its own menu links, so the learner and admin areas never show each other's pages.
 */
@Component({
  selector: 'lu-app-header',
  imports: [RouterLink, NgTemplateOutlet, CdkMenu, CdkMenuItemRadio, CdkMenuTrigger],
  templateUrl: './app-header.html',
  styleUrl: './app-header.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppHeader {
  /** Where the brand links to: the home page of the area. */
  readonly home = input('/');
  /** Area name shown next to the brand (e.g. "Quản trị"); also names the menu. */
  readonly area = input<string | null>(null);
  readonly links = input<readonly NavLink[]>([]);

  protected readonly theme = inject(ThemeService);
  protected readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly options = OPTIONS;
  protected readonly menuPositions = MENU_POSITIONS;
  protected readonly current = computed(
    () => OPTIONS.find((o) => o.value === this.theme.resolved()) ?? OPTIONS[0],
  );

  constructor() {
    // Publish the header height as --header-h so sticky bars below it (the admin sidebar) start
    // right under it, also when the header wraps to two rows on narrow screens.
    const host = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      if (typeof ResizeObserver === 'undefined') {
        return;
      }
      const root = document.documentElement;
      const observer = new ResizeObserver(() => root.style.setProperty('--header-h', `${host.offsetHeight}px`));
      observer.observe(host);
      destroyRef.onDestroy(() => observer.disconnect());
    });
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }

  protected select(value: ResolvedTheme): void {
    this.theme.set(value);
  }
}
