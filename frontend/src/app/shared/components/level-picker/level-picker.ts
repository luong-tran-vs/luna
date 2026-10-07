import { ChangeDetectionStrategy, Component, computed, ElementRef, forwardRef, inject, input, output, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';

import { LEVEL_NAMES, LEVELS } from '../../../core/models/lesson';
import { Icon } from '../icon/icon';

/** One choice of the picker; `hint` is the small line under the code. */
export interface LevelOption {
  value: string;
  label: string;
  hint?: string;
}

/** A1 → C2 with their Vietnamese names. */
export const LEVEL_OPTIONS: readonly LevelOption[] = LEVELS.map((l) => ({ value: l, label: l, hint: LEVEL_NAMES[l] }));

/**
 * Picks a CEFR level (or another of the given options) with a row of cards instead of a select:
 * a radio group, so arrows move the choice and Tab enters on the chosen card. Works with
 * formControlName or with [value] and (changed).
 */
@Component({
  selector: 'lu-level-picker',
  imports: [Icon],
  template: `
    <div
      class="options"
      role="radiogroup"
      [attr.aria-labelledby]="labelledBy()"
      [attr.aria-describedby]="describedBy()"
      [attr.aria-invalid]="invalid() || null"
      [class.compact]="compact()"
      [class.invalid]="invalid()"
    >
      @for (o of options(); track o.value; let i = $index) {
        <button
          type="button"
          role="radio"
          class="option"
          [attr.aria-checked]="o.value === current()"
          [tabIndex]="i === focusIndex() ? 0 : -1"
          [disabled]="disabled()"
          (click)="choose(o.value)"
          (keydown)="onKeydown($event, i)"
        >
          <span class="code">{{ o.label }}</span>
          @if (o.hint && !compact()) {
            <span class="hint">{{ o.hint }}</span>
          }
          @if (o.value === current()) {
            <span class="check" aria-hidden="true"><lu-icon name="check" /></span>
          }
        </button>
      }
    </div>
  `,
  styles: `
    :host {
      display: block;
    }
    .options {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(5.5rem, 1fr));
      gap: var(--space-2);
    }
    .options.invalid .option {
      border-color: var(--color-bad);
    }
    .options.compact {
      grid-template-columns: repeat(auto-fill, minmax(3.5rem, 1fr));
    }
    .option {
      position: relative;
      display: grid;
      align-content: center;
      justify-items: start;
      gap: 2px;
      min-height: 44px;
      padding: var(--space-2) var(--space-3);
      color: var(--color-text);
      text-align: left;
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
      cursor: pointer;
      transition: border-color 0.15s ease, background-color 0.15s ease;
    }
    .compact .option {
      justify-items: center;
      padding: var(--space-1) var(--space-2);
    }
    .option:hover:not(:disabled) {
      border-color: var(--color-primary);
    }
    .option:disabled {
      cursor: not-allowed;
      opacity: 0.6;
    }
    .code {
      font: var(--text-sm) var(--font-ui);
      font-weight: 700;
    }
    .hint {
      font: var(--text-xs) var(--font-ui);
      color: var(--color-text-muted);
    }
    /* Chosen: thick border, tinted background and a check, not color alone. */
    .option[aria-checked='true'] {
      color: var(--color-primary-ink);
      background: var(--color-primary-soft);
      border: 2px solid var(--color-primary);
    }
    .check {
      position: absolute;
      top: 4px;
      right: 4px;
      display: inline-flex;
      padding: 1px;
      color: var(--color-on-primary);
      background: var(--color-primary);
      border-radius: 50%;
      --icon-size: 12px;
    }
    @media (prefers-reduced-motion: reduce) {
      .option {
        transition: none;
      }
    }
  `,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => LevelPicker), multi: true }],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LevelPicker implements ControlValueAccessor {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  /** The choices; A1 → C2 by default. */
  readonly options = input<readonly LevelOption[]>(LEVEL_OPTIONS);
  /** Id of the element naming the group (its visible label). */
  readonly labelledBy = input<string | null>(null);
  /** Ids of the hint and error under the group. */
  readonly describedBy = input<string | null>(null);
  readonly invalid = input(false);
  /** Codes only, on one tighter row (for filters). */
  readonly compact = input(false);
  /** The chosen value when not used in a form. */
  readonly value = input<string | null>(null);
  readonly changed = output<string>();

  private readonly formValue = signal<string | null>(null);
  private readonly inForm = signal(false);
  protected readonly disabled = signal(false);
  protected readonly current = computed(() => (this.inForm() ? this.formValue() : this.value()));
  /** The card Tab lands on: the chosen one, else the first. */
  protected readonly focusIndex = computed(() => Math.max(0, this.options().findIndex((o) => o.value === this.current())));

  private onChange: (value: string) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  protected choose(value: string): void {
    if (this.disabled()) {
      return;
    }
    if (this.inForm()) {
      this.formValue.set(value);
      this.onChange(value);
      this.onTouched();
    }
    this.changed.emit(value);
  }

  /** Arrows move the choice, wrapping around, and the focus follows (radio group pattern). */
  protected onKeydown(event: KeyboardEvent, index: number): void {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const list = this.options();
    const next = (index + step + list.length) % list.length;
    this.choose(list[next].value);
    this.host.nativeElement.querySelectorAll<HTMLElement>('[role="radio"]')[next]?.focus();
  }

  writeValue(value: string | null): void {
    this.inForm.set(true);
    this.formValue.set(value);
  }

  registerOnChange(fn: (value: string) => void): void {
    this.inForm.set(true);
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled.set(disabled);
  }
}
