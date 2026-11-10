import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { ImageEngine } from '../../../core/models/word-bank';

/**
 * The choice of what draws a picture (added 2026-11-06): Gemini or the Codex CLI of the server, as
 * two cards with a radio mark. Codex is listed but cannot be chosen when the server does not run it.
 * The lesson pictures and the course page's scene batch share it.
 */
@Component({
  selector: 'lu-engine-picker',
  template: `
    <fieldset class="engines" [disabled]="disabled()">
      <legend class="engines-label">{{ label() }}</legend>
      <div class="engine-list">
        <label class="engine" [class.selected]="value() === 'gemini'">
          <input
            type="radio"
            [name]="name()"
            value="gemini"
            [checked]="value() === 'gemini'"
            (change)="changed.emit('gemini')"
          />
          <span class="engine-mark" aria-hidden="true"></span>
          <span class="engine-text">
            <span class="engine-name">Gemini</span>
            <span class="engine-desc">Dùng API key của máy chủ · tính phí theo từng ảnh</span>
          </span>
        </label>
        <label
          class="engine"
          [class.selected]="value() === 'codex'"
          [class.unavailable]="!codexAvailable()"
        >
          <input
            type="radio"
            [name]="name()"
            value="codex"
            [checked]="value() === 'codex'"
            [disabled]="!codexAvailable()"
            (change)="changed.emit('codex')"
          />
          <span class="engine-mark" aria-hidden="true"></span>
          <span class="engine-text">
            <span class="engine-name">Codex CLI <span class="engine-tag">ChatGPT</span></span>
            <span class="engine-desc">
              @if (codexAvailable()) {
                Dùng tài khoản ChatGPT trên máy chủ · không tốn API key · khoảng 1 phút mỗi ảnh
              } @else {
                Máy chủ chưa bật (CODEX_IMAGES)
              }
            </span>
          </span>
        </label>
      </div>
    </fieldset>
  `,
  styles: `
    .engines {
      display: grid;
      gap: var(--space-2);
      min-width: 0;
      margin: 0;
      padding: 0;
      border: 0;
    }

    .engines-label {
      padding: 0;
      margin-bottom: var(--space-2);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
    }

    .engine-list {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: var(--space-2);
    }

    .engine {
      position: relative;
      display: flex;
      align-items: flex-start;
      gap: var(--space-3);
      padding: var(--space-3);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
      cursor: pointer;
      transition:
        border-color 0.15s ease,
        box-shadow 0.15s ease;
    }

    @media (hover: hover) {
      .engine:hover:not(.unavailable) {
        border-color: var(--color-primary);
      }
    }

    .engine.selected {
      border-color: var(--color-primary);
      box-shadow: 0 0 0 1px var(--color-primary);
    }

    .engine.unavailable {
      cursor: not-allowed;
      opacity: 0.65;
    }

    /* The real radio stays in the page for keyboards and screen readers; the mark draws it. */
    .engine input {
      position: absolute;
      opacity: 0;
      pointer-events: none;
    }

    .engine:has(input:focus-visible) {
      outline: 2px solid var(--color-primary);
      outline-offset: 2px;
    }

    .engine-mark {
      flex: none;
      width: 18px;
      height: 18px;
      margin-top: 3px;
      border: 2px solid var(--color-text-muted);
      border-radius: 50%;
    }

    .engine.selected .engine-mark {
      border-color: var(--color-primary);
      background: radial-gradient(circle, var(--color-primary) 0 4px, transparent 5px);
    }

    .engine-text {
      display: grid;
      gap: 2px;
      min-width: 0;
    }

    .engine-name {
      font: var(--text-base) var(--font-ui);
      font-weight: 600;
    }

    .engine-tag {
      display: inline-block;
      margin-left: var(--space-1);
      padding: 0 var(--space-2);
      font: var(--text-xs) var(--font-ui);
      color: var(--color-accent-ink);
      vertical-align: 2px;
      background: var(--color-accent-soft);
      border-radius: var(--radius-full);
    }

    .engine-desc {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }

    @media (prefers-reduced-motion: reduce) {
      .engine {
        transition: none;
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EnginePicker {
  /** The engine chosen; "" when none is chosen yet. */
  readonly value = input.required<ImageEngine | ''>();
  readonly codexAvailable = input(false);
  readonly disabled = input(false);
  /** The name of the radio group; unique on the page. */
  readonly name = input('image-engine');
  readonly label = input('Tạo ảnh bằng');

  readonly changed = output<ImageEngine>();
}
