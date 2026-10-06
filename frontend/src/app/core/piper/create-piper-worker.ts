import { InjectionToken } from '@angular/core';

/** Starts a Piper worker. Kept apart from piper.messages.ts, which the worker itself imports. */
export function createPiperWorker(): Worker {
  return new Worker(new URL('./piper.worker', import.meta.url), { type: 'module' });
}

/** How NaturalVoiceService starts its worker; specs provide a fake. */
export const PIPER_WORKER = new InjectionToken<() => Worker>('PIPER_WORKER', {
  providedIn: 'root',
  factory: () => createPiperWorker,
});
