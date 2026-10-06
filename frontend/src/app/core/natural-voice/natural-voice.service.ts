import { DOCUMENT } from '@angular/common';
import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core';

import { PIPER_WORKER } from '../piper/create-piper-worker';
import { PiperRequest, PiperResponse } from '../piper/piper.messages';

/** Also read on start-up; "on" when the learner turned the natural voice on. */
export const NATURAL_VOICE_STORAGE_KEY = 'luna.naturalVoice';

/** The Piper voice (about 60MB, US English, female). */
const VOICE = 'en_US-hfc_female-medium';
/** Generated texts kept in memory (least recently used dropped first). */
const MAX_CACHED = 300;
/** Texts waiting to be generated; the oldest prefetches are dropped beyond this. */
const MAX_QUEUED = 400;

export type NaturalVoiceState = 'off' | 'loading' | 'ready' | 'error';

export interface Clip {
  /** Object URL of a WAV file. */
  url: string;
  /** Length at normal speed. */
  seconds: number;
}

/**
 * The natural voice: a Piper voice (VITS) running in a worker in the learner's browser. Off by
 * default; turning it on downloads the voice once (the browser keeps it). Texts are generated at
 * normal speed; the player changes the speed (playbackRate).
 *
 * Nobody waits for a text to be generated: texts are generated ahead (prefetch)
 * one at a time, and SpeechService plays a text with this voice only once it is ready (cached),
 * reading it with the browser voice meanwhile. A text asked for and not ready yet (want) goes to
 * the front of the queue so the next time it is.
 */
@Injectable({ providedIn: 'root' })
export class NaturalVoiceService {
  private readonly window = inject(DOCUMENT).defaultView;
  private readonly storage = this.window?.localStorage;
  private readonly createWorker = inject(PIPER_WORKER);

  /** Whether this browser can run the model at all. */
  readonly supported = typeof Worker !== 'undefined' && typeof WebAssembly !== 'undefined';

  private readonly _state = signal<NaturalVoiceState>('off');
  private readonly _downloaded = signal({ loaded: 0, total: 0 });
  private readonly _error = signal<string | null>(null);

  readonly state = this._state.asReadonly();
  readonly error = this._error.asReadonly();
  readonly enabled = computed(() => this._state() !== 'off');
  readonly ready = computed(() => this._state() === 'ready');
  /** Download progress of the model files, 0–100. */
  readonly percent = computed(() => {
    const { loaded, total } = this._downloaded();
    return total > 0 ? Math.min(100, Math.round((loaded / total) * 100)) : 0;
  });

  private worker: Worker | null = null;
  private nextId = 1;
  /** Generated texts, least recently used first. */
  private readonly clips = new Map<string, Clip>();
  /** Texts waiting, the next one first. */
  private queue: string[] = [];
  /** The text the worker is on, by message id. */
  private running: { id: number; text: string } | null = null;

  constructor() {
    if (this.readStored() && this.supported) {
      this.start();
    }
    inject(DestroyRef).onDestroy(() => this.shutdown());
  }

  /** Turns the natural voice on: remembers it and loads the model (from the browser cache after the first time). */
  enable(): void {
    if (!this.supported) {
      return;
    }
    this.writeStored(true);
    // Ask the browser not to evict the model when space runs low; refusal is fine.
    void this.window?.navigator.storage?.persist?.().catch(() => false);
    this.start();
  }

  /** Turns it off: the browser voice is used again. The downloaded model stays in the browser cache. */
  disable(): void {
    this.writeStored(false);
    this.shutdown();
    this._state.set('off');
  }

  /** The text's audio when it is already generated, else null. */
  cached(raw: string): Clip | null {
    const text = raw.trim();
    const clip = this.clips.get(text);
    if (clip) {
      // Recently used: move to the end so it is dropped last.
      this.clips.delete(text);
      this.clips.set(text, clip);
    }
    return clip ?? null;
  }

  /** A text someone asked to hear and is not ready: generate it next. */
  want(raw: string): void {
    const text = raw.trim();
    if (!text || !this.ready() || this.clips.has(text) || this.running?.text === text) {
      return;
    }
    this.queue = [text, ...this.queue.filter((t) => t !== text)];
    this.pump();
  }

  /** Texts likely to be heard soon: generate them, in order, after those already waiting. */
  prefetch(texts: readonly string[]): void {
    if (!this.ready()) {
      return;
    }
    const waiting = new Set(this.queue);
    for (const raw of texts) {
      const text = raw.trim();
      if (text && !this.clips.has(text) && this.running?.text !== text && !waiting.has(text)) {
        this.queue.push(text);
        waiting.add(text);
      }
    }
    if (this.queue.length > MAX_QUEUED) {
      this.queue = this.queue.slice(this.queue.length - MAX_QUEUED);
    }
    this.pump();
  }

  private pump(): void {
    if (this.running || !this.worker || this._state() !== 'ready') {
      return;
    }
    const text = this.queue.shift();
    if (text === undefined) {
      return;
    }
    const id = this.nextId++;
    this.running = { id, text };
    this.send({ type: 'speak', id, text });
  }

  private store(text: string, clip: Clip): void {
    this.clips.set(text, clip);
    while (this.clips.size > MAX_CACHED) {
      const [oldest, old] = this.clips.entries().next().value as [string, Clip];
      this.clips.delete(oldest);
      URL.revokeObjectURL(old.url);
    }
  }

  private start(): void {
    if (this.worker) {
      return;
    }
    this._state.set('loading');
    this._error.set(null);
    this._downloaded.set({ loaded: 0, total: 0 });
    try {
      this.worker = this.createWorker();
    } catch (err) {
      this.fail(String(err));
      return;
    }
    this.worker.addEventListener('message', ({ data }: MessageEvent<PiperResponse>) => this.onMessage(data));
    this.worker.addEventListener('error', (e) => this.fail(e.message || 'worker error'));
    this.send({ type: 'load', voice: VOICE });
  }

  private onMessage(data: PiperResponse): void {
    switch (data.type) {
      case 'progress':
        this._downloaded.set({ loaded: data.loaded, total: data.total });
        break;
      case 'loaded':
        this._state.set('ready');
        this.pump();
        break;
      case 'audio':
      case 'error': {
        if (data.id === undefined) {
          this.fail(data.type === 'error' ? data.message : 'unknown error');
          return;
        }
        const run = this.running;
        if (!run || run.id !== data.id) {
          return;
        }
        this.running = null;
        if (data.type === 'audio') {
          this.store(run.text, { url: URL.createObjectURL(new Blob([data.wav], { type: 'audio/wav' })), seconds: data.seconds });
        }
        // A text the model failed on is not kept: the browser voice reads it, and a later want() retries.
        this.pump();
        break;
      }
    }
  }

  /** The model could not load: the browser voice stays in use; turning the option off and on retries. */
  private fail(message: string): void {
    this.shutdown();
    this._error.set(message);
    this._state.set('error');
  }

  private shutdown(): void {
    this.worker?.terminate();
    this.worker = null;
    this.running = null;
    this.queue = [];
    this.clips.forEach((c) => URL.revokeObjectURL(c.url));
    this.clips.clear();
  }

  private send(message: PiperRequest): void {
    this.worker?.postMessage(message);
  }

  private readStored(): boolean {
    try {
      return this.storage?.getItem(NATURAL_VOICE_STORAGE_KEY) === 'on';
    } catch {
      return false;
    }
  }

  private writeStored(on: boolean): void {
    try {
      if (on) {
        this.storage?.setItem(NATURAL_VOICE_STORAGE_KEY, 'on');
      } else {
        this.storage?.removeItem(NATURAL_VOICE_STORAGE_KEY);
      }
    } catch {
      // Storage blocked: the choice holds for this visit only.
    }
  }
}
