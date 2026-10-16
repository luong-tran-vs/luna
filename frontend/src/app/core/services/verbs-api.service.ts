import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { VerbProgressList, VerbResult } from '../models/verbs';

/** The irregular verbs practice: which verbs the learner knows and the results of a round. */
@Injectable({ providedIn: 'root' })
export class VerbsApiService {
  private readonly http = inject(HttpClient);

  progress(): Observable<VerbProgressList> {
    return this.http.get<VerbProgressList>('/verbs/progress');
  }

  /** Records a finished round; returns the verbs it changed. */
  record(results: VerbResult[]): Observable<VerbProgressList> {
    return this.http.post<VerbProgressList>('/verbs/results', { results });
  }
}
