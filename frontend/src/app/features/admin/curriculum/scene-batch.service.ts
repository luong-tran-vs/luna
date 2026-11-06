import { inject, Injectable } from '@angular/core';

import { SceneBatch } from '../../../core/models/lesson';
import { AdminApiService } from '../admin-api.service';
import { BatchWatcher } from '../batch-watcher';

/**
 * The scene picture batch the server draws in the background (added 2026-11-06): of a topic of the
 * course page, or of one lesson from its page.
 */
@Injectable({ providedIn: 'root' })
export class SceneBatchService extends BatchWatcher<SceneBatch> {
  private readonly api = inject(AdminApiService);

  protected fetch() {
    return this.api.sceneBatch();
  }
}

/** The key of a topic of the course in a batch: its level and number, e.g. "A1-3". */
export function sceneBatchKey(level: string, topic: number): string {
  return `${level}-${topic}`;
}

/** The level a batch key names, '' for a malformed key. */
export function levelOfKey(key: string): string {
  const m = /^([A-C][12])-\d+$/.exec(key);
  return m ? m[1] : '';
}

/** The lesson a batch key names ("lesson:<id>", the lesson page's batch), '' otherwise. */
export function lessonOfKey(key: string): string {
  return key.startsWith('lesson:') ? key.slice('lesson:'.length) : '';
}
