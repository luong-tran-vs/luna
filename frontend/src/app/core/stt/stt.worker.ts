/// <reference lib="webworker" />

/*
 * Speech to text in the browser, for the admin page "Thử nhận dạng giọng nói" (trying models for
 * F10, the Speaking step): runs a Moonshine or Whisper model with Transformers.js, off the main
 * thread. Messages in: load (one model), transcribe (16 kHz mono audio). Messages out: progress,
 * loaded (with a warm-up run), text, error. Transformers.js keeps the model files in the browser's
 * Cache Storage, so each model downloads once.
 */

import { SttDevice, SttRequest, SttResponse } from './stt.messages';

/**
 * The self-contained build of Transformers.js (it bundles onnxruntime-web), from the CDN: the npm
 * package pulls in sharp and onnxruntime-node, which only Node needs. A variable, so the bundler
 * leaves the import to the browser.
 */
const TRANSFORMERS_URL = 'https://cdn.jsdelivr.net/npm/@huggingface/transformers@4.3.1/dist/transformers.min.js';

/** The part of a Transformers.js speech pipeline used here. */
type Recognizer = (audio: Float32Array) => Promise<{ text: string } | { text: string }[]>;

interface Transformers {
  env: { allowLocalModels: boolean };
  pipeline(
    task: 'automatic-speech-recognition',
    model: string,
    options: {
      device: SttDevice;
      dtype: string | Record<string, string>;
      progress_callback: (p: { status: string; file?: string; loaded?: number; total?: number }) => void;
    },
  ): Promise<Recognizer>;
}

let recognizer: Recognizer | null = null;
let chain: Promise<void> = Promise.resolve();

function post(message: SttResponse): void {
  postMessage(message);
}

/**
 * Quantized (8-bit) weights on the CPU; on the GPU the encoder stays full precision and the decoder
 * uses 4-bit weights, as the Transformers.js WebGPU examples do.
 */
function dtypeFor(device: SttDevice): string | Record<string, string> {
  return device === 'webgpu' ? { encoder_model: 'fp32', decoder_model_merged: 'q4' } : 'q8';
}

async function load(model: string, device: SttDevice): Promise<void> {
  recognizer = null;
  const url = TRANSFORMERS_URL;
  const { pipeline, env } = (await import(/* @vite-ignore */ url)) as Transformers;
  env.allowLocalModels = false;
  const files = new Map<string, { loaded: number; total: number }>();
  const started = performance.now();
  const created = await pipeline('automatic-speech-recognition', model, {
    device,
    dtype: dtypeFor(device),
    progress_callback: (p) => {
      if (p.status !== 'progress' || !p.file) {
        return;
      }
      files.set(p.file, { loaded: p.loaded ?? 0, total: p.total ?? 0 });
      let loaded = 0;
      let total = 0;
      files.forEach((f) => {
        loaded += f.loaded;
        total += f.total;
      });
      post({ type: 'progress', loaded, total });
    },
  });
  const loadedAt = performance.now();
  await created(new Float32Array(16000));
  recognizer = created;
  post({ type: 'loaded', ms: loadedAt - started, warmupMs: performance.now() - loadedAt });
}

async function transcribe(id: number, audio: Float32Array): Promise<void> {
  if (!recognizer) {
    post({ type: 'error', id, message: 'Chưa tải model.' });
    return;
  }
  const started = performance.now();
  const out = await recognizer(audio);
  const text = (Array.isArray(out) ? out.map((o) => o.text).join(' ') : out.text).trim();
  post({ type: 'text', id, text, ms: performance.now() - started });
}

addEventListener('message', ({ data }: MessageEvent<SttRequest>) => {
  chain = chain.then(async () => {
    try {
      await (data.type === 'load' ? load(data.model, data.device) : transcribe(data.id, data.audio));
    } catch (err) {
      post({ type: 'error', id: data.type === 'transcribe' ? data.id : undefined, message: String(err) });
    }
  });
});
