/** Messages between the page and the Piper worker (piper.worker.ts). */

export type PiperRequest = { type: 'load'; voice: string } | { type: 'speak'; id: number; text: string };

export type PiperResponse =
  | { type: 'progress'; loaded: number; total: number }
  /** ms: download and set-up; warmupMs: the first (throw-away) run; isolated: WASM may use several threads. */
  | { type: 'loaded'; ms: number; warmupMs: number; isolated: boolean }
  | { type: 'audio'; id: number; ms: number; wav: ArrayBuffer; seconds: number }
  | { type: 'error'; id?: number; message: string };
