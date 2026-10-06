import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../shared/components/icon/icon';
import { PalettePicker } from '../../shared/components/palette-picker/palette-picker';

/** The learner's color page, opened from the account tab: picks the app's color palette. */
@Component({
  selector: 'lu-colors',
  imports: [Icon, PalettePicker, RouterLink],
  template: `<section class="screen">
    <header class="screen-bar">
      <a class="round-btn" routerLink="/account" aria-label="Quay lại Tài khoản"><lu-icon name="arrow-left" /></a>
      <h1 class="screen-title" id="colors-heading">Màu giao diện</h1>
    </header>
    <p class="hint">Chọn màu bạn thích. Cả app đổi màu ngay và được nhớ trên trình duyệt này.</p>
    <lu-palette-picker labelledBy="colors-heading" />
  </section>`,
  styles: `
    .hint {
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Colors {}
