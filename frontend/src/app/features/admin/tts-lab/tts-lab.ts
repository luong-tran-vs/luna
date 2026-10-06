import { DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';

import { createPiperWorker } from '../../../core/piper/create-piper-worker';
import { PiperRequest, PiperResponse } from '../../../core/piper/piper.messages';
import { SpeechService } from '../../../core/services/speech.service';

/** Piper voices to try (VITS, about 60MB each). The natural voice of the app uses the first. */
const VOICES: readonly { id: string; label: string }[] = [
  { id: 'en_US-hfc_female-medium', label: 'en_US hfc_female (nữ, Mỹ — giọng app đang dùng)' },
  { id: 'en_US-lessac-medium', label: 'en_US lessac (nữ, Mỹ)' },
  { id: 'en_US-amy-medium', label: 'en_US amy (nữ, Mỹ)' },
  { id: 'en_US-kristin-medium', label: 'en_US kristin (nữ, Mỹ)' },
  { id: 'en_US-ryan-medium', label: 'en_US ryan (nam, Mỹ)' },
  { id: 'en_US-joe-medium', label: 'en_US joe (nam, Mỹ)' },
  { id: 'en_GB-alba-medium', label: 'en_GB alba (nữ, Anh)' },
  { id: 'en_GB-cori-medium', label: 'en_GB cori (nữ, Anh)' },
];

const SAMPLE = `On Sunday, my family has lunch at my grandmother's house. My father cooks fish and rice. ` +
  `My little brother sets the table, and I wash the vegetables. After lunch, we sit in the garden and talk. ` +
  `I love these quiet afternoons with my family.`;

interface Row {
  id: number;
  text: string;
  /** Generation time in seconds, null while running. */
  generated: number | null;
  /** Audio length in seconds (normal speed). */
  seconds: number | null;
  url: string | null;
  error: string | null;
}

type LoadState = 'idle' | 'loading' | 'ready' | 'error';

/**
 * Admin tool: runs a Piper voice in the browser (in a worker, as the learners' natural voice does)
 * and measures how long it takes to load and each sentence to generate, next to the browser voice.
 */
@Component({
  selector: 'lu-tts-lab',
  imports: [DecimalPipe],
  templateUrl: './tts-lab.html',
  styleUrl: './tts-lab.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TtsLab {
  private readonly speech = inject(SpeechService);
  private worker: Worker | null = null;
  private nextId = 1;
  private readonly audio = new Audio();
  private queue: number[] = [];

  protected readonly voices = VOICES;
  protected readonly cores = typeof navigator !== 'undefined' ? navigator.hardwareConcurrency : 0;
  protected readonly memory =
    typeof navigator !== 'undefined' ? ((navigator as { deviceMemory?: number }).deviceMemory ?? null) : null;
  protected readonly speechSupported = this.speech.supported;
  /** WASM can use several CPU threads only when the page is cross-origin isolated (COOP/COEP headers). */
  protected readonly isolated = typeof crossOriginIsolated !== 'undefined' && crossOriginIsolated;

  protected readonly voice = signal(VOICES[0].id);
  protected readonly speed = signal(1);
  protected readonly text = signal(SAMPLE);

  protected readonly loadState = signal<LoadState>('idle');
  protected readonly loadError = signal<string | null>(null);
  protected readonly downloaded = signal({ loaded: 0, total: 0 });
  protected readonly loadSeconds = signal<number | null>(null);
  /** The first, throw-away run after loading. */
  protected readonly warmupSeconds = signal<number | null>(null);
  /** The voice loaded. */
  protected readonly loadedVoice = signal<string | null>(null);

  protected readonly rows = signal<Row[]>([]);
  protected readonly running = computed(() => this.rows().some((r) => r.generated === null && r.error === null));
  protected readonly browserSeconds = signal<number | null>(null);

  protected readonly downloadPercent = computed(() => {
    const { loaded, total } = this.downloaded();
    return total > 0 ? Math.min(100, (loaded / total) * 100) : 0;
  });

  protected readonly summary = computed(() => {
    const done = this.rows().filter((r) => r.generated !== null && r.seconds);
    if (done.length === 0) {
      return null;
    }
    const generated = done.reduce((s, r) => s + (r.generated ?? 0), 0);
    const seconds = done.reduce((s, r) => s + (r.seconds ?? 0), 0);
    return { first: done[0].generated ?? 0, generated, seconds, ratio: generated / seconds };
  });

  constructor() {
    this.audio.addEventListener('ended', () => this.playNext());
    inject(DestroyRef).onDestroy(() => {
      this.worker?.terminate();
      this.audio.pause();
      this.speech.stop();
      this.rows().forEach((r) => r.url && URL.revokeObjectURL(r.url));
    });
  }

  /** Starts a fresh worker and loads the chosen voice. */
  protected load(): void {
    this.worker?.terminate();
    this.worker = createPiperWorker();
    this.worker.addEventListener('message', ({ data }: MessageEvent<PiperResponse>) => this.onMessage(data));
    this.worker.addEventListener('error', (e) => this.failLoad(e.message || 'Worker lỗi.'));
    this.loadedVoice.set(null);
    this.loadState.set('loading');
    this.loadError.set(null);
    this.loadSeconds.set(null);
    this.warmupSeconds.set(null);
    this.downloaded.set({ loaded: 0, total: 0 });
    this.send({ type: 'load', voice: this.voice() });
  }

  /** Generates every sentence in turn; each plays as soon as it is ready. */
  protected speakModel(): void {
    if (this.loadState() !== 'ready' || this.running()) {
      return;
    }
    this.audio.pause();
    this.queue = [];
    this.rows().forEach((r) => r.url && URL.revokeObjectURL(r.url));
    const rows = splitSentences(this.text()).map((text) => ({
      id: this.nextId++,
      text,
      generated: null,
      seconds: null,
      url: null,
      error: null,
    }));
    this.rows.set(rows);
    for (const r of rows) {
      this.send({ type: 'speak', id: r.id, text: r.text });
    }
  }

  protected speakBrowser(): void {
    this.audio.pause();
    const started = performance.now();
    this.browserSeconds.set(null);
    this.speech.speak(this.text(), this.speed(), {
      ended: () => this.browserSeconds.set((performance.now() - started) / 1000),
    });
  }

  protected replay(row: Row): void {
    if (row.url) {
      this.queue = [];
      this.playUrl(row.url);
    }
  }

  private onMessage(data: PiperResponse): void {
    switch (data.type) {
      case 'progress':
        this.downloaded.set({ loaded: data.loaded, total: data.total });
        break;
      case 'loaded':
        this.loadSeconds.set(data.ms / 1000);
        this.warmupSeconds.set(data.warmupMs / 1000);
        this.loadedVoice.set(this.voice());
        this.loadState.set('ready');
        break;
      case 'audio': {
        const url = URL.createObjectURL(new Blob([data.wav], { type: 'audio/wav' }));
        this.patch(data.id, { generated: data.ms / 1000, seconds: data.seconds, url });
        this.queue.push(data.id);
        if (this.audio.paused) {
          this.playNext();
        }
        break;
      }
      case 'error':
        if (data.id !== undefined) {
          this.patch(data.id, { error: data.message });
        } else {
          this.failLoad(data.message);
        }
        break;
    }
  }

  private playNext(): void {
    const id = this.queue.shift();
    const row = this.rows().find((r) => r.id === id);
    if (row?.url) {
      this.playUrl(row.url);
    }
  }

  /** Piper generates at normal speed, so the player changes it (the pitch is kept). */
  private playUrl(url: string): void {
    this.audio.defaultPlaybackRate = this.speed();
    this.audio.src = url;
    this.audio.playbackRate = this.speed();
    void this.audio.play();
  }

  private failLoad(message: string): void {
    this.loadState.set('error');
    this.loadError.set(message);
  }

  private patch(id: number, change: Partial<Row>): void {
    this.rows.update((rows) => rows.map((r) => (r.id === id ? { ...r, ...change } : r)));
  }

  private send(message: PiperRequest): void {
    this.worker?.postMessage(message);
  }
}

/** Splits text into sentences after . ! or ? followed by a space. */
export function splitSentences(text: string): string[] {
  return text
    .split(/(?<=[.!?])\s+/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}
