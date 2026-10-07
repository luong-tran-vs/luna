import { DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';

import { createSttWorker } from '../../../core/stt/create-stt-worker';
import { SttDevice, SttRequest, SttResponse } from '../../../core/stt/stt.messages';
import { SpeechService } from '../../../core/services/speech.service';
import { Comparison, compareDictation } from '../../../shared/utils/dictation-compare';
import { canRecord, MAX_SECONDS, Recording, VoiceRecorder } from '../../../core/stt/voice-recorder';

/** Models to try (8-bit weights on the CPU; sizes are what downloads once). */
const MODELS: readonly { id: string; label: string }[] = [
  { id: 'onnx-community/moonshine-tiny-ONNX', label: 'Moonshine tiny (khoảng 28MB)' },
  { id: 'onnx-community/moonshine-base-ONNX', label: 'Moonshine base (khoảng 62MB)' },
  { id: 'onnx-community/whisper-tiny.en', label: 'Whisper tiny.en (khoảng 41MB)' },
  { id: 'onnx-community/whisper-base.en', label: 'Whisper base.en (khoảng 77MB)' },
];

/** Short sentences like those of a lesson: the Speaking step reads one at a time. */
const SENTENCES: readonly string[] = [
  'I usually drink coffee in the morning.',
  'Could you tell me where the station is?',
  'My sister works at a hospital.',
  'We went to the beach last weekend.',
  'How much does this jacket cost?',
  "I'm looking forward to seeing you.",
];

interface Row {
  id: number;
  model: string;
  device: SttDevice;
  sentence: string;
  /** Length of the recording in seconds. */
  seconds: number;
  url: string;
  /** What the model heard; null while it runs. */
  heard: string | null;
  comparison: Comparison | null;
  /** Recognition time in seconds. */
  took: number | null;
  error: string | null;
}

type LoadState = 'idle' | 'loading' | 'ready' | 'error';

/**
 * Admin tool for F10 (Speaking): records a short sentence and turns it into text with a Moonshine
 * or Whisper model running in the browser (in a worker), to measure on a real phone how long it
 * takes and how well it hears, before choosing the model.
 */
@Component({
  selector: 'lu-stt-lab',
  imports: [DecimalPipe],
  templateUrl: './stt-lab.html',
  styleUrls: ['../tts-lab/tts-lab.css', './stt-lab.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SttLab {
  private readonly speech = inject(SpeechService);
  private readonly recorder = new VoiceRecorder();
  private worker: Worker | null = null;
  private nextId = 1;
  private readonly player = typeof Audio === 'undefined' ? null : new Audio();
  /** The last recording, to run again with another model. */
  private last: Recording | null = null;

  protected readonly models = MODELS;
  protected readonly sentences = SENTENCES;
  protected readonly maxSeconds = MAX_SECONDS;
  protected readonly cores = typeof navigator !== 'undefined' ? navigator.hardwareConcurrency : 0;
  protected readonly memory =
    typeof navigator !== 'undefined' ? ((navigator as { deviceMemory?: number }).deviceMemory ?? null) : null;
  protected readonly isolated = typeof crossOriginIsolated !== 'undefined' && crossOriginIsolated;
  protected readonly webgpu = typeof navigator !== 'undefined' && 'gpu' in navigator;
  protected readonly micAllowed = canRecord();

  protected readonly model = signal(MODELS[0].id);
  protected readonly device = signal<SttDevice>('wasm');
  protected readonly sentence = signal(SENTENCES[0]);

  protected readonly loadState = signal<LoadState>('idle');
  protected readonly loadError = signal<string | null>(null);
  protected readonly downloaded = signal({ loaded: 0, total: 0 });
  protected readonly loadSeconds = signal<number | null>(null);
  protected readonly warmupSeconds = signal<number | null>(null);
  /** The model loaded and where it runs. */
  protected readonly loaded = signal<{ model: string; device: SttDevice } | null>(null);

  protected readonly recording = signal(false);
  protected readonly level = signal(0);
  protected readonly elapsed = signal(0);
  protected readonly recordError = signal<string | null>(null);
  protected readonly hasRecording = signal(false);

  protected readonly rows = signal<Row[]>([]);
  protected readonly running = computed(() => this.rows().some((r) => r.heard === null && r.error === null));

  protected readonly downloadPercent = computed(() => {
    const { loaded, total } = this.downloaded();
    return total > 0 ? Math.min(100, (loaded / total) * 100) : 0;
  });
  /** Level bar width: speech is quiet in RMS terms, so it is scaled up. */
  protected readonly levelPercent = computed(() => Math.min(100, this.level() * 400));

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      this.recorder.stop();
      this.worker?.terminate();
      this.player?.pause();
      this.speech.stop();
      this.rows().forEach((r) => URL.revokeObjectURL(r.url));
    });
  }

  protected modelLabel(id: string): string {
    return MODELS.find((m) => m.id === id)?.label.replace(/ \(.*\)$/, '') ?? id;
  }

  /** Starts a fresh worker and loads the chosen model. */
  protected load(): void {
    this.worker?.terminate();
    this.worker = createSttWorker();
    this.worker.addEventListener('message', ({ data }: MessageEvent<SttResponse>) => this.onMessage(data));
    this.worker.addEventListener('error', (e) => this.failLoad(e.message || 'Worker lỗi.'));
    this.loaded.set(null);
    this.loadState.set('loading');
    this.loadError.set(null);
    this.loadSeconds.set(null);
    this.warmupSeconds.set(null);
    this.downloaded.set({ loaded: 0, total: 0 });
    this.send({ type: 'load', model: this.model(), device: this.device() });
  }

  protected nextSentence(): void {
    const i = SENTENCES.indexOf(this.sentence());
    this.sentence.set(SENTENCES[(i + 1) % SENTENCES.length]);
  }

  protected listen(): void {
    this.speech.speak(this.sentence(), 1);
  }

  /** Records one try; it is recognized at once when a model is loaded. */
  protected async record(): Promise<void> {
    if (this.recording()) {
      this.recorder.stop();
      return;
    }
    this.speech.stop();
    this.recordError.set(null);
    this.recording.set(true);
    try {
      const rec = await this.recorder.record((level, seconds) => {
        this.level.set(level);
        this.elapsed.set(seconds);
      });
      this.last = rec;
      this.hasRecording.set(true);
      this.recognize();
    } catch (err) {
      this.recordError.set(micError(err));
    } finally {
      this.recording.set(false);
      this.level.set(0);
    }
  }

  /** Recognizes the last recording with the model loaded (to compare models on the same audio). */
  protected recognize(): void {
    const rec = this.last;
    const loaded = this.loaded();
    if (!rec || !loaded || this.loadState() !== 'ready') {
      return;
    }
    const id = this.nextId++;
    this.rows.update((rows) => [
      {
        id,
        model: loaded.model,
        device: loaded.device,
        sentence: this.sentence(),
        seconds: rec.seconds,
        // Each row keeps its own URL, so revoking one never breaks another.
        url: URL.createObjectURL(rec.blob),
        heard: null,
        comparison: null,
        took: null,
        error: null,
      },
      ...rows,
    ]);
    // A copy: the worker takes the buffer, and the recording may be recognized again.
    const audio = rec.audio.slice();
    this.worker?.postMessage({ type: 'transcribe', id, audio } satisfies SttRequest, [audio.buffer]);
  }

  protected play(row: Row): void {
    if (this.player) {
      this.player.src = row.url;
      void this.player.play();
    }
  }

  private onMessage(data: SttResponse): void {
    switch (data.type) {
      case 'progress':
        this.downloaded.set({ loaded: data.loaded, total: data.total });
        break;
      case 'loaded':
        this.loadSeconds.set(data.ms / 1000);
        this.warmupSeconds.set(data.warmupMs / 1000);
        this.loaded.set({ model: this.model(), device: this.device() });
        this.loadState.set('ready');
        break;
      case 'text':
        this.rows.update((rows) =>
          rows.map((r) =>
            r.id === data.id
              ? { ...r, heard: data.text, took: data.ms / 1000, comparison: compareDictation(r.sentence, data.text) }
              : r,
          ),
        );
        break;
      case 'error':
        if (data.id === undefined) {
          this.failLoad(data.message);
        } else {
          this.rows.update((rows) => rows.map((r) => (r.id === data.id ? { ...r, error: data.message } : r)));
        }
        break;
    }
  }

  private failLoad(message: string): void {
    this.loadState.set('error');
    this.loadError.set(message);
  }

  private send(message: SttRequest): void {
    this.worker?.postMessage(message);
  }
}

/** A message for a microphone that could not start. */
export function micError(err: unknown): string {
  const name = err instanceof DOMException ? err.name : '';
  if (name === 'NotAllowedError' || name === 'SecurityError') {
    return 'Chưa cho phép dùng micro. Hãy cho phép trong cài đặt trang của trình duyệt rồi thử lại.';
  }
  if (name === 'NotFoundError') {
    return 'Không tìm thấy micro trên máy này.';
  }
  return `Không ghi âm được: ${String(err)}`;
}
