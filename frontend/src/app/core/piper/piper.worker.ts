/// <reference lib="webworker" />

/*
 * Runs a Piper voice (VITS, about 60MB) in the browser with @mintplex-labs/piper-tts-web, off the
 * main thread, for the natural voice (NaturalVoiceService) and the TTS lab. Messages in: load (one
 * voice), speak (one text). Messages out: progress, loaded (with a warm-up run), audio (a WAV buffer
 * with timings), error. Texts are generated one at a time in the order they arrive. The library
 * keeps the voice in the origin private file system, so it downloads once; where it cannot write
 * there, model-cache.ts keeps it in IndexedDB instead.
 */

import { TtsSession } from '@mintplex-labs/piper-tts-web';

import { cachingFetch, indexedDbStore, isVoiceFile, libraryCanStore } from './model-cache';
import { PiperRequest, PiperResponse } from './piper.messages';

if (!libraryCanStore() && typeof indexedDB !== 'undefined') {
  self.fetch = cachingFetch(self.fetch.bind(self), indexedDbStore(), isVoiceFile);
}

/**
 * The ONNX runtime files must match the onnxruntime-web the library imports (1.30.0, pinned in
 * package.json: change both together); the library's default points at an older version. The
 * phonemizer keeps the library's default.
 */
const WASM_PATHS = {
  ...TtsSession.WASM_LOCATIONS,
  onnxWasm: 'https://cdn.jsdelivr.net/npm/onnxruntime-web@1.30.0/dist/',
};

let session: TtsSession | null = null;
let chain: Promise<void> = Promise.resolve();

function post(message: PiperResponse, transfer: Transferable[] = []): void {
  postMessage(message, transfer);
}

/** Length in seconds of a PCM WAV file, from its header. */
function wavSeconds(wav: ArrayBuffer): number {
  const view = new DataView(wav);
  const sampleRate = view.getUint32(24, true);
  const channels = view.getUint16(22, true);
  const bits = view.getUint16(34, true);
  return (wav.byteLength - 44) / (sampleRate * channels * (bits / 8));
}

async function load(voice: string): Promise<void> {
  // The library keeps one session and would reuse it (with the old model) for another voice.
  (TtsSession as unknown as { _instance: TtsSession | null })._instance = null;
  session = null;
  const files = new Map<string, { loaded: number; total: number }>();
  const started = performance.now();
  const created = await TtsSession.create({
    voiceId: voice,
    wasmPaths: WASM_PATHS,
    progress: (p) => {
      if (p.url.startsWith('tts://')) {
        return; // progress of a long text being generated, not of a download
      }
      files.set(p.url, { loaded: p.loaded, total: p.total });
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
  await created.predict('Hello.');
  session = created;
  post({ type: 'loaded', ms: loadedAt - started, warmupMs: performance.now() - loadedAt, isolated: self.crossOriginIsolated });
}

async function speak(id: number, text: string): Promise<void> {
  if (!session) {
    post({ type: 'error', id, message: 'Chưa tải giọng.' });
    return;
  }
  const started = performance.now();
  const wav = await (await session.predict(text)).arrayBuffer();
  post({ type: 'audio', id, ms: performance.now() - started, wav, seconds: wavSeconds(wav) }, [wav]);
}

addEventListener('message', ({ data }: MessageEvent<PiperRequest>) => {
  chain = chain.then(async () => {
    try {
      await (data.type === 'load' ? load(data.voice) : speak(data.id, data.text));
    } catch (err) {
      post({ type: 'error', id: data.type === 'speak' ? data.id : undefined, message: String(err) });
    }
  });
});
