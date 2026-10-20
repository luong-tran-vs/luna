import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { TalkSession, TalkSummary, TalkTopic } from '../models/talk';

/** The talk API (added 2026-10-20): conversations with the AI about a studied topic. */
@Injectable({ providedIn: 'root' })
export class TalkApiService {
  private readonly http = inject(HttpClient);

  topics(): Observable<TalkTopic[]> {
    return this.http.get<{ topics: TalkTopic[] }>('/talk/topics').pipe(map((r) => r.topics));
  }

  sessions(topicId = ''): Observable<TalkSummary[]> {
    const params = topicId ? new HttpParams().set('topicId', topicId) : undefined;
    return this.http
      .get<{ sessions: TalkSummary[] }>('/talk/sessions', { params })
      .pipe(map((r) => r.sessions));
  }

  get(id: string): Observable<TalkSession> {
    return this.unwrap(this.http.get<{ session: TalkSession }>(`/talk/sessions/${id}`));
  }

  /** Starts a conversation: the AI picks a situation and says the first line. */
  start(topicId: string): Observable<TalkSession> {
    return this.unwrap(this.http.post<{ session: TalkSession }>('/talk/sessions', { topicId }));
  }

  /** The learner's line; the session comes back with the AI's answer. */
  say(id: string, text: string, spoken: boolean): Observable<TalkSession> {
    return this.unwrap(
      this.http.post<{ session: TalkSession }>(`/talk/sessions/${id}/turns`, { text, spoken }),
    );
  }

  /** Ends the conversation; the session comes back with the review. */
  end(id: string): Observable<TalkSession> {
    return this.unwrap(this.http.post<{ session: TalkSession }>(`/talk/sessions/${id}/end`, null));
  }

  private unwrap(o: Observable<{ session: TalkSession }>): Observable<TalkSession> {
    return o.pipe(map((r) => r.session));
  }
}
