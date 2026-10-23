import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import {
  Goals,
  LearningPath,
  LessonStudy,
  MyLessons,
  PublicTopic,
  SetGoalResult,
  Step,
  TopicSummary,
} from '../models/study';

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

  /** The goal's lessons; with allTopics the lessons completed in every topic (the review tab). */
  myLessons(allTopics = false): Observable<MyLessons> {
    return this.http.get<MyLessons>('/lessons/mine', allTopics ? { params: { all: '1' } } : {});
  }

  /** The goal's topic split into one milestone per level, each lesson a step. */
  path(): Observable<LearningPath> {
    return this.http.get<LearningPath>('/lessons/path');
  }

  /** What the learner did in the topic's roadmap at level, with a rating. */
  topicSummary(topicId: string, level: string): Observable<TopicSummary> {
    return this.http.get<TopicSummary>(`/topics/${topicId}/summary`, { params: { level } });
  }
}
