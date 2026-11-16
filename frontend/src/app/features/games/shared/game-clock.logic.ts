/**
 * A countdown read from a monotonic clock (performance.now()), not counted with setInterval, so a
 * slow or throttled timer never makes it drift (research R7). Pure: the caller passes the time.
 */
export interface ClockState {
  /** The length of the countdown, ms. */
  duration: number;
  /** Time already used before the current run, ms. */
  used: number;
  /** When the current run started (ms, performance.now()); null while paused. */
  since: number | null;
}

/** A clock of duration ms started at now. */
export function startClock(duration: number, now: number): ClockState {
  return { duration, used: 0, since: now };
}

/** Stops the clock at now, keeping the time used. */
export function pauseClock(c: ClockState, now: number): ClockState {
  return c.since === null ? c : { ...c, used: c.used + (now - c.since), since: null };
}

/** Starts a paused clock again at now. */
export function resumeClock(c: ClockState, now: number): ClockState {
  return c.since !== null ? c : { ...c, since: now };
}

/** The time left at now, ms, never below 0. */
export function remaining(c: ClockState, now: number): number {
  const used = c.used + (c.since === null ? 0 : now - c.since);
  return Math.max(0, c.duration - used);
}

/** Whether the time is up at now. */
export function timeUp(c: ClockState, now: number): boolean {
  return remaining(c, now) <= 0;
}

/** "0:44" for 44 000 ms (rounded up, so 0:01 shows until the end). */
export function clockLabel(ms: number): string {
  const s = Math.ceil(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
