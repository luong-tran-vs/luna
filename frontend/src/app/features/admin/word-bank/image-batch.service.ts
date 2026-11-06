import { computed, inject, Injectable, InjectionToken, signal } from '@angular/core';

import { BankImageBatch } from '../../../core/models/word-bank';
import { AdminApiService } from '../admin-api.service';

/** How often, in ms, the admin pages ask how a running picture batch is going (tests shorten it). */
export const BANK_BATCH_POLL_MS = new InjectionToken<number>('BANK_BATCH_POLL_MS', { factory: () => 3000 });

/** How often, in ms, they look for a batch started elsewhere (another tab) while none runs. */
export const BANK_BATCH_IDLE_POLL_MS = new InjectionToken<number>('BANK_BATCH_IDLE_POLL_MS', {
  factory: () => 30000,
});

/**
 * The picture batch the server draws in the background (added 2026-11-04), shared by the admin
 * layout, which labels it on every page, and the word bank, which runs it. While a page watches,
 * the status is asked for every few seconds during a batch and now and then otherwise.
 */
@Injectable({ providedIn: 'root' })
export class ImageBatchService {
  private readonly api = inject(AdminApiService);
  private readonly pollMs = inject(BANK_BATCH_POLL_MS);
  private readonly idleMs = inject(BANK_BATCH_IDLE_POLL_MS);

  /** The running, or last, batch; null until known. */
  readonly status = signal<BankImageBatch | null>(null);
  readonly running = computed(() => this.status()?.running ?? false);
  /** The admin hid the result of the batch started then. */
  private readonly hiddenAt = signal<string | null>(null);
  /** A finished batch whose result is still to be shown. */
  readonly result = computed(() => {
    const b = this.status();
    return b && !b.running && b.total > 0 && b.startedAt !== this.hiddenAt() ? b : null;
  });

  private watchers = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  /** Starts following the batch; call the returned function to stop. */
  watch(): () => void {
    this.watchers++;
    if (this.watchers === 1) {
      this.refresh();
    }
    let stopped = false;
    return () => {
      if (stopped) {
        return;
      }
      stopped = true;
      this.watchers--;
      if (this.watchers === 0) {
        clearTimeout(this.timer);
      }
    };
  }

  /** Takes a status the server just answered (after starting or stopping a batch). */
  set(b: BankImageBatch): void {
    this.status.set(b);
    this.schedule();
  }

  hideResult(): void {
    this.hiddenAt.set(this.status()?.startedAt ?? null);
  }

  private refresh(): void {
    this.api.bankImageBatch().subscribe({
      next: (b) => this.set(b),
      // Not knowing the batch only hides its label; ask again later.
      error: () => this.schedule(),
    });
  }

  private schedule(): void {
    clearTimeout(this.timer);
    if (this.watchers > 0) {
      this.timer = setTimeout(() => this.refresh(), this.running() ? this.pollMs : this.idleMs);
    }
  }
}

/** How many topic names the scope lists before counting the rest. */
const SCOPE_TOPICS = 3;

/**
 * What a batch draws, after "cho": the filter's topic, every word lacking a picture or the ticked
 * words, with the topics those words belong to, e.g. `các từ thiếu ảnh của chủ đề "Nhà cửa"` or
 * `5 từ đã chọn (chủ đề: Chào hỏi, Nhà cửa)`.
 */
export function batchScope(b: BankImageBatch): string {
  const topic = b.topicName ? ` của chủ đề "${b.topicName}"` : '';
  let scope = b.ticked ? `${b.total} từ đã chọn${topic}` : topic ? `các từ thiếu ảnh${topic}` : 'mọi từ thiếu ảnh';
  if (!b.ticked && b.search) {
    scope += ` khớp "${b.search}"`;
  }
  const topics = b.topics ?? [];
  if (!b.topicName && topics.length) {
    const named = topics.slice(0, SCOPE_TOPICS).join(', ');
    scope +=
      topics.length <= SCOPE_TOPICS
        ? ` (chủ đề: ${named})`
        : ` (${topics.length} chủ đề: ${named} và ${topics.length - SCOPE_TOPICS} chủ đề khác)`;
  }
  return scope;
}
