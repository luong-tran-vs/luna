import { ChangeDetectionStrategy, Component, DestroyRef, effect, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { WritingNotifier } from '../../../core/services/writing-notifier.service';

/** How long a toast stays on screen. */
export const TOAST_DURATION = 8000;

/** A short in-app notice at the bottom of the page (F8: a writing has its result). */
@Component({
  selector: 'lu-toast',
  imports: [RouterLink],
  templateUrl: './toast.html',
  styleUrl: './toast.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Toast {
  protected readonly notifier = inject(WritingNotifier);
  private timer?: ReturnType<typeof setTimeout>;

  constructor() {
    effect(() => {
      clearTimeout(this.timer);
      if (this.notifier.toast()) {
        this.timer = setTimeout(() => this.notifier.dismissToast(), TOAST_DURATION);
      }
    });
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
  }
}
