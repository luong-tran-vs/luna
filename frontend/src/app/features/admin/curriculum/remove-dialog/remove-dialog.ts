import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';

/**
 * Asks how to take a lesson out of the course (added 2026-10-26): only out of its slot and roadmap,
 * kept as a draft in the lesson list (the default), or deleted for good. Focus starts on the choice;
 * Escape cancels; Tab stays inside.
 */
@Component({
  selector: 'lu-remove-dialog',
  templateUrl: './remove-dialog.html',
  styleUrls: [
    '../../../../shared/components/confirm-dialog/confirm-dialog.css',
    './remove-dialog.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class RemoveDialog {
  readonly open = input(false);
  /** The lesson's title, and whether learners see it now. */
  readonly lessonTitle = input.required<string>();
  readonly published = input(false);

  /** true deletes the lesson, false keeps it as a draft. */
  readonly confirmed = output<boolean>();
  readonly cancelled = output<void>();

  protected readonly del = signal(false);

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    // Each opening starts on the safe choice.
    effect(() => {
      if (this.open()) {
        this.del.set(false);
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        this.dialog()?.nativeElement.querySelector<HTMLInputElement>('input:checked')?.focus();
      }
    });
  }

  protected onEscape(): void {
    if (this.open()) {
      this.cancelled.emit();
    }
  }

  /** Keeps Tab focus inside the dialog while open. */
  protected trapTab(event: KeyboardEvent): void {
    if (event.key !== 'Tab') {
      return;
    }
    const items = Array.from(
      this.dialog()?.nativeElement.querySelectorAll<HTMLElement>('input:checked, button') ?? [],
    );
    const first = items[0];
    const last = items[items.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  }
}
