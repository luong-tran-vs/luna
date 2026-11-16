import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, map, Observable, of, shareReplay } from 'rxjs';

import { serverUrl } from '../api-url';
import { TopicImageList } from '../models/topic-images';

/**
 * The topic pictures for the learner's pages (added 2026-11-16): a map from topic id to picture
 * URL, read once per page load. The pictures only decorate: on an error the map is empty.
 */
@Injectable({ providedIn: 'root' })
export class TopicImagesService {
  private readonly http = inject(HttpClient);
  private cache: Observable<ReadonlyMap<string, string>> | null = null;

  /** The picture URL (ready for an img) of each topic that has one. */
  all(): Observable<ReadonlyMap<string, string>> {
    this.cache ??= this.http.get<TopicImageList>('/topic-images').pipe(
      map((r) => new Map(r.images.map((i) => [i.topicId, serverUrl(i.imageUrl)] as const))),
      catchError(() => {
        this.cache = null;
        return of(new Map<string, string>());
      }),
      shareReplay(1),
    );
    return this.cache;
  }
}
