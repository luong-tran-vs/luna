import { ChangeDetectionStrategy, Component, inject } from '@angular/core';

import { ThemePreference, ThemeService } from '../../../core/services/theme.service';
import { PalettePicker } from '../../../shared/components/palette-picker/palette-picker';
import { StatusChip } from '../status-chip/status-chip';

const MODES: readonly { id: ThemePreference; label: string }[] = [
  { id: 'light', label: 'Sáng' },
  { id: 'dark', label: 'Tối' },
  { id: 'system', label: 'Theo thiết bị' },
];

/**
 * Admin appearance: light/dark mode and the color palette of the app. A choice applies at once,
 * so the page itself is the preview; a sample card shows the main parts.
 */
@Component({
  selector: 'lu-appearance',
  imports: [PalettePicker, StatusChip],
  templateUrl: './appearance.html',
  styleUrl: './appearance.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Appearance {
  protected readonly theme = inject(ThemeService);

  protected readonly modes = MODES;

  protected chooseMode(id: ThemePreference): void {
    this.theme.set(id);
  }
}
