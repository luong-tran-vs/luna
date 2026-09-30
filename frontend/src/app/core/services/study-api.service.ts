import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { Goals, MyLessons, PublicTopic, SetGoalResult, Step, Today } from '../models/study';

/** The daily flow (L): goals, today's lesson and its steps, the learner's lessons. */
@Injectable({ providedIn: 'root' })
export class StudyApiService {
  private readonly http = inject(HttpClient);

  topics(level: string): Observable<PublicTopic[]> {
    return this.http.get<{ topics: PublicTopic[] }>('/api/topics', { params: { level } }).pipe(map((r) => r.topics));
  }

  goals(): Observable<Goals> {
    return this.http.get<Goals>('/api/goals');
  }

  setGoal(topicId: string): Observable<SetGoalResult> {
    return this.http.post<SetGoalResult>('/api/goals', { topicId });
  }

  today(): Observable<Today> {
    return this.http.get<Today>('/api/today');
  }

  completeStep(step: Step): Observable<Today> {
    return this.http.post<Today>(`/api/today/steps/${step}/complete`, null);
  }

  savePosition(step: Step, sentenceIndex: number): Observable<void> {
    return this.http.put<void>('/api/today/position', { step, sentenceIndex });
  }

  myLessons(): Observable<MyLessons> {
    return this.http.get<MyLessons>('/api/lessons/mine');
  }
}
