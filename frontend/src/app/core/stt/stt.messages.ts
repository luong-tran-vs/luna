/** Where the model runs: the CPU (WebAssembly) or the GPU (WebGPU, where the browser has it). */
export type SttDevice = 'wasm' | 'webgpu';

/** Messages to the speech-to-text worker (stt.worker.ts). */
export type SttRequest =
  | { type: 'load'; model: string; device: SttDevice }
  /** Mono audio at 16 kHz. */
  | { type: 'transcribe'; id: number; audio: Float32Array };

/** Messages from the speech-to-text worker. */
export type SttResponse =
  | { type: 'progress'; loaded: number; total: number }
  /** ms: download and set-up; warmupMs: a first run on one second of silence. */
  | { type: 'loaded'; ms: number; warmupMs: number }
  | { type: 'text'; id: number; text: string; ms: number }
  /** Without an id: the model could not load. */
  | { type: 'error'; id?: number; message: string };
