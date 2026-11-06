import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { ImageEngine, storedImageEngine, storeImageEngine } from '../../../core/models/word-bank';
import { EnginePicker } from '../engine-picker/engine-picker';

/**
 * Asks what draws a batch of pictures (added 2026-11-06): Gemini or the
 * Codex CLI, remembered as in the word bank; the pictures are then drawn one after another in the
 * background. Focus starts on the choice; Escape cancels; Tab stays inside.
 */
@Component({
  selector: 'lu-engine-dialog',
  imports: [EnginePicker],
  template: `
    @if (open()) {
      <div class="backdrop">
        <div
          #dialog
          class="dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="engine-dialog-title"
          aria-describedby="engine-dialog-what"
          tabindex="-1"
          (keydown)="trapTab($event)"
        >
          <h2 class="title" id="engine-dialog-title">{{ heading() }}</h2>
          <p class="message" id="engine-dialog-what">
            {{ message() }}
          </p>
          @if (note()) {
            <p class="hint">{{ note() }}</p>
          }
          <lu-engine-picker
            name="engine-dialog-engine"
            [value]="chosen()"
            [codexAvailable]="codexAvailable()"
            (changed)="choose($event)"
          />
          <div class="actions">
            <button type="button" class="btn" (click)="cancelled.emit()">Huỷ</button>
            <button type="button" class="btn-primary" (click)="confirmed.emit(chosen())">
              {{ confirmLabel() }}
            </button>
          </div>
        </div>
      </div>
    }
  `,
  styleUrls: ['../confirm-dialog/confirm-dialog.css'],
  styles: `
    .dialog {
      width: min(100%, 560px);
    }

    .hint {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class EngineDialog {
  readonly open = input(false);
  readonly heading = input.required<string>();
  /** What will be drawn; note, when set, a line below it (what is left out). */
  readonly message = input('');
  readonly note = input('');
  readonly confirmLabel = input('Tạo ảnh');
  readonly codexAvailable = input(false);

  readonly confirmed = output<ImageEngine>();
  readonly cancelled = output<void>();

  private readonly engine = signal<ImageEngine>(storedImageEngine());
  /** Codex only where the server offers it. */
  protected readonly chosen = computed<ImageEngine>(() =>
    this.codexAvailable() ? this.engine() : 'gemini',
  );

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        this.engine.set(storedImageEngine());
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        this.dialog()?.nativeElement.querySelector<HTMLInputElement>('input:checked')?.focus();
      }
    });
  }

  protected choose(engine: ImageEngine): void {
    this.engine.set(engine);
    storeImageEngine(engine);
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
