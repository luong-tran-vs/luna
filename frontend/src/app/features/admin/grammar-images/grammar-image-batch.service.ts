import { inject, Injectable } from '@angular/core';

import { GrammarImageBatch } from '../../../core/models/grammar-admin';
import { AdminApiService } from '../admin-api.service';
import { BatchWatcher } from '../batch-watcher';

/**
 * The batch drawing the pictures of grammar examples in the background (added 2026-11-06): of one
 * lesson from its page, or of every lesson of a level from the grammar list.
 */
@Injectable({ providedIn: 'root' })
export class GrammarImageBatchService extends BatchWatcher<GrammarImageBatch> {
  private readonly api = inject(AdminApiService);

  protected fetch() {
    return this.api.grammarImageBatch();
  }
}

/** The batch key of one grammar lesson. */
export function pointBatchKey(pointId: string): string {
  return `point:${pointId}`;
}

/** The batch key of every lesson of a level. */
export function levelBatchKey(level: string): string {
  return `level:${level}`;
}
