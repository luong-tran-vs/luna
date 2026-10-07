import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { PracticeView } from '../../core/models/practice';
import { AnswerInput, AnswerResult, AskResult, LookupResult, ReadingLesson } from '../../core/models/reading';
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

  /** Asks the AI what text means in that sentence (F9). */
  ask(id: string, text: string, sentenceIndex: number): Observable<AskResult> {
    return this.http.post<AskResult>(`/api/lessons/${id}/ask`, { text, sentenceIndex });
  }

  vocabulary(id: string): Observable<LessonVocabulary> {
    return this.http.get<LessonVocabulary>(`/api/lessons/${id}/vocabulary`);
  }

  /** Sends one comprehension answer (F15). */
  answer(id: string, input: AnswerInput): Observable<AnswerResult> {
    return this.http.post<AnswerResult>(`/api/lessons/${id}/answers`, input);
  }

  /** Forgets this learner's comprehension answers, to answer the questions again. */
  resetAnswers(id: string): Observable<void> {
    return this.http.delete<void>(`/api/lessons/${id}/answers`);
  }

  /** The lesson's practice steps (F17); empty when not generated yet. */
  practice(id: string): Observable<PracticeView> {
    return this.http.get<PracticeView>(`/api/lessons/${id}/practice`);
  }
}
