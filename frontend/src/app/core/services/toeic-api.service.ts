import { DOCUMENT } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';

import {
  AttemptView,
  CheckReply,
  ComposeInput,
  ComposePlan,
  DeleteGroupsReport,
  DraftInput,
  DraftReport,
  GroupAdmin,
  GroupFull,
  GroupImages,
  ImportReport,
  MistakesView,
  MissingPictures,
  PhotoBatch,
  QuestionDetail,
  ResultView,
  RoundItem,
  SaveProgress,
  StartAttempt,
  TestAdmin,
  TestDetail,
  TestInput,
  ToeicLibrary,
  ToeicPart,
  ToeicStatus,
} from '../models/toeic';
import { ApiError } from '../interceptors/error-interceptor';
import { isEmptyProgress, mergeProgress } from './toeic-queue.logic';

/** The key prefix of the progress not sent yet, per attempt. */
export const PENDING_PREFIX = 'toeic-pending:';

/**
 * The TOEIC API (F27). A progress that cannot be sent (no network) is kept in localStorage, merged
 * with the later ones, and sent again by flush() when the network is back.
 */
@Injectable({ providedIn: 'root' })
export class ToeicApiService {
  private readonly http = inject(HttpClient);
  private readonly storage = inject(DOCUMENT).defaultView?.localStorage;

  // --- learner ---

  library(): Observable<ToeicLibrary> {
    return this.http.get<ToeicLibrary>('/toeic');
  }

  test(key: string): Observable<TestDetail> {
    return this.http.get<TestDetail>(`/toeic/tests/${encodeURIComponent(key)}`);
  }

  start(body: StartAttempt): Observable<AttemptView> {
    return this.http.post<AttemptView>('/toeic/attempts', body);
  }

  attempt(id: string): Observable<AttemptView> {
    return this.http.get<AttemptView>(`/toeic/attempts/${encodeURIComponent(id)}`);
  }

  /**
   * Saves progress; on a network error it is queued and the call still completes (false). A
   * submitted attempt (409) drops what was queued.
   */
  save(id: string, progress: SaveProgress): Observable<boolean> {
    const body = mergeProgress(this.pending(id), progress);
    if (isEmptyProgress(body)) {
      return of(true);
    }
    return this.http.put<void>(`/toeic/attempts/${encodeURIComponent(id)}`, body).pipe(
      tap(() => this.setPending(id, null)),
      map(() => true),
      catchError((err: unknown) => {
        if (err instanceof ApiError && err.kind === 'network') {
          this.setPending(id, body);
          return of(false);
        }
        this.setPending(id, null);
        throw err;
      }),
    );
  }

  /** Sends what is queued for an attempt, if anything. */
  flush(id: string): Observable<boolean> {
    return this.pending(id) ? this.save(id, {}) : of(true);
  }

  /** The progress queued for an attempt, if any. */
  pending(id: string): SaveProgress | null {
    try {
      const raw = this.storage?.getItem(PENDING_PREFIX + id);
      return raw ? (JSON.parse(raw) as SaveProgress) : null;
    } catch {
      return null;
    }
  }

  private setPending(id: string, p: SaveProgress | null): void {
    try {
      if (p) {
        this.storage?.setItem(PENDING_PREFIX + id, JSON.stringify(p));
      } else {
        this.storage?.removeItem(PENDING_PREFIX + id);
      }
    } catch {
      // Storage may be off (private mode): the progress just is not kept.
    }
  }

  /** Submits with the last progress (the queued one merged in). */
  submit(id: string, last: SaveProgress): Observable<ResultView> {
    const body = mergeProgress(this.pending(id), last);
    return this.http
      .post<ResultView>(`/toeic/attempts/${encodeURIComponent(id)}/submit`, body)
      .pipe(tap(() => this.setPending(id, null)));
  }

  result(id: string): Observable<ResultView> {
    return this.http.get<ResultView>(`/toeic/attempts/${encodeURIComponent(id)}/result`);
  }

  question(id: string, qid: string): Observable<QuestionDetail> {
    return this.http.get<QuestionDetail>(
      `/toeic/attempts/${encodeURIComponent(id)}/questions/${encodeURIComponent(qid)}`,
    );
  }

  saveWord(body: {
    word: string;
    ipa?: string;
    meaningVi: string;
    sentence?: string;
  }): Observable<void> {
    return this.http.post<void>('/toeic/vocab', body);
  }

  mistakes(part?: ToeicPart): Observable<MistakesView> {
    return this.http.get<MistakesView>('/toeic/mistakes', { params: part ? { part } : {} });
  }

  mistakeRound(part: ToeicPart | null, count: number): Observable<{ items: RoundItem[] }> {
    const params: Record<string, number> = { count };
    if (part) {
      params['part'] = part;
    }
    return this.http.get<{ items: RoundItem[] }>('/toeic/mistakes/round', { params });
  }

  checkMistake(qid: string, choice: number): Observable<CheckReply> {
    return this.http.post<CheckReply>(`/toeic/mistakes/${encodeURIComponent(qid)}/check`, {
      choice,
    });
  }

  finishMistakes(): Observable<void> {
    return this.http.post<void>('/toeic/mistakes/finish', {});
  }

  // --- admin ---

  importGroups(text: unknown, replace: boolean, dryRun: boolean): Observable<ImportReport> {
    const body = { ...(text as object), replace };
    return this.http.post<ImportReport>('/admin/toeic/import', body, {
      params: dryRun ? { dryRun: 'true' } : {},
    });
  }

  groups(filter: { part?: ToeicPart | null; status?: ToeicStatus | '' }): Observable<GroupAdmin[]> {
    const params: Record<string, string> = {};
    if (filter.part) {
      params['part'] = String(filter.part);
    }
    if (filter.status) {
      params['status'] = filter.status;
    }
    return this.http
      .get<{ groups: GroupAdmin[] }>('/admin/toeic/groups', { params })
      .pipe(map((r) => r.groups));
  }

  group(key: string): Observable<GroupFull> {
    return this.http
      .get<{ group: GroupFull }>(`/admin/toeic/groups/${encodeURIComponent(key)}`)
      .pipe(map((r) => r.group));
  }

  setGroupStatus(key: string, status: ToeicStatus): Observable<unknown> {
    return this.http.patch(`/admin/toeic/groups/${encodeURIComponent(key)}`, { status });
  }

  setGroupsStatus(keys: string[], status: ToeicStatus): Observable<{ changed: number }> {
    return this.http.post<{ changed: number }>('/admin/toeic/groups/status', { keys, status });
  }

  /** Deletes groups from the bank (with their pictures); the ones in use are kept and reported. */
  deleteGroups(keys: string[]): Observable<DeleteGroupsReport> {
    return this.http.post<DeleteGroupsReport>('/admin/toeic/groups/delete', { keys });
  }

  tests(): Observable<TestAdmin[]> {
    return this.http.get<{ tests: TestAdmin[] }>('/admin/toeic/tests').pipe(map((r) => r.tests));
  }

  testAdmin(key: string): Observable<TestAdmin> {
    return this.http
      .get<{ test: TestAdmin }>(`/admin/toeic/tests/${encodeURIComponent(key)}`)
      .pipe(map((r) => r.test));
  }

  createTest(body: TestInput | unknown): Observable<TestAdmin> {
    return this.http.post<{ test: TestAdmin }>('/admin/toeic/tests', body).pipe(map((r) => r.test));
  }

  updateTest(key: string, body: TestInput | unknown): Observable<TestAdmin> {
    return this.http
      .put<{ test: TestAdmin }>(`/admin/toeic/tests/${encodeURIComponent(key)}`, body)
      .pipe(map((r) => r.test));
  }

  setTestStatus(key: string, status: ToeicStatus): Observable<TestAdmin> {
    return this.http
      .patch<{ test: TestAdmin }>(`/admin/toeic/tests/${encodeURIComponent(key)}`, { status })
      .pipe(map((r) => r.test));
  }

  /** Deletes a test that is not published and no learner has taken; its groups stay in the bank. */
  deleteTest(key: string): Observable<void> {
    return this.http.delete<void>(`/admin/toeic/tests/${encodeURIComponent(key)}`);
  }

  /** "Ghép đề tự động": the plan (dryRun) or the draft test made from the published groups. */
  composeTest(input: ComposeInput): Observable<{ plan: ComposePlan; test?: TestAdmin }> {
    return this.http.post<{ plan: ComposePlan; test?: TestAdmin }>(
      '/admin/toeic/tests/compose',
      input,
    );
  }

  /** "Nhờ AI soạn": new groups of one part, kept as drafts. */
  draftGroups(input: DraftInput): Observable<DraftReport> {
    return this.http.post<DraftReport>('/admin/toeic/groups/draft', input);
  }

  groupImages(key: string): Observable<GroupImages> {
    return this.http.get<GroupImages>(`/admin/toeic/groups/${encodeURIComponent(key)}/images`);
  }

  drawImage(key: string, slot: string, engine: string): Observable<GroupImages> {
    return this.http.post<GroupImages>(
      `/admin/toeic/images/${encodeURIComponent(key)}/${encodeURIComponent(slot)}/generate`,
      { engine },
    );
  }

  uploadImage(key: string, slot: string, file: Blob): Observable<GroupImages> {
    return this.http.put<GroupImages>(
      `/admin/toeic/images/${encodeURIComponent(key)}/${encodeURIComponent(slot)}`,
      file,
      { headers: { 'Content-Type': file.type || 'application/octet-stream' } },
    );
  }

  deleteImage(key: string, slot: string): Observable<GroupImages> {
    return this.http.delete<GroupImages>(
      `/admin/toeic/images/${encodeURIComponent(key)}/${encodeURIComponent(slot)}`,
    );
  }

  photoBatch(): Observable<{ batch: PhotoBatch; codex: boolean }> {
    return this.http.get<{ batch: PhotoBatch; codex: boolean }>('/admin/toeic/images/batch');
  }

  /**
   * Draws the missing pictures of a test (its Part 1 photos) or of a part of the bank (1, 6, 7), of
   * some levels only when `levels` is given.
   */
  startPhotoBatch(
    target: { testKey?: string; part?: number; keys?: string[]; levels?: number[] },
    engine: string,
  ): Observable<PhotoBatch> {
    return this.http
      .post<{ batch: PhotoBatch }>('/admin/toeic/images/batch', { ...target, engine })
      .pipe(map((r) => r.batch));
  }

  /**
   * How many pictures a batch of each picture part would draw, by part ("1", "6", "7"): in all and
   * by level (`byLevel["1"]["450"]`).
   */
  missingPictures(): Observable<MissingPictures> {
    return this.http.get<MissingPictures>('/admin/toeic/images/missing');
  }

  stopPhotoBatch(): Observable<PhotoBatch> {
    return this.http
      .post<{ batch: PhotoBatch }>('/admin/toeic/images/batch/stop', {})
      .pipe(map((r) => r.batch));
  }
}
