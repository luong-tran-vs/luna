import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { countWords, wordRange } from '../../../core/models/generate';
import { ImageSettingsInput, Level } from '../../../core/models/lesson';

/** A generated lesson being reviewed on the roadmap page (F7). Never stored until saved. */
export interface DraftState {
  key: number;
  /** Level the draft was written for; it is saved to the roadmap of that level. */
  level: Level;
  title: string;
  content: string;
  /** F18: target words asked for this draft, and those not found in it (from generation). */
  targetWords: string[];
  missingWords: string[];
  /** Grammar point the draft was written for; saved with the lesson. */
  grammarPointId?: string;
  /** The length asked for (words); a draft outside its ±20% range gets a warning. */
  targetLength: number;
  /** F23: pictures for the vocabulary words of the saved lesson; null for none. */
  images?: ImageSettingsInput | null;
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

  /** "Dùng 3/4 từ mục tiêu" (F18). */
  protected usedTargets(d: DraftState): string {
    return `Dùng ${d.targetWords.length - d.missingWords.length}/${d.targetWords.length} từ mục tiêu`;
  }

  /** Warning when the content is off the asked length, e.g. "ngắn hơn yêu cầu (96–144 từ)"; null otherwise. */
  protected lengthWarning(d: DraftState): string | null {
    if (!d.targetLength) {
      return null;
    }
    const n = countWords(d.content);
    const { min, max } = wordRange(d.targetLength);
    if (n < min) {
      return `ngắn hơn yêu cầu (${min}–${max} từ)`;
    }
    return n > max ? `dài hơn yêu cầu (${min}–${max} từ)` : null;
  }

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
