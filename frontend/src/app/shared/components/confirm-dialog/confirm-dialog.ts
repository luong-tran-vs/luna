import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  input,
  output,
  viewChild,
} from '@angular/core';

/**
 * Modal confirmation shown in the page (not window.confirm, so it can be styled and tested).
 * Focus moves to the cancel button when it opens; Escape cancels; Tab stays inside.
 */
@Component({
  selector: 'lu-confirm-dialog',
  templateUrl: './confirm-dialog.html',
  styleUrl: './confirm-dialog.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class ConfirmDialog {
  readonly open = input(false);
  readonly title = input.required<string>();
  readonly message = input('');
  readonly confirmLabel = input('Xác nhận');
  readonly cancelLabel = input('Huỷ');
  readonly danger = input(false);
  /** Turns the confirm button off, e.g. while nothing is ticked (added 2026-11-12). */
  readonly confirmDisabled = input(false);

  readonly confirmed = output<void>();
  readonly cancelled = output<void>();

  private readonly cancelButton = viewChild<ElementRef<HTMLButtonElement>>('cancel');
  private readonly confirmButton = viewChild<ElementRef<HTMLButtonElement>>('confirm');
  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    afterRenderEffect(() => {
      if (this.open()) {
        this.cancelButton()?.nativeElement.focus();
      }
    });
  }

  protected onEscape(): void {
    if (this.open()) {
      this.cancelled.emit();
    }
  }

  /** Keeps Tab focus inside while open: the buttons, and what a page put in (ticks). */
  protected trapTab(event: KeyboardEvent): void {
    if (event.key !== 'Tab') {
      return;
    }
    const items = Array.from(
      this.dialog()?.nativeElement.querySelectorAll<HTMLElement>(
        'input:not(:disabled), select:not(:disabled), button:not(:disabled), a[href]',
      ) ?? [],
    );
    const first = items[0] ?? this.cancelButton()?.nativeElement;
    const last = items[items.length - 1] ?? this.confirmButton()?.nativeElement;
    const active = document.activeElement;
    if (event.shiftKey && active === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && active === last) {
      event.preventDefault();
      first?.focus();
    }
  }
}
