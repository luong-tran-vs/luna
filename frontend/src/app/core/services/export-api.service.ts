import { HttpClient, HttpResponse } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

/** A downloaded export: the JSON file and the name the server gave it. */
export interface ExportFile {
  blob: Blob;
  filename: string;
}

const FALLBACK_NAME = 'luna-export.json';

/** Reads the file name from `Content-Disposition: attachment; filename="…"`. */
export function filenameFrom(disposition: string | null): string {
  const match = disposition ? /filename="?([^";]+)"?/i.exec(disposition) : null;
  return match?.[1]?.trim() || FALLBACK_NAME;
}

/** The learner's data export (F13). */
@Injectable({ providedIn: 'root' })
export class ExportApiService {
  private readonly http = inject(HttpClient);

  download(): Observable<ExportFile> {
    return this.http
      .get('/api/export', { responseType: 'blob', observe: 'response' })
      .pipe(
        map((res: HttpResponse<Blob>) => ({
          blob: res.body ?? new Blob([], { type: 'application/json' }),
          filename: filenameFrom(res.headers.get('Content-Disposition')),
        })),
      );
  }
}
