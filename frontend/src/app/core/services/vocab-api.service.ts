import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import {
  BulkResult,
  Card,
  CardDetails,
  CardInput,
  CardPage,
  CardQuery,
  DueCard,
  DueList,
  LessonCounts,
  PracticeMisses,
  ReviewInput,
  WordRef,
} from '../models/vocab';

/** The learner's notebook: cards and reviews (contracts/vocab-api.md). */
@Injectable({ providedIn: 'root' })
export class VocabApiService {
  private readonly http = inject(HttpClient);

  saveCard(input: CardInput): Observable<Card> {
    return this.http.post<{ card: Card }>('/api/vocab/cards', input).pipe(map((r) => r.card));
  }

  words(): Observable<WordRef[]> {
    return this.http.get<{ words: WordRef[] }>('/api/vocab/words').pipe(map((r) => r.words));
  }

  due(limit = 50): Observable<DueList> {
    return this.http.get<DueList>('/api/vocab/review/due', { params: { limit: String(limit) } });
  }

  review(cardId: string, input: ReviewInput): Observable<DueCard> {
    return this.http.post<{ card: DueCard }>(`/api/vocab/cards/${cardId}/review`, input).pipe(map((r) => r.card));
  }

  list(query: CardQuery): Observable<CardPage> {
    const params: Record<string, string> = { page: String(query.page ?? 1) };
    if (query.q) {
      params['q'] = query.q;
    }
    if (query.lessonId) {
      params['lessonId'] = query.lessonId;
    }
    return this.http.get<CardPage>('/api/vocab/cards', { params });
  }

  lessons(): Observable<LessonCounts> {
    return this.http.get<LessonCounts>('/api/vocab/lessons');
  }

  update(cardId: string, details: CardDetails): Observable<Card> {
    return this.http.patch<{ card: Card }>(`/api/vocab/cards/${cardId}`, details).pipe(map((r) => r.card));
  }

  remove(cardId: string): Observable<void> {
    return this.http.delete<void>(`/api/vocab/cards/${cardId}`);
  }

  /** Saves words of a lesson's Vocabulary section; words already saved are skipped. */
  bulk(lessonId: string, lemmas: string[]): Observable<BulkResult> {
    return this.http.post<BulkResult>('/api/vocab/cards/bulk', { lessonId, lemmas });
  }

  /** Puts words of a lesson's practice that were answered wrong into the review queue, due now. */
  practiceMisses(lessonId: string, words: string[]): Observable<PracticeMisses> {
    return this.http.post<PracticeMisses>('/api/vocab/practice-misses', { lessonId, words });
  }
}
