import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  VerbImage,
  VerbImageBatch,
  VerbImageBatchRequest,
  VerbImageList,
} from '../../../core/models/verb-images';

const BASE = '/admin';

/** The admin API of the verb pictures (added 2026-11-10). */
@Injectable({ providedIn: 'root' })
export class VerbImagesApiService {
  private readonly http = inject(HttpClient);

  /** The verbs that have a picture, and which of bases have one in the word bank. */
  list(bases: readonly string[]): Observable<VerbImageList> {
    return this.http.get<VerbImageList>(`${BASE}/verb-images`, {
      params: { bases: bases.join(',') },
    });
  }

  upload(base: string, file: Blob): Observable<VerbImage> {
    return this.http.put<VerbImage>(`${BASE}/verb-images/${encodeURIComponent(base)}`, file, {
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
    });
  }

  /** The word bank's picture of the same word becomes the verb's. */
  fromBank(base: string): Observable<VerbImage> {
    return this.http.post<VerbImage>(
      `${BASE}/verb-images/${encodeURIComponent(base)}/from-bank`,
      {},
    );
  }

  remove(base: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/verb-images/${encodeURIComponent(base)}`);
  }

  batch(): Observable<VerbImageBatch> {
    return this.http.get<VerbImageBatch>(`${BASE}/verb-image-batch`);
  }

  startBatch(req: VerbImageBatchRequest): Observable<VerbImageBatch> {
    return this.http.post<VerbImageBatch>(`${BASE}/verb-image-batch`, req);
  }

  stopBatch(): Observable<VerbImageBatch> {
    return this.http.post<VerbImageBatch>(`${BASE}/verb-image-batch/stop`, {});
  }
}
