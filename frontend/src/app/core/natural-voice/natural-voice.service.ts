import { DOCUMENT } from '@angular/common';
import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core';

import { PIPER_WORKER } from '../piper/create-piper-worker';
import { PiperRequest, PiperResponse } from '../piper/piper.messages';

/** Also read on start-up; "on" when the learner turned the natural voice on. */
export const NATURAL_VOICE_STORAGE_KEY = 'luna.naturalVoice';
/** "on" when the learner turned the browser voice off: only the natural voice reads. */
export const NATURAL_VOICE_ONLY_STORAGE_KEY = 'luna.naturalVoiceOnly';

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
 *
 * With "only" on, the browser voice is never used: a text not ready yet is generated first and
 * played once it is (request), even while the model is still loading; when the model cannot load,
 * nothing is read (the learner sees an error) rather than falling back to the browser voice.
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
  private readonly _only = signal(this.readStored(NATURAL_VOICE_ONLY_STORAGE_KEY));
  /** The learner turned the browser voice off (kept even while the natural voice is off). */
  readonly only = this._only.asReadonly();
  /** True when the browser voice must not read: "only" is on and the natural voice is turned on. */
  readonly exclusive = computed(() => this._only() && this._state() !== 'off');
  /** Download progress of the model files, 0–100. */
  readonly percent = computed(() => {
    const { loaded, total } = this._downloaded();
    return total > 0 ? Math.min(100, Math.round((loaded / total) * 100)) : 0;
  });
  /**
   * Loading with nothing (left) to download: the voice comes from the browser's storage and the
   * model is starting, which takes a few seconds on every visit.
   */
  readonly starting = computed(() => {
    const { loaded, total } = this._downloaded();
    return this._state() === 'loading' && (total === 0 || loaded >= total);
  });

  private worker: Worker | null = null;
  private nextId = 1;
  /** Generated texts, least recently used first. */
  private readonly clips = new Map<string, Clip>();
  /** Texts waiting, the next one first. */
  private queue: string[] = [];
  /** The text the worker is on, by message id. */
  private running: { id: number; text: string } | null = null;
  /** Callers of request() waiting for a text, by text. */
  private readonly waiters = new Map<string, ((clip: Clip | null) => void)[]>();

  constructor() {
    if (this.readStored(NATURAL_VOICE_STORAGE_KEY) && this.supported) {
      this.start();
    }
    inject(DestroyRef).onDestroy(() => this.shutdown());
  }

  /** Turns the natural voice on: remembers it and loads the model (from the browser cache after the first time). */
  enable(): void {
    if (!this.supported) {
      return;
    }
    this.writeStored(NATURAL_VOICE_STORAGE_KEY, true);
    this.start();
  }

  /** Turns it off: the browser voice is used again. The downloaded model stays in the browser cache. */
  disable(): void {
    this.writeStored(NATURAL_VOICE_STORAGE_KEY, false);
    this.shutdown();
    this._state.set('off');
  }

  /** Turns the browser voice off (only the natural voice reads) or back on. */
  setOnly(on: boolean): void {
    this.writeStored(NATURAL_VOICE_ONLY_STORAGE_KEY, on);
    this._only.set(on);
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

  /**
   * The text's audio, generated next when not ready: resolves once it is, or with null when the
   * model fails on it, cannot load or is turned off. Works while the model is still loading.
   */
  request(raw: string): Promise<Clip | null> {
    const text = raw.trim();
    const clip = this.cached(text);
    if (clip || !text || (this._state() !== 'loading' && this._state() !== 'ready')) {
      return Promise.resolve(clip);
    }
    return new Promise((resolve) => {
      this.waiters.set(text, [...(this.waiters.get(text) ?? []), resolve]);
      if (this.running?.text !== text) {
        this.queue = [text, ...this.queue.filter((t) => t !== text)];
        this.pump();
      }
    });
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

  private resolve(text: string, clip: Clip | null): void {
    const waiting = this.waiters.get(text);
    this.waiters.delete(text);
    waiting?.forEach((done) => done(clip));
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
    // Ask the browser not to evict the stored model when space runs low (also for learners who
    // turned the voice on before this was asked); refusal is fine.
    void this.window?.navigator.storage?.persist?.().catch(() => false);
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
        let clip: Clip | null = null;
        if (data.type === 'audio') {
          clip = { url: URL.createObjectURL(new Blob([data.wav], { type: 'audio/wav' })), seconds: data.seconds };
          this.store(run.text, clip);
        }
        this.resolve(run.text, clip);
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
    [...this.waiters.keys()].forEach((text) => this.resolve(text, null));
    this.clips.forEach((c) => URL.revokeObjectURL(c.url));
    this.clips.clear();
  }

  private send(message: PiperRequest): void {
    this.worker?.postMessage(message);
  }

  private readStored(key: string): boolean {
    try {
      return this.storage?.getItem(key) === 'on';
    } catch {
      return false;
    }
  }

  private writeStored(key: string, on: boolean): void {
    try {
      if (on) {
        this.storage?.setItem(key, 'on');
      } else {
        this.storage?.removeItem(key);
      }
    } catch {
      // Storage blocked: the choice holds for this visit only.
    }
  }
}
