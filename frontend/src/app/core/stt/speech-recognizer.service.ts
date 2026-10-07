import { computed, DestroyRef, inject, Injectable, InjectionToken, signal } from '@angular/core';

import { createSttWorker } from './create-stt-worker';
import { SttDevice, SttRequest, SttResponse } from './stt.messages';

/**
 * The model of the Speaking step: Moonshine base (about 62MB at 8 bits), made for short English
 * clips. Provisional until the models are measured on phones (admin page "Thử nhận dạng giọng nói").
 */
export const SPEAKING_MODEL = 'onnx-community/moonshine-base-ONNX';
const DEVICE: SttDevice = 'wasm';

export type RecognizerState = 'off' | 'loading' | 'ready' | 'error';

/** How the service starts its worker; specs provide a fake. */
export const STT_WORKER = new InjectionToken<() => Worker>('STT_WORKER', {
  providedIn: 'root',
  factory: () => createSttWorker,
});

/**
 * Speech to text in the learner's browser, for the Speaking step: loads the model on first use
 * (downloaded once, then from the browser's cache) and turns a short recording into text.
 */
@Injectable({ providedIn: 'root' })
export class SpeechRecognizerService {
  private readonly createWorker = inject(STT_WORKER);

  /** Whether this browser can run the model at all. */
  readonly supported = typeof Worker !== 'undefined' && typeof WebAssembly !== 'undefined';

  private readonly _state = signal<RecognizerState>('off');
  private readonly _downloaded = signal({ loaded: 0, total: 0 });
  private readonly _error = signal<string | null>(null);
  readonly state = this._state.asReadonly();
  readonly error = this._error.asReadonly();
  /** Download progress of the model files, 0–100. */
  readonly percent = computed(() => {
    const { loaded, total } = this._downloaded();
    return total > 0 ? Math.min(100, Math.round((loaded / total) * 100)) : 0;
  });
  /** Loading with nothing (left) to download: the model is starting. */
  readonly starting = computed(() => {
    const { loaded, total } = this._downloaded();
    return this._state() === 'loading' && (total === 0 || loaded >= total);
  });

  private worker: Worker | null = null;
  private nextId = 1;
  private readonly waiting = new Map<number, { resolve: (text: string) => void; reject: (err: Error) => void }>();

  constructor() {
    inject(DestroyRef).onDestroy(() => this.shutdown());
  }

  /** Loads the model unless it is loading or loaded; after an error, tries again. */
  load(): void {
    if (!this.supported || this._state() === 'loading' || this._state() === 'ready') {
      return;
    }
    this.shutdown();
    this._state.set('loading');
    this._error.set(null);
    this._downloaded.set({ loaded: 0, total: 0 });
    try {
      this.worker = this.createWorker();
    } catch (err) {
      this.fail(String(err));
      return;
    }
    this.worker.addEventListener('message', ({ data }: MessageEvent<SttResponse>) => this.onMessage(data));
    this.worker.addEventListener('error', (e) => this.fail(e.message || 'worker error'));
    this.send({ type: 'load', model: SPEAKING_MODEL, device: DEVICE });
  }

  /** What was said in a recording (16 kHz mono). Rejects when the model is not ready or fails. */
  transcribe(audio: Float32Array): Promise<string> {
    if (this._state() !== 'ready' || !this.worker) {
      return Promise.reject(new Error('not ready'));
    }
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.waiting.set(id, { resolve, reject });
      const copy = audio.slice();
      this.worker!.postMessage({ type: 'transcribe', id, audio: copy } satisfies SttRequest, [copy.buffer]);
    });
  }

  private onMessage(data: SttResponse): void {
    switch (data.type) {
      case 'progress':
        this._downloaded.set({ loaded: data.loaded, total: data.total });
        break;
      case 'loaded':
        this._state.set('ready');
        break;
      case 'text':
        this.waiting.get(data.id)?.resolve(data.text);
        this.waiting.delete(data.id);
        break;
      case 'error':
        if (data.id === undefined) {
          this.fail(data.message);
        } else {
          this.waiting.get(data.id)?.reject(new Error(data.message));
          this.waiting.delete(data.id);
        }
        break;
    }
  }

  private fail(message: string): void {
    this.shutdown();
    this._error.set(message);
    this._state.set('error');
  }

  private shutdown(): void {
    this.worker?.terminate();
    this.worker = null;
    this.waiting.forEach((w) => w.reject(new Error('stopped')));
    this.waiting.clear();
  }

  private send(message: SttRequest): void {
    this.worker?.postMessage(message);
  }
}
