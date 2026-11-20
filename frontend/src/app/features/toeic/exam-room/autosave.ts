import { apiUrl } from '../../../core/api-url';
import { SaveProgress } from '../../../core/models/toeic';

/** Saves wait this long after a choice, so quick clicks go in one request. */
export const CHOICE_DELAY_MS = 1000;
/** While the clock runs, the progress is saved this often. */
export const TICK_SAVE_MS = 5000;

/**
 * Sends the progress while the page is being hidden or closed (research R4): a keepalive fetch
 * outlives the page, unlike an HttpClient request.
 */
export function saveOnHide(attemptId: string, body: SaveProgress): void {
  try {
    void fetch(apiUrl(`/toeic/attempts/${encodeURIComponent(attemptId)}`), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      credentials: 'include',
      keepalive: true,
    }).catch(() => undefined);
  } catch {
    // Nothing more to do: the 5-second saves and the queue already hold most of it.
  }
}

/**
 * The active time of an attempt: what was spent before plus the current run, which stops while the
 * page is hidden (research R4: the time does not run while the learner is away).
 */
export interface ActiveTime {
  before: number;
  since: number | null;
}

export function activeMs(t: ActiveTime, now: number): number {
  return t.before + (t.since === null ? 0 : now - t.since);
}

export function pauseActive(t: ActiveTime, now: number): ActiveTime {
  return t.since === null ? t : { before: activeMs(t, now), since: null };
}

export function resumeActive(t: ActiveTime, now: number): ActiveTime {
  return t.since === null ? { ...t, since: now } : t;
}
