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
  untracked,
  viewChild,
} from '@angular/core';

import { TOEIC_LEVELS } from '../toeic-admin.logic';

/**
 * The first step of "Vẽ ảnh còn thiếu…" in Kho cụm câu (2026-11-23): which levels to draw, each
 * with the pictures it still misses. A level chosen that already has every picture is told so (it
 * gets no new picture); the other chosen levels get only their missing ones. "Tiếp tục" then asks
 * for the engine. Focus starts on the first box; Escape cancels; Tab stays inside.
 */
@Component({
  selector: 'lu-toeic-level-dialog',
  template: `
    @if (open()) {
      <div class="backdrop">
        <div
          #dialog
          class="dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="level-dialog-title"
          aria-describedby="level-dialog-what"
          tabindex="-1"
          (keydown)="trapTab($event)"
        >
          <h2 class="title" id="level-dialog-title">{{ heading() }}</h2>
          <p class="message" id="level-dialog-what">
            Chọn các mức điểm cần vẽ. Chỉ vẽ ảnh còn thiếu, kể cả cụm nháp.
          </p>
          <fieldset class="levels">
            <legend class="sr-only">Mức điểm cần vẽ</legend>
            @for (l of levels; track l) {
              <label class="level">
                <input type="checkbox" [checked]="chosen().includes(l)" (change)="toggle(l)" />
                <b>{{ l }}</b>
                <span class="count" [class.full]="missingAt(l) === 0">
                  {{ missingAt(l) === 0 ? '✓ Đủ ảnh' : 'Còn ' + missingAt(l) + ' ảnh' }}
                </span>
              </label>
            }
          </fieldset>
          @if (fullChosen().length) {
            <p class="notice" role="status">
              Mức {{ fullChosen().join(', ') }} đã đủ ảnh, sẽ không vẽ lại.
              @if (total() > 0) {
                Các mức còn lại vẫn vẽ ảnh còn thiếu.
              }
            </p>
          }
          @if (chosen().length > 0 && total() > 0) {
            <p class="hint">Sẽ vẽ {{ total() }} ảnh.</p>
          } @else if (chosen().length === 0) {
            <p class="hint">Chọn ít nhất một mức điểm.</p>
          }
          <div class="actions">
            <button type="button" class="btn" (click)="cancelled.emit()">Huỷ</button>
            <button
              type="button"
              class="btn-primary"
              [disabled]="total() === 0"
              (click)="confirm()"
            >
              Tiếp tục
            </button>
          </div>
        </div>
      </div>
    }
  `,
  styleUrls: ['../../../../shared/components/confirm-dialog/confirm-dialog.css'],
  styles: `
    .dialog {
      width: min(100%, 520px);
    }
    .levels {
      display: grid;
      gap: var(--space-1);
      margin: 0;
      padding: 0;
      border: 0;
    }
    .level {
      display: flex;
      gap: var(--space-2);
      align-items: center;
      min-height: 44px;
      font: var(--text-sm) var(--font-ui);
      cursor: pointer;
    }
    .count {
      margin-left: auto;
      font-weight: 700;
    }
    .count.full {
      color: var(--color-ok);
    }
    .notice {
      margin: 0;
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      color: var(--color-warn);
    }
    .hint {
      margin: 0;
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }
    .sr-only {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip: rect(0 0 0 0);
      white-space: nowrap;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class LevelDialog {
  readonly open = input(false);
  readonly heading = input.required<string>();
  /** The pictures missing at each level ("450" → n) of the part. */
  readonly missing = input<Record<string, number>>({});
  /** The levels ticked when the dialog opens (the earlier choice, on "Quay lại"). */
  readonly selected = input<number[]>([]);

  /** The levels chosen (full ones too), at least one of them still missing pictures. */
  readonly confirmed = output<number[]>();
  readonly cancelled = output<void>();

  protected readonly levels = TOEIC_LEVELS;
  protected readonly chosen = signal<number[]>([]);
  /** The chosen levels that already have every picture. */
  protected readonly fullChosen = computed(() =>
    this.chosen().filter((l) => this.missingAt(l) === 0),
  );
  /** The pictures the chosen levels will get. */
  protected readonly total = computed(() =>
    this.chosen().reduce((sum, l) => sum + this.missingAt(l), 0),
  );

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        this.chosen.set([...untracked(this.selected)]);
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        this.dialog()?.nativeElement.querySelector<HTMLInputElement>('input')?.focus();
      }
    });
  }

  protected missingAt(level: number): number {
    return this.missing()[String(level)] ?? 0;
  }

  protected toggle(level: number): void {
    this.chosen.update((ls) =>
      ls.includes(level) ? ls.filter((l) => l !== level) : [...ls, level].sort((a, b) => a - b),
    );
  }

  protected confirm(): void {
    if (this.total() > 0) {
      this.confirmed.emit(this.chosen());
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
        'input:not(:disabled), button:not(:disabled)',
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
