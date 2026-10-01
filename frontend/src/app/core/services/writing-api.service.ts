import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { LessonWriting, UnseenCount, Writing, WritingSummary } from '../models/writing';

/** The writing API (F8, specs/014-writing-step/contracts/writing-api.md). */
@Injectable({ providedIn: 'root' })
export class WritingApiService {
  private readonly http = inject(HttpClient);

  lessonWriting(lessonId: string): Observable<LessonWriting> {
    return this.http.get<LessonWriting>(`/api/lessons/${lessonId}/writing`);
  }

  saveDraft(lessonId: string, text: string): Observable<Writing> {
    return this.unwrap(this.http.put<{ writing: Writing }>(`/api/lessons/${lessonId}/writing`, { text }));
  }

  submit(lessonId: string, text: string): Observable<Writing> {
    return this.unwrap(this.http.post<{ writing: Writing }>(`/api/lessons/${lessonId}/writing/submit`, { text }));
  }

  list(): Observable<WritingSummary[]> {
    return this.http.get<{ writings: WritingSummary[] }>('/api/writings').pipe(map((r) => r.writings));
  }

  get(id: string): Observable<Writing> {
    return this.unwrap(this.http.get<{ writing: Writing }>(`/api/writings/${id}`));
  }

  seen(id: string): Observable<void> {
    return this.http.post<void>(`/api/writings/${id}/seen`, null);
  }

  regrade(id: string): Observable<Writing> {
    return this.unwrap(this.http.post<{ writing: Writing }>(`/api/writings/${id}/regrade`, null));
  }

  unseenCount(): Observable<UnseenCount> {
    return this.http.get<UnseenCount>('/api/writings/unseen-count');
  }

  private unwrap(obs: Observable<{ writing: Writing }>): Observable<Writing> {
    return obs.pipe(map((r) => r.writing));
  }
}
