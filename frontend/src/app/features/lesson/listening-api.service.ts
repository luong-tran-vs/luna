import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { DictationInput, DictationSummary } from '../../core/models/dictation';

@Injectable({ providedIn: 'root' })
export class ListeningApiService {
  private readonly http = inject(HttpClient);

  summary(lessonId: string): Observable<DictationSummary> {
    return this.http
      .get<{ summary: DictationSummary }>(`/api/lessons/${lessonId}/dictation/summary`)
      .pipe(map((r) => r.summary));
  }

  /** Forgets this learner's dictation results of the lesson, to do the Listening step again. */
  reset(lessonId: string): Observable<void> {
    return this.http.delete<void>(`/api/lessons/${lessonId}/dictation`);
  }

  record(lessonId: string, input: DictationInput): Observable<DictationSummary> {
    return this.http
      .post<{ summary: DictationSummary }>(`/api/lessons/${lessonId}/dictation`, input)
      .pipe(map((r) => r.summary));
  }
}
