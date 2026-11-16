import { DOCUMENT } from '@angular/common';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, concatMap, from, last, map, Observable, of, tap } from 'rxjs';

import {
  DailyResultBody,
  DailyResultReply,
  DailyView,
  GameId,
  GameResultBody,
  GameResultReply,
  GamesList,
  GrammarRound,
  LettersRound,
  MemoryRound,
  PictureRound,
  SpotCounts,
  SpotGenerateReply,
  SpotImportItem,
  SpotImportReply,
  SpotItem,
  SpotItemEdit,
  VerbMiss,
} from '../models/games';
import { ApiError } from '../interceptors/error-interceptor';

/** The key of the results not sent yet (research R7). */
export const PENDING_KEY = 'luna.games.pending';

/** A result kept on this device until it is sent. */
export interface PendingResult {
  game: GameId;
  body: GameResultBody;
}

/**
 * The games API (F26). A finished round's result that cannot be sent (no network) is kept in
 * localStorage and sent again when the games page opens (flushPending).
 */
@Injectable({ providedIn: 'root' })
export class GamesApiService {
  private readonly http = inject(HttpClient);
  private readonly storage = inject(DOCUMENT).defaultView?.localStorage;

  list(): Observable<GamesList> {
    return this.http.get<GamesList>('/games');
  }

  pictureRound(): Observable<PictureRound> {
    return this.http.get<PictureRound>('/games/picture/round');
  }

  lettersRound(): Observable<LettersRound> {
    return this.http.get<LettersRound>('/games/letters/round');
  }

  /** The pairs of "Lật thẻ"; with a lesson, that lesson's words. */
  memoryRound(lessonId?: string | null): Observable<MemoryRound> {
    const params: Record<string, string> = lessonId ? { lessonId } : {};
    return this.http.get<MemoryRound>('/games/memory/round', { params });
  }

  grammarRound(): Observable<GrammarRound> {
    return this.http.get<GrammarRound>('/games/grammar/round');
  }

  /** Sends a round's result; when the network fails it is kept to be sent later. */
  finish(game: GameId, body: GameResultBody): Observable<GameResultReply | null> {
    return this.http.post<GameResultReply>(`/games/${game}/results`, body).pipe(
      catchError((err: unknown) => {
        if (isNetworkError(err)) {
          this.keep({ game, body });
          return of(null);
        }
        throw err;
      }),
    );
  }

  daily(): Observable<DailyView> {
    return this.http.get<DailyView>('/games/daily');
  }

  dailyResult(body: DailyResultBody): Observable<DailyResultReply> {
    return this.http.post<DailyResultReply>('/games/daily/result', body);
  }

  dailyPractice(wrongWords: string[], wrongVerbs: VerbMiss[]): Observable<{ rescheduled: number }> {
    return this.http.post<{ rescheduled: number }>('/games/daily/practice', {
      wrongWords,
      wrongVerbs,
    });
  }

  /** The results kept on this device. */
  pending(): PendingResult[] {
    try {
      const raw = this.storage?.getItem(PENDING_KEY);
      const list: unknown = raw ? JSON.parse(raw) : [];
      return Array.isArray(list) ? (list as PendingResult[]) : [];
    } catch {
      return [];
    }
  }

  private save(list: PendingResult[]): void {
    try {
      if (list.length) {
        this.storage?.setItem(PENDING_KEY, JSON.stringify(list));
      } else {
        this.storage?.removeItem(PENDING_KEY);
      }
    } catch {
      // No storage (private mode): the result is lost, the game still works.
    }
  }

  private keep(p: PendingResult): void {
    this.save([...this.pending(), p].slice(-20));
  }

  /**
   * Sends the kept results one by one. A result refused by the server (400, 403, 409) is dropped;
   * a network failure keeps it and the ones after it. Emits how many were sent.
   */
  flushPending(): Observable<number> {
    const list = this.pending();
    if (!list.length) {
      return of(0);
    }
    let sent = 0;
    let stopped = false;
    const rest: PendingResult[] = [];
    return from(list).pipe(
      concatMap((p) => {
        if (stopped) {
          rest.push(p);
          return of(null);
        }
        return this.http.post<GameResultReply>(`/games/${p.game}/results`, p.body).pipe(
          tap(() => sent++),
          catchError((err: unknown) => {
            if (isNetworkError(err) || errorStatus(err) >= 500) {
              stopped = true;
              rest.push(p);
            }
            return of(null);
          }),
        );
      }),
      last(),
      tap(() => this.save(rest)),
      map(() => sent),
    );
  }

  // --- admin ---

  spotCounts(): Observable<SpotCounts> {
    return this.http.get<SpotCounts>('/admin/games/spot-counts');
  }

  spotItems(pointId: string): Observable<{ items: SpotItem[] }> {
    return this.http.get<{ items: SpotItem[] }>('/admin/games/spot-items', { params: { pointId } });
  }

  generateSpots(pointId: string): Observable<SpotGenerateReply> {
    return this.http.post<SpotGenerateReply>('/admin/games/spot-items/generate', { pointId });
  }

  /** Approves every draft of the points at once (all points when pointIds is omitted). */
  approveSpotDrafts(pointIds?: string[]): Observable<{ approved: number }> {
    return this.http.post<{ approved: number }>(
      '/admin/games/spot-items/approve-drafts',
      pointIds ? { pointIds } : {},
    );
  }

  /** Sentences written outside the AI; dryRun only checks them. */
  importSpots(
    pointId: string,
    items: SpotImportItem[],
    dryRun: boolean,
  ): Observable<SpotImportReply> {
    return this.http.post<SpotImportReply>('/admin/games/spot-items/import', {
      pointId,
      items,
      dryRun,
    });
  }

  updateSpot(id: string, edit: SpotItemEdit): Observable<{ item: SpotItem }> {
    return this.http.put<{ item: SpotItem }>(`/admin/games/spot-items/${id}`, edit);
  }

  approveSpot(id: string): Observable<{ item: SpotItem }> {
    return this.http.post<{ item: SpotItem }>(`/admin/games/spot-items/${id}/approve`, {});
  }

  deleteSpot(id: string): Observable<void> {
    return this.http.delete<void>(`/admin/games/spot-items/${id}`);
  }
}

/** Whether the request never reached the server (offline, connection lost). */
function isNetworkError(err: unknown): boolean {
  return (err instanceof HttpErrorResponse || err instanceof ApiError) && err.status === 0;
}

/**
 * The HTTP status of a failed request: the app's ApiError (error interceptor) or a raw
 * HttpErrorResponse (tests); 0 when the server was not reached or the error is something else.
 */
export function errorStatus(err: unknown): number {
  if (err instanceof ApiError || err instanceof HttpErrorResponse) {
    return err.status;
  }
  return 0;
}

/** The JSON body of a failed request, null when there is none. */
export function errorBody(err: unknown): unknown {
  if (err instanceof ApiError) {
    return err.body;
  }
  if (err instanceof HttpErrorResponse) {
    return err.error as unknown;
  }
  return null;
}
