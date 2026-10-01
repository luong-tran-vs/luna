import { CdkMenu, CdkMenuItemRadio, CdkMenuTrigger } from '@angular/cdk/menu';
import { ConnectedPosition } from '@angular/cdk/overlay';
import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../core/services/auth.service';
import { ResolvedTheme, ThemeService } from '../../../core/services/theme.service';
import { WritingNotifier } from '../../../core/services/writing-notifier.service';

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

@Component({
  selector: 'lu-app-header',
  imports: [RouterLink, NgTemplateOutlet, CdkMenu, CdkMenuItemRadio, CdkMenuTrigger],
  templateUrl: './app-header.html',
  styleUrl: './app-header.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppHeader {
  protected readonly theme = inject(ThemeService);
  protected readonly auth = inject(AuthService);
  protected readonly notifier = inject(WritingNotifier);
  private readonly router = inject(Router);
  protected readonly options = OPTIONS;
  protected readonly menuPositions = MENU_POSITIONS;
  protected readonly current = computed(
    () => OPTIONS.find((o) => o.value === this.theme.resolved()) ?? OPTIONS[0],
  );

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }

  protected select(value: ResolvedTheme): void {
    this.theme.set(value);
  }
}
