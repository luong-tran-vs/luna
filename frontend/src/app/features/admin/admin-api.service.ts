import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { AiUsage } from '../../core/models/ai-usage';
import { GrammarPoint } from '../../core/models/grammar';
import { GrammarContent, GrammarExerciseInput, GrammarLessonAdmin, GrammarLessonSummary, GrammarReportGroup } from '../../core/models/grammar-admin';
import { GenerateInput, GenerateResult } from '../../core/models/generate';
import {
  AnnotationInput,
  ExtrasInput,
  TranslationInput,
  FixSuggestion,
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
import {
  BankMeaningsFilled,
  BankMissing,
  BankPage,
  BankWord,
  BankWordAdded,
  BankWordDetails,
  BankWordInput,
} from '../../core/models/word-bank';
import { Account, AccountInput } from '../../core/models/user';

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

  /** F22: asks the AI for a corrected version of one flagged item; nothing is saved. */
  suggestFix(id: string, area: FlagArea, index: number): Observable<FixSuggestion> {
    return this.http
      .post<{ suggestion: FixSuggestion }>(`${BASE}/lessons/${id}/check/suggest`, { area, index })
      .pipe(map((r) => r.suggestion));
  }

  /** F22: saves the admin's correction of one item; the other flags of its part are kept. */
  applyFix(id: string, fix: FixSuggestion): Observable<Lesson> {
    return this.unwrap(this.http.post<{ lesson: Lesson }>(`${BASE}/lessons/${id}/check/apply`, fix));
  }

  /** Shows the lesson to learners (published) or hides it again as a draft. */
  setPublished(id: string, published: boolean): Observable<Lesson> {
    return this.unwrap(this.http.put<{ lesson: Lesson }>(`${BASE}/lessons/${id}/published`, { published }));
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

  /**
   * F24: one page (from 1) of the word bank, filtered by text, by what the words lack and, with a
   * topicId, to the words of that topic.
   */
  bankWords(q: string, missing: BankMissing, topicId: string, page: number): Observable<BankPage> {
    let params = new HttpParams().set('page', page);
    if (q) {
      params = params.set('q', q);
    }
    if (missing) {
      params = params.set('missing', missing);
    }
    if (topicId) {
      params = params.set('topicId', topicId);
    }
    return this.http.get<BankPage>(`${BASE}/words`, { params });
  }

  /**
   * F24: adds a word; an empty IPA or meaning is filled from the dictionary. With a topic the word
   * also goes into its list (an error when the list holds it already).
   */
  addBankWord(input: BankWordInput): Observable<BankWordAdded> {
    return this.http.post<BankWordAdded>(`${BASE}/words`, input);
  }

  /** F24: adds the topic words missing from the bank; returns how many were added. */
  /** Requests and tokens sent to the AI: last minute, last hour by minute, today, last 7 days. */
  aiUsage(): Observable<AiUsage> {
    return this.http.get<AiUsage>(`${BASE}/ai-usage`);
  }

  importBankWords(): Observable<number> {
    return this.http.post<{ added: number }>(`${BASE}/words/import`, null).pipe(map((r) => r.added));
  }

  /**
   * F24: one AI request for what the topic's bank words lack (meaning, IPA); returns how many
   * words were asked about and how many got a meaning and an IPA.
   */
  fillBankMissing(topicId: string): Observable<BankMeaningsFilled> {
    return this.http.post<BankMeaningsFilled>(`${BASE}/words/fill-missing`, { topicId });
  }

  updateBankWord(lemma: string, details: BankWordDetails): Observable<BankWord> {
    return this.http.patch<BankWord>(`${BASE}/words/${encodeURIComponent(lemma)}`, details);
  }

  deleteBankWord(lemma: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/words/${encodeURIComponent(lemma)}`);
  }

  uploadBankImage(lemma: string, file: Blob): Observable<BankWord> {
    return this.http.put<BankWord>(`${BASE}/words/${encodeURIComponent(lemma)}/image`, file, {
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
    });
  }

  importBankImage(lemma: string, url: string): Observable<BankWord> {
    return this.http.post<BankWord>(`${BASE}/words/${encodeURIComponent(lemma)}/image/import`, { url });
  }

  /** F24: one AI request that draws the word's picture (paid). */
  generateBankImage(lemma: string, style = ''): Observable<BankWord> {
    return this.http.post<BankWord>(`${BASE}/words/${encodeURIComponent(lemma)}/image/generate`, { style });
  }

  deleteBankImage(lemma: string): Observable<BankWord> {
    return this.http.delete<BankWord>(`${BASE}/words/${encodeURIComponent(lemma)}/image`);
  }

  private unwrapGrammar(obs: Observable<{ lesson: GrammarLessonAdmin }>): Observable<GrammarLessonAdmin> {
    return obs.pipe(map((r) => r.lesson));
  }

  /** Every account with its role (GET /api/admin/users). */
  accounts(): Observable<Account[]> {
    return this.http.get<{ users: Account[] }>(`${BASE}/users`).pipe(map((r) => r.users));
  }

  createAccount(input: AccountInput): Observable<Account> {
    return this.http.post<{ user: Account }>(`${BASE}/users`, input).pipe(map((r) => r.user));
  }

  /** An empty password keeps the current one; the server refuses to change your own role. */
  updateAccount(id: string, input: AccountInput): Observable<Account> {
    return this.http.put<{ user: Account }>(`${BASE}/users/${id}`, input).pipe(map((r) => r.user));
  }

  /** Deletes the account with its learning data; the server refuses to delete your own. */
  deleteAccount(id: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/users/${id}`);
  }

  private unwrap(obs: Observable<{ lesson: Lesson }>): Observable<Lesson> {
    return obs.pipe(map((r) => r.lesson));
  }
}
