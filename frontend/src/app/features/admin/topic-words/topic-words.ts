import { ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal, viewChild } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Topic, topicLabel, TopicWord } from '../../../core/models/topic';
import { AdminApiService } from '../admin-api.service';
import { Loading } from '../../../shared/components/loading/loading';

/** One row of the list being edited: a saved word or a new one, until Lưu. */
interface WordRow {
  key: number;
  text: string;
  /** Null for a word added on this page. */
  saved: TopicWord | null;
  /** Saved word marked for removal (kept on screen so it can be restored). */
  removed: boolean;
}

/** Splits pasted text into words: one per line or comma-separated; trims and collapses spaces. */
export function parseWords(text: string): string[] {
  return text
    .split(/[\n,]+/)
    .map((w) => w.trim().replace(/\s+/g, ' '))
    .filter((w) => w.length > 0);
}

/**
 * Vocabulary list of one topic (F18): coverage of each word in the topic's lessons, remove
 * words, add many at once, and save the whole list. Server errors `words.{i}` point into the
 * array sent, so they are shown under the matching row.
 */
@Component({
  selector: 'lu-topic-words',
  imports: [Loading, RouterLink],
  templateUrl: './topic-words.html',
  styleUrl: './topic-words.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TopicWords {
  private readonly api = inject(AdminApiService);
  private readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';
  private readonly addBox = viewChild<ElementRef<HTMLTextAreaElement>>('addBox');

  protected readonly topic = signal<Topic | null>(null);
  protected readonly title = computed(() => {
    const t = this.topic();
    return t ? `Từ vựng · ${topicLabel(t)}` : 'Từ vựng';
  });
  protected readonly notFound = signal(false);
  protected readonly loadError = signal<string | null>(null);

  /** The list as last loaded or saved. */
  private readonly loaded = signal<TopicWord[] | null>(null);
  protected readonly rows = signal<WordRow[]>([]);
  private nextKey = 1;

  protected readonly draft = signal('');
  protected readonly saving = signal(false);
  protected readonly saved = signal(false);
  protected readonly note = signal<string | null>(null);
  protected readonly topError = signal<string | null>(null);
  protected readonly rowErrors = signal<Record<number, string>>({});

  protected readonly summary = computed(() => {
    const list = this.loaded() ?? [];
    return { used: list.filter((w) => w.used).length, total: list.length };
  });
  protected readonly pendingCount = computed(() => {
    const rows = this.rows();
    return { added: rows.filter((r) => !r.saved).length, removed: rows.filter((r) => r.removed).length };
  });

  constructor() {
    void this.load();
  }

  private async load(): Promise<void> {
    try {
      const [topics, words] = await Promise.all([
        firstValueFrom(this.api.topics()),
        firstValueFrom(this.api.topicWords(this.id)),
      ]);
      const topic = topics.find((t) => t.id === this.id);
      if (!topic) {
        this.notFound.set(true);
        return;
      }
      this.topic.set(topic);
      this.reset(words);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        this.notFound.set(true);
      } else {
        this.loadError.set('Không tải được danh sách từ.');
      }
    }
  }

  private reset(words: TopicWord[]): void {
    this.loaded.set(words);
    this.rows.set(words.map((w) => ({ key: this.nextKey++, text: w.text, saved: w, removed: false })));
    this.rowErrors.set({});
    this.topError.set(null);
  }

  protected status(row: WordRow): string {
    if (!row.saved) {
      return 'Mới';
    }
    if (row.removed) {
      return 'Sẽ xoá';
    }
    return row.saved.used ? `Đã dùng · ${row.saved.lessonCount} bài` : 'Chưa dùng';
  }

  protected onDraft(event: Event): void {
    this.draft.set((event.target as HTMLTextAreaElement).value);
    this.saved.set(false);
  }

  /** Ctrl/Cmd+Enter in the box adds the words, like the button. */
  protected onDraftKey(event: KeyboardEvent): void {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      this.addDraft();
    }
  }

  /** Moves the words typed in the box into the list as new rows; skips words already there. */
  protected addDraft(): void {
    const words = parseWords(this.draft());
    this.draft.set('');
    this.saved.set(false);
    if (words.length === 0) {
      this.note.set(null);
      return;
    }
    const seen = new Set(this.rows().filter((r) => !r.removed).map((r) => r.text.toLowerCase()));
    const added: WordRow[] = [];
    const skipped: string[] = [];
    for (const text of words) {
      const lower = text.toLowerCase();
      if (seen.has(lower)) {
        skipped.push(text);
        continue;
      }
      seen.add(lower);
      added.push({ key: this.nextKey++, text, saved: null, removed: false });
    }
    this.rows.update((rows) => [...rows, ...added]);
    this.note.set(
      [
        added.length ? `Đã thêm ${added.length} từ mới, bấm Lưu để lưu.` : '',
        skipped.length ? `Bỏ qua ${skipped.length} từ đã có: ${skipped.join(', ')}.` : '',
      ]
        .filter(Boolean)
        .join(' '),
    );
  }

  /** ✕ on a saved word marks it removed (and back); on a new word drops it. */
  protected toggle(row: WordRow): void {
    this.saved.set(false);
    this.clearError(row.key);
    if (!row.saved) {
      this.rows.update((rows) => rows.filter((r) => r.key !== row.key));
      this.addBox()?.nativeElement.focus();
      return;
    }
    this.rows.update((rows) => rows.map((r) => (r.key === row.key ? { ...r, removed: !r.removed } : r)));
  }

  private clearError(key: number): void {
    if (this.rowErrors()[key]) {
      this.rowErrors.update((errors) => Object.fromEntries(Object.entries(errors).filter(([k]) => Number(k) !== key)));
    }
  }

  protected cancel(): void {
    this.draft.set('');
    this.note.set(null);
    this.saved.set(false);
    this.reset(this.loaded() ?? []);
  }

  /** Sends the kept words (in order) and the new ones; the textarea is added first. */
  protected async save(): Promise<void> {
    if (this.saving()) {
      return;
    }
    if (this.draft().trim()) {
      this.addDraft();
    }
    const sent = this.rows().filter((r) => !r.removed);
    this.saving.set(true);
    this.saved.set(false);
    this.topError.set(null);
    this.rowErrors.set({});
    try {
      const words = await firstValueFrom(this.api.setTopicWords(this.id, sent.map((r) => r.text)));
      this.reset(words);
      this.note.set(null);
      this.saved.set(true);
    } catch (err) {
      this.showErrors(err, sent);
    } finally {
      this.saving.set(false);
    }
  }

  private showErrors(err: unknown, sent: WordRow[]): void {
    const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
    if (err instanceof ApiError && err.status === 404) {
      this.notFound.set(true);
      return;
    }
    const fields = err instanceof ApiError && err.status === 400 ? (body?.fields ?? {}) : {};
    const rowErrors: Record<number, string> = {};
    const other: string[] = [];
    for (const [field, message] of Object.entries(fields)) {
      const match = /^words\.(\d+)$/.exec(field);
      const row = match ? sent[Number(match[1])] : undefined;
      if (row) {
        rowErrors[row.key] = message;
      } else {
        other.push(message);
      }
    }
    this.rowErrors.set(rowErrors);
    const count = Object.keys(rowErrors).length;
    if (other.length) {
      this.topError.set(other.join(' '));
    } else if (count) {
      this.topError.set(`Chưa lưu: ${count} từ cần sửa (xem bên dưới).`);
    } else {
      this.topError.set(body?.message ?? 'Không lưu được, vui lòng thử lại.');
    }
  }
}
