import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { GenerateInput, GenerateResult } from '../../core/models/generate';
import {
  AnnotationInput,
  ExtrasInput,
  JobKind,
  Lesson,
  LessonFilter,
  LessonInput,
  LessonSummary,
} from '../../core/models/lesson';
import { Topic, TopicInput, TopicRoadmap } from '../../core/models/topic';

const BASE = '/api/admin';

@Injectable({ providedIn: 'root' })
export class AdminApiService {
  private readonly http = inject(HttpClient);

  list(filter: LessonFilter = {}): Observable<LessonSummary[]> {
    let params = new HttpParams();
    if (filter.level) {
      params = params.set('level', filter.level);
    }
    if (filter.topicId) {
      params = params.set('topicId', filter.topicId);
    }
    return this.http
      .get<{ lessons: LessonSummary[] }>(`${BASE}/lessons`, { params })
      .pipe(map((r) => r.lessons));
  }

  get(id: string): Observable<Lesson> {
    return this.unwrap(this.http.get<{ lesson: Lesson }>(`${BASE}/lessons/${id}`));
  }

  create(input: LessonInput): Observable<Lesson> {
    return this.unwrap(this.http.post<{ lesson: Lesson }>(`${BASE}/lessons`, input));
  }

  update(id: string, input: LessonInput): Observable<Lesson> {
    return this.unwrap(this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}`, input));
  }

  remove(id: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/lessons/${id}`);
  }

  retry(id: string, job: JobKind): Observable<Lesson> {
    return this.unwrap(
      this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/retry`, null, { params: { job } }),
    );
  }

  /** Saves the questions, grammar note and writing prompt (F15). */
  updateExtras(id: string, input: ExtrasInput): Observable<Lesson> {
    return this.unwrap(this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}/extras`, input));
  }

  saveAnnotations(id: string, annotations: AnnotationInput[]): Observable<Lesson> {
    return this.unwrap(
      this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}/annotations`, { annotations }),
    );
  }

  topics(level?: string): Observable<Topic[]> {
    const params = level ? { level } : undefined;
    return this.http.get<{ topics: Topic[] }>(`${BASE}/topics`, { params }).pipe(map((r) => r.topics));
  }

  createTopic(input: TopicInput): Observable<Topic> {
    return this.http.post<{ topic: Topic }>(`${BASE}/topics`, input).pipe(map((r) => r.topic));
  }

  updateTopic(id: string, input: TopicInput): Observable<Topic> {
    return this.http.put<{ topic: Topic }>(`${BASE}/topics/${id}`, input).pipe(map((r) => r.topic));
  }

  deleteTopic(id: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/topics/${id}`);
  }

  topicRoadmap(topicId: string): Observable<TopicRoadmap> {
    return this.http.get<TopicRoadmap>(`${BASE}/topics/${topicId}/roadmap`);
  }

  setTopicRoadmap(topicId: string, lessonIds: string[]): Observable<TopicRoadmap> {
    return this.http.put<TopicRoadmap>(`${BASE}/topics/${topicId}/roadmap`, { lessonIds });
  }

  generateLessons(topicId: string, input: GenerateInput): Observable<GenerateResult> {
    return this.http.post<GenerateResult>(`${BASE}/topics/${topicId}/generate`, input);
  }

  private unwrap(obs: Observable<{ lesson: Lesson }>): Observable<Lesson> {
    return obs.pipe(map((r) => r.lesson));
  }
}
