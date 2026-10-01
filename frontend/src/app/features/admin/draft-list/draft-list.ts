import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { countWords } from '../../../core/models/generate';

/** A generated lesson being reviewed on the roadmap page (F7). Never stored until saved. */
export interface DraftState {
  key: number;
  title: string;
  content: string;
  saving: boolean;
  /** Save error not tied to a field (network, server). */
  error: string | null;
  /** Field errors from the server: title, content. */
  fields: Record<string, string>;
}

export interface DraftChange {
  key: number;
  title?: string;
  content?: string;
}

/** Editable list of AI drafts with Lưu, Bỏ and Lưu tất cả. The parent owns the state. */
@Component({
  selector: 'lu-draft-list',
  templateUrl: './draft-list.html',
  styleUrl: './draft-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DraftList {
  readonly drafts = input.required<DraftState[]>();
  /** True while any save is running (Lưu tất cả included). */
  readonly busy = input(false);

  readonly changed = output<DraftChange>();
  readonly save = output<number>();
  readonly discard = output<number>();
  readonly saveAll = output<void>();

  protected readonly countWords = countWords;

  protected rows(content: string): number {
    return Math.min(20, Math.max(6, Math.ceil(content.length / 60) + content.split('\n').length));
  }

  protected describedBy(d: DraftState, field: string, hint?: string): string | null {
    const ids = [hint, d.fields[field] ? `draft-${d.key}-${field}-error` : undefined].filter(Boolean);
    return ids.length ? ids.join(' ') : null;
  }

  protected onTitle(key: number, event: Event): void {
    this.changed.emit({ key, title: (event.target as HTMLInputElement).value });
  }

  protected onContent(key: number, event: Event): void {
    this.changed.emit({ key, content: (event.target as HTMLTextAreaElement).value });
  }
}
