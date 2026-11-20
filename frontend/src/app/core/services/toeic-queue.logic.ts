import { SaveProgress } from '../models/toeic';

/**
 * Merges a newer progress into one not sent yet (research R4): answers are merged (a later -1 still
 * clears the choice on the server), the smaller time left and the larger elapsed time win, the
 * phase never goes back to the listening, and the rest takes the newer value when it has one.
 */
export function mergeProgress(older: SaveProgress | null, newer: SaveProgress): SaveProgress {
  if (!older) {
    return { ...newer, answers: newer.answers ? { ...newer.answers } : undefined };
  }
  const out: SaveProgress = { ...older, ...dropUndefined(newer) };
  if (older.answers || newer.answers) {
    out.answers = { ...(older.answers ?? {}), ...(newer.answers ?? {}) };
  }
  if (older.remainingMs !== undefined && newer.remainingMs !== undefined) {
    out.remainingMs = Math.min(older.remainingMs, newer.remainingMs);
  }
  if (older.elapsedMs !== undefined && newer.elapsedMs !== undefined) {
    out.elapsedMs = Math.max(older.elapsedMs, newer.elapsedMs);
  }
  if (older.phase === 'reading' && newer.phase === 'listening') {
    out.phase = 'reading';
  }
  if (older.listeningSegment !== undefined && newer.listeningSegment !== undefined) {
    out.listeningSegment = Math.max(older.listeningSegment, newer.listeningSegment);
  }
  return out;
}

function dropUndefined(p: SaveProgress): SaveProgress {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(p)) {
    if (v !== undefined) {
      out[k] = v;
    }
  }
  return out as SaveProgress;
}

/** Whether a progress carries nothing to send. */
export function isEmptyProgress(p: SaveProgress): boolean {
  return Object.values(dropUndefined(p)).every(
    (v) => typeof v === 'object' && v !== null && !Array.isArray(v) && Object.keys(v).length === 0,
  );
}
