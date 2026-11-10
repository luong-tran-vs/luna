import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';

import { serverUrl } from '../../core/api-url';

/**
 * The pictures of the irregular verbs (added 2026-11-10), drawn by the admin on "Ảnh động từ": the
 * URL of each verb's picture, loaded once. A verb without one has none, and the pages then hide the
 * picture's frame.
 */
@Injectable({ providedIn: 'root' })
export class VerbPictures {
  private readonly http = inject(HttpClient);
  private requested = false;

  /** The picture URL of each verb that has one, by base form. */
  readonly urls = signal<ReadonlyMap<string, string>>(new Map());

  /** Loads the list the first time; a failure only leaves the pictures out. */
  load(): void {
    if (this.requested) {
      return;
    }
    this.requested = true;
    this.http.get<{ images: { base: string; imageUrl: string }[] }>('/verbs/images').subscribe({
      next: (r) => this.urls.set(new Map(r.images.map((i) => [i.base, serverUrl(i.imageUrl)]))),
      error: () => {
        this.requested = false;
      },
    });
  }

  url(base: string): string {
    return this.urls().get(base) ?? '';
  }
}
