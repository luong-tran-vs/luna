import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { LookupResult, ReadingLesson } from '../../core/models/reading';
import { LessonVocabulary } from '../../core/models/vocab';

@Injectable({ providedIn: 'root' })
export class ReadingApiService {
  private readonly http = inject(HttpClient);

  getLesson(id: string): Observable<ReadingLesson> {
    return this.http.get<{ lesson: ReadingLesson }>(`/api/lessons/${id}`).pipe(map((r) => r.lesson));
  }

  lookup(id: string, q: string, sentence: number): Observable<LookupResult> {
    return this.http.get<LookupResult>(`/api/lessons/${id}/lookup`, {
      params: { q, sentence: String(sentence) },
    });
  }

  vocabulary(id: string): Observable<LessonVocabulary> {
    return this.http.get<LessonVocabulary>(`/api/lessons/${id}/vocabulary`);
  }
}
