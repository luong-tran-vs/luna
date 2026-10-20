import { cachingFetch, FileStore, inMemoryUrl, isPhonemizerFile, isVoiceFile } from './model-cache';

const MODEL = 'https://huggingface.co/diffusionstudio/piper-voices/resolve/main/en/en_US/ljspeech/medium/en_US-ljspeech-medium.onnx';

/** A FileStore in memory. */
function memoryStore(): FileStore & { files: Map<string, Blob> } {
  const files = new Map<string, Blob>();
  return {
    files,
    read: async (url) => files.get(url),
    write: async (url, blob) => {
      files.set(url, blob);
    },
  };
}

describe('model cache', () => {
  it('keeps only the voice model and its config', () => {
    expect(isVoiceFile(MODEL)).toBe(true);
    expect(isVoiceFile(`${MODEL}.json`)).toBe(true);
    expect(isVoiceFile('https://cdn.jsdelivr.net/npm/onnxruntime-web@1.30.0/dist/ort-wasm.wasm')).toBe(false);
  });

  it('downloads a voice file once, streaming it to the caller, then answers from the store', async () => {
    const store = memoryStore();
    const base = vi.fn(async () => new Response('model-bytes', { headers: { 'Content-Type': 'application/octet-stream' } }));
    const fetcher = cachingFetch(base as unknown as typeof fetch, store, isVoiceFile);

    expect(await (await fetcher(MODEL)).text()).toBe('model-bytes');
    await vi.waitFor(() => expect(store.files.has(MODEL)).toBe(true));

    const again = await fetcher(MODEL);
    expect(await again.text()).toBe('model-bytes');
    expect(again.headers.get('Content-Length')).toBe('11');
    expect(base).toHaveBeenCalledTimes(1);
  });

  it('passes other requests and failed downloads through without keeping them', async () => {
    const store = memoryStore();
    const base = vi.fn(async () => new Response('nope', { status: 404 }));
    const fetcher = cachingFetch(base as unknown as typeof fetch, store, isVoiceFile);

    expect((await fetcher(MODEL)).status).toBe(404);
    await fetcher('https://example.com/a.wasm');
    expect(base).toHaveBeenCalledTimes(2);
    expect(store.files.size).toBe(0);
  });
});

// Added 2026-10-20: the phonemizer's files are downloaded once and read from memory.
describe('phonemizer files', () => {
  const DATA = 'https://cdn.jsdelivr.net/npm/@diffusionstudio/piper-wasm@1.0.0/build/piper_phonemize.data';

  it('tells the phonemizer files apart', () => {
    expect(isPhonemizerFile(DATA)).toBe(true);
    expect(isPhonemizerFile(DATA.replace('.data', '.wasm'))).toBe(true);
    expect(isPhonemizerFile(MODEL)).toBe(false);
  });

  it('gives a blob: URL of the file, downloaded once', async () => {
    const get = vi.fn(async () => new Response(new Uint8Array([1, 2, 3])));
    const made: Blob[] = [];
    const makeUrl = (blob: Blob) => {
      made.push(blob);
      return 'blob:luna/1';
    };
    expect(await inMemoryUrl(get as unknown as typeof fetch, DATA, 'application/octet-stream', makeUrl)).toBe('blob:luna/1');
    expect(get).toHaveBeenCalledTimes(1);
    expect(made.map((b) => [b.size, b.type])).toEqual([[3, 'application/octet-stream']]);
  });

  it('keeps the original URL when the download fails', async () => {
    const failing = vi.fn(async () => new Response('x', { status: 404 }));
    expect(await inMemoryUrl(failing as unknown as typeof fetch, DATA, 'x')).toBe(DATA);
    const offline = vi.fn(async () => {
      throw new TypeError('offline');
    });
    expect(await inMemoryUrl(offline as unknown as typeof fetch, DATA, 'x')).toBe(DATA);
  });
});
