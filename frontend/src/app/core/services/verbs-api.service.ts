import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { VerbProgressList, VerbResult } from '../models/verbs';

/** The learner's irregular verbs: the ones learned and their reviews. */
@Injectable({ providedIn: 'root' })
export class VerbsApiService {
  private readonly http = inject(HttpClient);

  progress(): Observable<VerbProgressList> {
    return this.http.get<VerbProgressList>('/verbs/progress');
  }

  /** Marks verbs as learned (they join the reviews) or no longer learned; returns them. */
  setLearned(bases: string[], learned: boolean): Observable<VerbProgressList> {
    return this.http.post<VerbProgressList>('/verbs/learned', { bases, learned });
  }

  /** Records a finished review round; returns the verbs it changed. */
  record(results: VerbResult[]): Observable<VerbProgressList> {
    return this.http.post<VerbProgressList>('/verbs/results', { results });
  }
}
