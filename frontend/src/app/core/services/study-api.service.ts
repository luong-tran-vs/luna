import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { Goals, LessonStudy, MyLessons, PublicTopic, SetGoalResult, Step } from '../models/study';

/** The study flow (L): goals, the learner's lessons and the steps of the lesson being studied. */
@Injectable({ providedIn: 'root' })
export class StudyApiService {
  private readonly http = inject(HttpClient);

  topics(level: string): Observable<PublicTopic[]> {
    return this.http.get<{ topics: PublicTopic[] }>('/topics', { params: { level } }).pipe(map((r) => r.topics));
  }

  goals(): Observable<Goals> {
    return this.http.get<Goals>('/goals');
  }

  /** A goal is the roadmap of a topic at one level. */
  setGoal(topicId: string, level: string): Observable<SetGoalResult> {
    return this.http.post<SetGoalResult>('/goals', { topicId, level });
  }

  /** A lesson's steps for the learner; it only reads. */
  lessonStudy(lessonId: string): Observable<LessonStudy> {
    return this.http.get<LessonStudy>(`/lessons/${lessonId}/study`);
  }

  completeStep(lessonId: string, step: Step): Observable<LessonStudy> {
    return this.http.post<LessonStudy>(`/lessons/${lessonId}/steps/${step}/complete`, null);
  }

  /** Finishes the lesson without writing: the Write step is optional. */
  skipWrite(lessonId: string): Observable<LessonStudy> {
    return this.http.post<LessonStudy>(`/lessons/${lessonId}/steps/write/skip`, null);
  }

  savePosition(lessonId: string, step: Step, sentenceIndex: number): Observable<void> {
    return this.http.put<void>(`/lessons/${lessonId}/position`, { step, sentenceIndex });
  }

  myLessons(): Observable<MyLessons> {
    return this.http.get<MyLessons>('/lessons/mine');
  }
}
