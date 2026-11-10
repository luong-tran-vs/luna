import { inject, Injectable } from '@angular/core';

import { VerbImageBatch } from '../../../core/models/verb-images';
import { BatchWatcher } from '../batch-watcher';
import { VerbImagesApiService } from './verb-images-api.service';

/** The batch drawing the verb pictures in the background (added 2026-11-10). */
@Injectable({ providedIn: 'root' })
export class VerbImageBatchService extends BatchWatcher<VerbImageBatch> {
  private readonly api = inject(VerbImagesApiService);

  protected fetch() {
    return this.api.batch();
  }
}
