import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  forwardRef,
  input,
  linkedSignal,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';

/** One choice of the list. */
export interface SearchOption {
  value: string;
  label: string;
  /** A small badge after the label (not searched). */
  hint?: string;
  /** Colored counts after the label, e.g. what the topic's words lack (not searched). */
  badges?: readonly SearchBadge[];
}

/** A count with a colored dot; `label` says what it counts, for screen readers and the tooltip. */
export interface SearchBadge {
  text: string;
  label: string;
  /** A CSS color, e.g. var(--color-bad). */
  color: string;
}

/**
 * A select you can type into to filter its choices (ARIA combobox with a listbox). Typing ignores
 * case and Vietnamese accents ("gia dinh" finds "Gia đình"); arrows move, Enter picks, Escape
 * closes without changing. When closed the box shows the chosen label, so a choice the parent
 * refuses (it keeps [value]) shows the previous one again.
 */
@Component({
  selector: 'lu-search-select',
  template: `
    <div class="box">
      <input
        #box
        type="text"
        role="combobox"
        autocomplete="off"
        [id]="inputId()"
        [attr.name]="name()"
        [attr.aria-expanded]="open()"
        [attr.aria-controls]="listId()"
        aria-autocomplete="list"
        [attr.aria-describedby]="describedBy()"
        [attr.aria-invalid]="invalid() || null"
        [class.invalid]="invalid()"
        [disabled]="disabled()"
        [attr.aria-activedescendant]="open() && active() >= 0 ? optionId(active()) : null"
        [placeholder]="open() ? 'Gõ để tìm…' : placeholder()"
        [value]="open() ? query() : currentLabel()"
        (focus)="show()"
        (click)="show()"
        (input)="search($any($event.target).value)"
        (keydown)="onKey($event)"
        (blur)="onBlur()"
      />
      <span class="caret" aria-hidden="true">▾</span>
    </div>
    @if (open()) {
      <ul class="options" role="listbox" [id]="listId()">
        @for (o of filtered(); track o.value; let i = $index) {
          <!-- Keys stay on the combobox input (aria-activedescendant), so options take only the mouse. -->
          <!-- eslint-disable-next-line @angular-eslint/template/click-events-have-key-events, @angular-eslint/template/interactive-supports-focus -->
          <li
            role="option"
            class="option"
            [id]="optionId(i)"
            [attr.data-value]="o.value"
            [attr.aria-selected]="o.value === current()"
            [class.active]="i === active()"
            (mousedown)="$event.preventDefault()"
            (click)="pick(o)"
            (mouseenter)="active.set(i)"
          >
            <span class="label">{{ o.label }}</span>
            @if (o.hint) {
              <span class="hint"><span class="visually-hidden">, </span>{{ o.hint }}</span>
            }
            @if (o.badges?.length) {
              <span class="badges">
                @for (b of o.badges; track b.label) {
                  <span class="badge" [style.--badge]="b.color" [title]="b.label">
                    <span class="visually-hidden">, </span>{{ b.text }}<span class="visually-hidden"> {{ b.label }}</span>
                  </span>
                }
              </span>
            }
          </li>
        } @empty {
          <li class="empty" role="presentation">Không tìm thấy</li>
        }
      </ul>
    }
  `,
  styles: `
    :host {
      position: relative;
      display: block;
    }

    .box {
      position: relative;
    }

    input {
      box-sizing: border-box;
      width: 100%;
      min-height: 44px;
      padding: var(--space-2) var(--space-6) var(--space-2) var(--space-3);
      font: var(--text-base) var(--font-ui);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-text-muted);
      border-radius: var(--radius-md);
      cursor: pointer;
    }

    input[aria-expanded='true'] {
      cursor: text;
    }

    input.invalid {
      border-color: var(--color-bad);
    }

    input:disabled {
      cursor: not-allowed;
      opacity: 0.6;
    }

    .caret {
      position: absolute;
      top: 50%;
      right: var(--space-3);
      color: var(--color-text-muted);
      transform: translateY(-50%);
      pointer-events: none;
    }

    .options {
      position: absolute;
      z-index: 20;
      top: calc(100% + var(--space-1));
      right: 0;
      left: 0;
      max-height: 320px;
      margin: 0;
      padding: var(--space-1);
      overflow-y: auto;
      list-style: none;
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
      box-shadow: var(--shadow-card);
      animation: drop 120ms ease-out;
    }

    .option {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: space-between;
      gap: var(--space-1) var(--space-2);
      padding: var(--space-2) var(--space-3);
      border-radius: var(--radius-md);
      cursor: pointer;
    }

    .option.active {
      background: var(--color-primary-soft);
      color: var(--color-primary-ink);
    }

    .option[aria-selected='true'] {
      font-weight: 600;
    }

    .badges {
      display: inline-flex;
      gap: var(--space-1);
    }

    .badge {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 0 var(--space-2);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      font-variant-numeric: tabular-nums;
      color: var(--color-text);
      background: color-mix(in srgb, var(--badge) 14%, transparent);
      border: 1px solid var(--badge);
      border-radius: 999px;
    }

    .badge::before {
      width: 8px;
      height: 8px;
      background: var(--badge);
      border-radius: 50%;
      content: '';
    }

    .hint {
      padding: 0 var(--space-2);
      font: var(--text-sm) var(--font-ui);
      font-weight: 400;
      color: var(--color-text);
      border: 1px solid var(--color-warn);
      border-radius: 999px;
    }

    .empty {
      padding: var(--space-2) var(--space-3);
      color: var(--color-text-muted);
    }

    @keyframes drop {
      from {
        opacity: 0;
        transform: translateY(-4px);
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .options {
        animation: none;
      }
    }
  `,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => SearchSelect), multi: true }],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SearchSelect implements ControlValueAccessor {
  readonly options = input.required<readonly SearchOption[]>();
  readonly value = input('');
  readonly inputId = input.required<string>();
  readonly name = input<string | null>(null);
  readonly placeholder = input('');
  /** Ids of the hint and error under the box. */
  readonly describedBy = input<string | null>(null);
  readonly invalid = input(false);
  readonly changed = output<string>();

  private readonly box = viewChild.required<ElementRef<HTMLInputElement>>('box');

  /**
   * The chosen value: [value], or the form control's. Without a form a pick only emits (changed),
   * so a parent that refuses it keeps the box on [value].
   */
  protected readonly current = linkedSignal(() => this.value());
  private onChange: ((value: string) => void) | null = null;
  private onTouched: () => void = () => undefined;
  protected readonly disabled = signal(false);

  protected readonly open = signal(false);
  protected readonly query = signal('');
  protected readonly active = signal(-1);
  protected readonly listId = computed(() => `${this.inputId()}-list`);
  protected readonly currentLabel = computed(() => this.options().find((o) => o.value === this.current())?.label ?? '');

  writeValue(value: unknown): void {
    this.current.set(typeof value === 'string' ? value : '');
  }

  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled.set(disabled);
  }

  protected onBlur(): void {
    this.close();
    this.onTouched();
  }
  protected readonly filtered = computed(() => {
    const q = fold(this.query().trim());
    return q ? this.options().filter((o) => fold(o.label).includes(q)) : this.options();
  });

  protected optionId(i: number): string {
    return `${this.inputId()}-option-${i}`;
  }

  protected show(): void {
    if (this.open()) {
      return;
    }
    this.query.set('');
    this.active.set(Math.max(0, this.options().findIndex((o) => o.value === this.current())));
    this.open.set(true);
  }

  protected close(): void {
    this.open.set(false);
    this.query.set('');
  }

  protected search(text: string): void {
    this.query.set(text);
    this.open.set(true);
    this.active.set(this.filtered().length > 0 ? 0 : -1);
  }

  protected pick(option: SearchOption): void {
    this.close();
    if (option.value === this.current()) {
      return;
    }
    if (this.onChange) {
      this.current.set(option.value);
      this.onChange(option.value);
    }
    this.changed.emit(option.value);
  }

  protected onKey(event: KeyboardEvent): void {
    const count = this.filtered().length;
    switch (event.key) {
      case 'ArrowDown':
      case 'ArrowUp': {
        event.preventDefault();
        if (!this.open()) {
          this.show();
          return;
        }
        if (count > 0) {
          const step = event.key === 'ArrowDown' ? 1 : -1;
          this.active.update((i) => (i + step + count) % count);
          this.scrollToActive();
        }
        return;
      }
      case 'Enter': {
        if (this.open()) {
          event.preventDefault();
          const option = this.filtered()[this.active()];
          if (option) {
            this.pick(option);
          }
        }
        return;
      }
      case 'Escape': {
        if (this.open()) {
          event.preventDefault();
          event.stopPropagation();
          this.close();
          this.box().nativeElement.select();
        }
        return;
      }
    }
  }

  private scrollToActive(): void {
    queueMicrotask(() => document.getElementById(this.optionId(this.active()))?.scrollIntoView?.({ block: 'nearest' }));
  }
}

/** Lower case without accents, so "gia dinh" matches "Gia đình". */
function fold(text: string): string {
  return text.normalize('NFD').replace(/\p{M}/gu, '').replace(/đ/gi, 'd').toLowerCase();
}
