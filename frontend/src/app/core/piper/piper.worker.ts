/// <reference lib="webworker" />

/*
 * Runs a Piper voice (VITS, about 60MB) in the browser with @mintplex-labs/piper-tts-web, off the
 * main thread, for the natural voice (NaturalVoiceService) and the TTS lab. Messages in: load (one
 * voice), speak (one text). Messages out: progress, loaded (with a warm-up run), audio (a WAV buffer
 * with timings), error. Texts are generated one at a time in the order they arrive. The library
 * keeps the voice in the origin private file system, so it downloads once; where it cannot write
 * there, model-cache.ts keeps it in IndexedDB instead.
 */

import { HF_BASE, PATH_MAP, remove, stored, TtsSession, VoiceId } from '@mintplex-labs/piper-tts-web';

import {
  cachingFetch,
  indexedDbStore,
  inMemoryUrl,
  isPhonemizerFile,
  isVoiceFile,
  libraryCanStore,
} from './model-cache';
import { PiperRequest, PiperResponse } from './piper.messages';

const canKeep = typeof indexedDB !== 'undefined';
if (!libraryCanStore() && canKeep) {
  self.fetch = cachingFetch(self.fetch.bind(self), indexedDbStore(), isVoiceFile);
}

/** The phonemizer's files are kept in IndexedDB too: the library itself never stores them. */
const phonemizerFetch = canKeep ? cachingFetch(self.fetch.bind(self), indexedDbStore(), isPhonemizerFile) : self.fetch.bind(self);

/**
 * The phonemizer's files, downloaded once per worker and then read from memory (added 2026-10-20):
 * the library built a new phonemizer for every sentence, and each fetched piper_phonemize.data and
 * .wasm again, so a lesson read aloud downloaded them over and over. The library is also patched
 * (frontend/patches/, applied by patch-package on npm install) to keep one phonemizer per session.
 */
let phonemizer: Promise<{ piperData: string; piperWasm: string }> | null = null;
function phonemizerPaths(): Promise<{ piperData: string; piperWasm: string }> {
  return (phonemizer ??= Promise.all([
    inMemoryUrl(phonemizerFetch, TtsSession.WASM_LOCATIONS.piperData, 'application/octet-stream'),
    inMemoryUrl(phonemizerFetch, TtsSession.WASM_LOCATIONS.piperWasm, 'application/wasm'),
  ]).then(([piperData, piperWasm]) => ({ piperData, piperWasm })));
}

/**
 * The ONNX runtime files must match the onnxruntime-web the library imports (1.30.0, pinned in
 * package.json: change both together); the library's default points at an older version. The
 * phonemizer's files are the library's default, read from memory (phonemizerPaths).
 */
const WASM_PATHS = {
  ...TtsSession.WASM_LOCATIONS,
  onnxWasm: 'https://cdn.jsdelivr.net/npm/onnxruntime-web@1.30.0/dist/',
};

/**
 * Voices no longer used (2026-10-20): their licences allow non-commercial or research use only
 * (hfc_female, ryan, lessac) or are unknown (amy). Their files are deleted from the browser so no
 * copy stays on the learner's device.
 */
const RETIRED_VOICES: readonly VoiceId[] = [
  'en_US-hfc_female-medium',
  'en_US-ryan-medium',
  'en_US-lessac-medium',
  'en_US-amy-medium',
];

/** Deletes the files of RETIRED_VOICES from the library's store and from IndexedDB. Best effort. */
async function forgetRetiredVoices(): Promise<void> {
  try {
    if (libraryCanStore()) {
      const kept = await stored();
      await Promise.all(RETIRED_VOICES.filter((id) => kept.includes(id)).map((id) => remove(id)));
    }
    if (canKeep) {
      const store = indexedDbStore();
      const urls = RETIRED_VOICES.flatMap((id) => [`${HF_BASE}/${PATH_MAP[id]}`, `${HF_BASE}/${PATH_MAP[id]}.json`]);
      await Promise.all(urls.map((url) => store.remove?.(url)));
    }
  } catch {
    // Left for the next start: the files only take space.
  }
}

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
    wasmPaths: { ...WASM_PATHS, ...(await phonemizerPaths()) },
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
  void forgetRetiredVoices();
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
