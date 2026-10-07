/** Starts a speech-to-text worker. Kept apart from stt.messages.ts, which the worker itself imports. */
export function createSttWorker(): Worker {
  return new Worker(new URL('./stt.worker', import.meta.url), { type: 'module' });
}
