/*
 * Keeps the Piper voice files in IndexedDB, for browsers where the library's own store (the origin
 * private file system) cannot be written: a page opened over plain http on the network (no secure
 * context, e.g. a phone on http://<LAN IP>) or a browser without FileSystemFileHandle.createWritable.
 * Without it the 60MB voice would download again on every visit there. Used by piper.worker.ts.
 */

/** Where a file is kept between visits. */
export interface FileStore {
  read(url: string): Promise<Blob | undefined>;
  write(url: string, blob: Blob): Promise<void>;
}

/** The voice files: the model (.onnx) and its config (.onnx.json). */
export function isVoiceFile(url: string): boolean {
  return url.startsWith('https://huggingface.co/') && /\.onnx(\.json)?$/.test(url);
}

/** True when the library can keep its files itself (then this cache stays out of the way). */
export function libraryCanStore(): boolean {
  return (
    self.isSecureContext &&
    typeof navigator.storage?.getDirectory === 'function' &&
    typeof FileSystemFileHandle !== 'undefined' &&
    'createWritable' in FileSystemFileHandle.prototype
  );
}

/**
 * A fetch that answers the files `keep` selects from `store`, and stores them the first time they
 * download. A download still streams, so the caller sees its progress.
 */
export function cachingFetch(
  base: typeof fetch,
  store: FileStore,
  keep: (url: string) => boolean,
): typeof fetch {
  return async (input, init) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
    if (!keep(url)) {
      return base(input, init);
    }
    const kept = await store.read(url).catch(() => undefined);
    if (kept) {
      return new Response(kept, {
        headers: { 'Content-Type': kept.type || 'application/octet-stream', 'Content-Length': String(kept.size) },
      });
    }
    const res = await base(input, init);
    if (!res.ok || !res.body) {
      return res;
    }
    const [forCaller, forStore] = res.body.tee();
    // Not awaited: the caller reads its copy meanwhile. A failed write only means downloading again.
    // The blob takes its type from the response's Content-Type.
    void new Response(forStore, { headers: res.headers })
      .blob()
      .then((blob) => store.write(url, blob))
      .catch(() => undefined);
    return new Response(forCaller, { status: res.status, statusText: res.statusText, headers: res.headers });
  };
}

const DB_NAME = 'luna-piper';
const STORE = 'files';

/** The files kept in IndexedDB (database luna-piper, keyed by URL). */
export function indexedDbStore(): FileStore {
  let db: Promise<IDBDatabase> | null = null;
  const open = () =>
    (db ??= new Promise((resolve, reject) => {
      const req = indexedDB.open(DB_NAME, 1);
      req.onupgradeneeded = () => req.result.createObjectStore(STORE);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    }));
  const run = <T>(mode: IDBTransactionMode, action: (s: IDBObjectStore) => IDBRequest<T>) =>
    open().then(
      (d) =>
        new Promise<T>((resolve, reject) => {
          const req = action(d.transaction(STORE, mode).objectStore(STORE));
          req.onsuccess = () => resolve(req.result);
          req.onerror = () => reject(req.error);
        }),
    );
  return {
    read: (url) => run<Blob | undefined>('readonly', (s) => s.get(url)),
    write: (url, blob) => run('readwrite', (s) => s.put(blob, url)).then(() => undefined),
  };
}
