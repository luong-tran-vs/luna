import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { GrammarPoint } from '../../core/models/grammar';
import { GrammarContent, GrammarExerciseInput, GrammarLessonAdmin, GrammarLessonSummary, GrammarReportGroup } from '../../core/models/grammar-admin';
import { GenerateInput, GenerateResult } from '../../core/models/generate';
import {
  AnnotationInput,
  ExtrasInput,
  TranslationInput,
  FlagArea,
  ImageSettingsInput,
  JobKind,
  Lesson,
  LessonFilter,
  LessonImages,
  LessonInput,
  LessonSummary,
} from '../../core/models/lesson';
import { Level } from '../../core/models/lesson';
import { Topic, TopicInput, TopicRoadmap, TopicWord, TopicWordInput, WordPlan } from '../../core/models/topic';

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

  /** Replaces the practice's translation sentences (the full list); 409 no_practice / practice_running / practice_changed. */
  updateTranslations(id: string, translations: TranslationInput[]): Observable<Lesson> {
    return this.unwrap(this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}/practice/translations`, { translations }));
  }

  /** F23: the picture settings of a lesson and how many of its words have a picture. */
  lessonImages(id: string): Observable<LessonImages> {
    return this.http.get<LessonImages>(`${BASE}/lessons/${id}/images`);
  }

  /** F23: turns the word pictures on (drawing the words that have none) or off. */
  setLessonImages(id: string, input: ImageSettingsInput): Observable<LessonImages> {
    return this.http.put<LessonImages>(`${BASE}/lessons/${id}/images`, input);
  }

  /** F23: uploads the picture of one word (the file is the body); the server scales it down. */
  uploadWordImage(id: string, lemma: string, file: Blob): Observable<LessonImages> {
    return this.http.put<LessonImages>(`${BASE}/lessons/${id}/images/${encodeURIComponent(lemma)}`, file, {
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
    });
  }

  /** F23: the server downloads the picture behind a link and stores it for one word. */
  importWordImage(id: string, lemma: string, url: string): Observable<LessonImages> {
    return this.http.post<LessonImages>(`${BASE}/lessons/${id}/images/${encodeURIComponent(lemma)}/import`, { url });
  }

  /** F23: removes the picture of one word. */
  deleteWordImage(id: string, lemma: string): Observable<LessonImages> {
    return this.http.delete<LessonImages>(`${BASE}/lessons/${id}/images/${encodeURIComponent(lemma)}`);
  }

  /** Generates the practice again with one AI request (F17); 202 with the lesson. */
  regeneratePractice(id: string): Observable<Lesson> {
    return this.unwrap(
      this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/practice/regenerate`, null),
    );
  }

  /** F22: a second AI re-checks the generated parts and flags what looks wrong. */
  checkLesson(id: string): Observable<Lesson> {
    return this.unwrap(this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/check`, null));
  }

  /** F22: keeps a flagged spot as it is, after the admin looked at it. */
  confirmLessonFlag(id: string, area: FlagArea, index: number): Observable<Lesson> {
    return this.unwrap(this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/check/confirm`, { area, index }));
  }

  /** F22: marks the check as finished; the server refuses while flags are unconfirmed. */
  verifyLesson(id: string): Observable<Lesson> {
    return this.unwrap(this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/check/verify`, null));
  }

  saveAnnotations(id: string, annotations: AnnotationInput[]): Observable<Lesson> {
    return this.unwrap(
      this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}/annotations`, { annotations }),
    );
  }

  topics(): Observable<Topic[]> {
    return this.http.get<{ topics: Topic[] }>(`${BASE}/topics`).pipe(map((r) => r.topics));
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

  /** The roadmap of a topic at one level. */
  topicRoadmap(topicId: string, level: Level): Observable<TopicRoadmap> {
    return this.http.get<TopicRoadmap>(`${BASE}/topics/${topicId}/roadmap`, { params: { level } });
  }

  setTopicRoadmap(topicId: string, level: Level, lessonIds: string[]): Observable<TopicRoadmap> {
    return this.http.put<TopicRoadmap>(`${BASE}/topics/${topicId}/roadmap`, { lessonIds }, { params: { level } });
  }

  generateLessons(topicId: string, input: GenerateInput): Observable<GenerateResult> {
    return this.http.post<GenerateResult>(`${BASE}/topics/${topicId}/generate`, input);
  }

  /** F18: the topic's vocabulary list with coverage, in list order. */
  topicWords(topicId: string): Observable<TopicWord[]> {
    return this.http.get<{ words: TopicWord[] }>(`${BASE}/topics/${topicId}/words`).pipe(map((r) => r.words));
  }

  /** F18: replaces the whole vocabulary list; returns it normalized with coverage. */
  setTopicWords(topicId: string, words: TopicWordInput[]): Observable<TopicWord[]> {
    return this.http
      .put<{ words: TopicWord[] }>(`${BASE}/topics/${topicId}/words`, { words })
      .pipe(map((r) => r.words));
  }

  /**
   * F18: target words for lessons of `level` split into `count` groups of up to `perLesson` words,
   * least used first, and how many more unused words of that level the topic needs (`shortage`).
   */
  wordPlan(topicId: string, level: Level, count: number, perLesson: number): Observable<WordPlan> {
    const params = { level, count, perLesson };
    return this.http.get<WordPlan>(`${BASE}/topics/${topicId}/word-plan`, { params });
  }

  /** F18: asks the AI for `count` new words of `level` and adds them to the topic; returns the added words and the list. */
  suggestTopicWords(topicId: string, level: Level, count: number): Observable<{ added: string[]; words: TopicWord[] }> {
    return this.http.post<{ added: string[]; words: TopicWord[] }>(`${BASE}/topics/${topicId}/words/suggest`, { count, level });
  }

  /** Grammar points of the curriculum for a level, in curriculum order, with how many lessons use each. */
  grammarPoints(level: string, topicId?: string): Observable<GrammarPoint[]> {
    let params = new HttpParams().set('level', level);
    if (topicId) {
      params = params.set('topicId', topicId);
    }
    return this.http.get<{ points: GrammarPoint[] }>(`${BASE}/grammar`, { params }).pipe(map((r) => r.points));
  }

  /** F20: the grammar lessons that exist (points without a lesson are absent). */
  grammarLessons(): Observable<GrammarLessonSummary[]> {
    return this.http.get<{ lessons: GrammarLessonSummary[] }>(`${BASE}/grammar-lessons`).pipe(map((r) => r.lessons));
  }

  /** F20: one grammar lesson with its content; 404 when the point has none. */
  grammarLesson(pointId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(this.http.get<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}`));
  }

  /** F20: one AI request; saves a draft. `force` replaces an edited or published lesson. */
  generateGrammarLesson(pointId: string, force: boolean): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/generate`, { force }),
    );
  }

  saveGrammarLesson(pointId: string, content: GrammarContent): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.put<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}`, { content }),
    );
  }

  /** `acknowledgeFlags` publishes although the AI check left flagged exercises (otherwise 409). */
  publishGrammarLesson(pointId: string, acknowledgeFlags = false): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(
        `${BASE}/grammar-lessons/${pointId}/publish`,
        acknowledgeFlags ? { acknowledgeFlags: true } : null,
      ),
    );
  }

  /** F21: one AI request that solves the exercises on its own and flags the ones that disagree. */
  checkGrammarLesson(pointId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/check`, null),
    );
  }

  /** F21b: the admin says a flagged exercise is right as it is. */
  confirmGrammarCheck(pointId: string, exerciseId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/checks/${exerciseId}/confirm`, null),
    );
  }

  /** F21b: marks the lesson as checked by the admin; 409 grammar_not_checked before any AI check. */
  verifyGrammarLesson(pointId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/verify`, null),
    );
  }

  /** F21b: replaces one exercise (same kind). */
  saveGrammarExercise(pointId: string, exerciseId: string, exercise: GrammarExerciseInput): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.put<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/exercises/${exerciseId}`, exercise),
    );
  }

  /** F21b: removes one exercise; 400 when the section would fall under its minimum. */
  deleteGrammarExercise(pointId: string, exerciseId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.delete<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/exercises/${exerciseId}`),
    );
  }

  /** F21: open learner reports, grouped by (point, exercise), newest first. */
  grammarReports(): Observable<GrammarReportGroup[]> {
    return this.http.get<{ reports: GrammarReportGroup[] }>(`${BASE}/grammar-reports`).pipe(map((r) => r.reports));
  }

  /** F21: closes the open reports of one exercise; returns how many were closed. */
  resolveGrammarReport(pointId: string, exerciseId: string): Observable<number> {
    return this.http
      .post<{ resolved: number }>(`${BASE}/grammar-reports/resolve`, { pointId, exerciseId })
      .pipe(map((r) => r.resolved));
  }

  unpublishGrammarLesson(pointId: string): Observable<GrammarLessonAdmin> {
    return this.unwrapGrammar(
      this.http.post<{ lesson: GrammarLessonAdmin }>(`${BASE}/grammar-lessons/${pointId}/unpublish`, null),
    );
  }

  private unwrapGrammar(obs: Observable<{ lesson: GrammarLessonAdmin }>): Observable<GrammarLessonAdmin> {
    return obs.pipe(map((r) => r.lesson));
  }

  private unwrap(obs: Observable<{ lesson: Lesson }>): Observable<Lesson> {
    return obs.pipe(map((r) => r.lesson));
  }
}
