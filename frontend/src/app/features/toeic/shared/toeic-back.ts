import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../../shared/components/icon/icon';

/** The back button at the top of a TOEIC page: a round arrow and the name of where it goes. */
@Component({
  selector: 'lu-toeic-back',
  imports: [RouterLink, Icon],
  template: `
    <nav class="screen-bar" aria-label="Điều hướng">
      <a class="round-btn" [routerLink]="link()" [attr.aria-label]="'Quay lại ' + label()">
        <lu-icon name="arrow-left" />
      </a>
      <span class="back-label" aria-hidden="true">{{ label() }}</span>
    </nav>
  `,
  styles: `
    :host {
      display: block;
    }
    .screen-bar {
      min-height: 0;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToeicBack {
  /** Where the button goes (a router link). */
  readonly link = input.required<string | (string | number)[]>();
  /** The page it goes to, e.g. "Luyện thi TOEIC". */
  readonly label = input.required<string>();
}
