import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { Observable } from 'rxjs';

import { AttemptBody, AttemptResult, GrammarDetail, GrammarList, ReportBody } from '../models/grammar-study';
import { Level } from '../models/lesson';

/** The learner's grammar section (F20): list, one lesson and the scores of its exercises. */
@Injectable({ providedIn: 'root' })
export class GrammarApiService {
  private readonly http = inject(HttpClient);

  points(level?: Level): Observable<GrammarList> {
    return this.http.get<GrammarList>('/grammar', { params: level ? { level } : {} });
  }

  get(pointId: string): Observable<GrammarDetail> {
    return this.http.get<GrammarDetail>(`/grammar/${encodeURIComponent(pointId)}`);
  }

  recordAttempt(pointId: string, body: AttemptBody): Observable<AttemptResult> {
    return this.http.post<AttemptResult>(`/grammar/${encodeURIComponent(pointId)}/attempts`, body);
  }

  /** F21: tells the admins an exercise looks wrong; reporting it again reopens the report. */
  reportExercise(pointId: string, body: ReportBody): Observable<{ ok: boolean }> {
    return this.http.post<{ ok: boolean }>(`/grammar/${encodeURIComponent(pointId)}/reports`, body);
  }

  /** Exercises reported in this session ("pointId/exerciseId"), so the button shows them as reported. */
  readonly reported = signal<ReadonlySet<string>>(new Set());

  markReported(pointId: string, exerciseId: string): void {
    this.reported.update((s) => new Set(s).add(`${pointId}/${exerciseId}`));
  }
}
