import { inject, Injectable } from '@angular/core';

import { GrammarPictureBatch } from '../../../core/models/grammar-admin';
import { AdminApiService } from '../admin-api.service';
import { BatchWatcher } from '../batch-watcher';

/**
 * The batch adding "Nhìn ảnh chọn từ" exercises to every lesson of a level in the background (added
 * 2026-11-06), started from the grammar list.
 */
@Injectable({ providedIn: 'root' })
export class GrammarPictureBatchService extends BatchWatcher<GrammarPictureBatch> {
  private readonly api = inject(AdminApiService);

  protected fetch() {
    return this.api.grammarPictureBatch();
  }
}
