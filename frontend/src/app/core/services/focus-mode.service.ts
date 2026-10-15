import { Injectable, signal } from '@angular/core';

/**
 * Pages that take the whole phone screen for a while without their own route (a review session):
 * while on, the learner layout hides its tab bar like on a lesson.
 */
@Injectable({ providedIn: 'root' })
export class FocusMode {
  readonly on = signal(false);
}
