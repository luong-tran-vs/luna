import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  TopicImageBatch,
  TopicImageBatchRequest,
  TopicImageList,
} from '../../../core/models/topic-images';

const BASE = '/admin';

/** The admin API of the topic pictures (added 2026-11-16). */
@Injectable({ providedIn: 'root' })
export class TopicImagesApiService {
  private readonly http = inject(HttpClient);

  /** The topics that have a picture. */
  list(): Observable<TopicImageList> {
    return this.http.get<TopicImageList>(`${BASE}/topic-images`);
  }

  remove(topicId: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/topic-images/${encodeURIComponent(topicId)}`);
  }

  batch(): Observable<TopicImageBatch> {
    return this.http.get<TopicImageBatch>(`${BASE}/topic-image-batch`);
  }

  startBatch(req: TopicImageBatchRequest): Observable<TopicImageBatch> {
    return this.http.post<TopicImageBatch>(`${BASE}/topic-image-batch`, req);
  }

  stopBatch(): Observable<TopicImageBatch> {
    return this.http.post<TopicImageBatch>(`${BASE}/topic-image-batch/stop`, {});
  }
}
