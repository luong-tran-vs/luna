import { inject, Injectable } from '@angular/core';

import { TopicImageBatch } from '../../../core/models/topic-images';
import { BatchWatcher } from '../batch-watcher';
import { TopicImagesApiService } from './topic-images-api.service';

/** The batch drawing the topic pictures in the background (added 2026-11-16). */
@Injectable({ providedIn: 'root' })
export class TopicImageBatchService extends BatchWatcher<TopicImageBatch> {
  protected readonly hideKey = 'luna.admin.hiddenBatch.topicImages';
  private readonly api = inject(TopicImagesApiService);

  protected fetch() {
    return this.api.batch();
  }
}
