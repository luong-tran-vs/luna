import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';

export type ToeicTab = 'tests' | 'practice' | 'tags' | 'results' | 'mistakes';

/**
 * The tabs of the TOEIC area (FR-002): tests, practice by part, by question type, my results and
 * the mistakes notebook with its count. The sections of later batches show "sắp có".
 */
@Component({
  selector: 'lu-toeic-tabs',
  imports: [RouterLink],
  template: `
    <nav class="tabs" aria-label="Luyện thi TOEIC">
      <a class="tab" routerLink="/toeic" [class.on]="active() === 'tests'" [attr.aria-current]="active() === 'tests' ? 'page' : null">Đề thi</a>
      <a class="tab" routerLink="/toeic/practice" [class.on]="active() === 'practice'" [attr.aria-current]="active() === 'practice' ? 'page' : null">Luyện theo Part</a>
      <span class="tab soon" aria-disabled="true">Luyện theo dạng câu <small>sắp có</small></span>
      <span class="tab soon" aria-disabled="true">Kết quả của tôi <small>sắp có</small></span>
      <a class="tab" routerLink="/toeic/mistakes" [class.on]="active() === 'mistakes'" [attr.aria-current]="active() === 'mistakes' ? 'page' : null">
        Sổ câu sai
        @if (mistakes() !== null) {
          <span class="count">· {{ mistakes() }}</span>
        } @else if (guest()) {
          <span class="count" aria-label="chỉ dành cho thành viên">🔒</span>
        }
      </a>
    </nav>
  `,
  styles: `
    .tabs {
      display: flex;
      gap: var(--space-2);
      overflow-x: auto;
      padding-bottom: var(--space-1);
    }
    .tab {
      display: inline-flex;
      flex: none;
      gap: var(--space-1);
      align-items: center;
      min-height: 40px;
      padding: 0 var(--space-3);
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
      color: var(--color-text);
      text-decoration: none;
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-full);
    }
    .tab.on {
      color: var(--color-on-primary);
      background: var(--color-primary);
      border-color: var(--color-primary);
    }
    .tab.soon {
      color: var(--color-text-muted);
      border-style: dashed;
    }
    .tab small {
      font: var(--text-xs) var(--font-ui);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToeicTabs {
  readonly active = input.required<ToeicTab>();
  readonly mistakes = input<number | null>(null);
  readonly guest = input(false);
}
