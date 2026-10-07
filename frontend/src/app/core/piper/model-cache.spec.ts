import { cachingFetch, FileStore, isVoiceFile } from './model-cache';

const MODEL = 'https://huggingface.co/diffusionstudio/piper-voices/resolve/main/en/en_US/hfc_female/medium/en_US-hfc_female-medium.onnx';

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
