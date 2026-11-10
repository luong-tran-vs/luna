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
 * Codex CLI, picked each time (no default since 2026-11-10); the pictures are then drawn one after another in the
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
          @if (chosen() === '') {
            <p class="hint">Chọn Gemini hoặc Codex CLI để tạo ảnh.</p>
          }
          <div class="actions">
            <button type="button" class="btn" (click)="cancelled.emit()">Huỷ</button>
            <button
              type="button"
              class="btn-primary"
              [disabled]="chosen() === ''"
              (click)="confirm()"
            >
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
  /**
   * The admin must pick the engine each time (asked 2026-11-10: no default, for every picture):
   * nothing is chosen when the dialog opens and "Tạo ảnh" waits for a choice. false preselects the
   * last choice of this browser.
   */
  readonly requireChoice = input(true);

  readonly confirmed = output<ImageEngine>();
  readonly cancelled = output<void>();

  private readonly engine = signal<ImageEngine | ''>(storedImageEngine());
  /** Codex only where the server offers it; "" while none is chosen. */
  protected readonly chosen = computed<ImageEngine | ''>(() => {
    const e = this.engine();
    if (e === 'codex' && !this.codexAvailable()) {
      return this.requireChoice() ? '' : 'gemini';
    }
    return e;
  });

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        this.engine.set(this.requireChoice() ? '' : storedImageEngine());
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        const root = this.dialog()?.nativeElement;
        (
          root?.querySelector<HTMLInputElement>('input:checked') ??
          root?.querySelector<HTMLInputElement>('input:not(:disabled)')
        )?.focus();
      }
    });
  }

  protected confirm(): void {
    const e = this.chosen();
    if (e !== '') {
      this.confirmed.emit(e);
    }
  }

  protected choose(engine: ImageEngine): void {
    this.engine.set(engine);
    if (!this.requireChoice()) {
      storeImageEngine(engine);
    }
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
      this.dialog()?.nativeElement.querySelectorAll<HTMLElement>(
        this.chosen() === ''
          ? 'input:not(:disabled), button:not(:disabled)'
          : 'input:checked, button',
      ) ?? [],
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
