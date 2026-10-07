import { TestBed } from '@angular/core/testing';

import { PIPER_WORKER } from '../piper/create-piper-worker';
import { PiperRequest, PiperResponse } from '../piper/piper.messages';
import { NATURAL_VOICE_ONLY_STORAGE_KEY, NATURAL_VOICE_STORAGE_KEY, NaturalVoiceService } from './natural-voice.service';

/** Stands in for the Piper worker: records what it is asked and answers on demand. */
class FakeWorker {
  readonly posted: PiperRequest[] = [];
  terminated = false;
  private readonly listeners: ((e: { data: PiperResponse }) => void)[] = [];

  postMessage(message: PiperRequest): void {
    this.posted.push(message);
  }

  addEventListener(type: string, listener: (e: { data: PiperResponse }) => void): void {
    if (type === 'message') {
      this.listeners.push(listener);
    }
  }

  terminate(): void {
    this.terminated = true;
  }

  answer(data: PiperResponse): void {
    this.listeners.forEach((l) => l({ data }));
  }

  speaks(): Extract<PiperRequest, { type: 'speak' }>[] {
    return this.posted.filter((m): m is Extract<PiperRequest, { type: 'speak' }> => m.type === 'speak');
  }
}

describe('NaturalVoiceService', () => {
  let workers: FakeWorker[];
  const worker = () => workers[workers.length - 1];

  const create = () => TestBed.inject(NaturalVoiceService);
  const ready = () => {
    const service = create();
    service.enable();
    worker().answer({ type: 'loaded', ms: 10, warmupMs: 5, isolated: true });
    return service;
  };
  const audio = (id: number) => worker().answer({ type: 'audio', id, ms: 100, wav: new ArrayBuffer(8), seconds: 2 });

  beforeEach(() => {
    localStorage.clear();
    workers = [];
    vi.stubGlobal('Worker', class {});
    let n = 0;
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => `blob:${++n}`, revokeObjectURL: vi.fn() }));
    TestBed.configureTestingModule({
      providers: [
        {
          provide: PIPER_WORKER,
          useValue: () => {
            const w = new FakeWorker();
            workers.push(w);
            return w as unknown as Worker;
          },
        },
      ],
    });
  });

  afterEach(() => vi.unstubAllGlobals());

  it('is off by default and starts no worker', () => {
    const service = create();
    expect(service.state()).toBe('off');
    expect(workers.length).toBe(0);
  });

  it('loads the Piper voice when turned on, then is ready', () => {
    const service = create();
    service.enable();
    expect(localStorage.getItem(NATURAL_VOICE_STORAGE_KEY)).toBe('on');
    expect(worker().posted[0]).toEqual({ type: 'load', voice: 'en_US-hfc_female-medium' });
    expect(service.state()).toBe('loading');
    expect(service.starting()).toBe(true);

    worker().answer({ type: 'progress', loaded: 45, total: 90 });
    expect(service.percent()).toBe(50);
    expect(service.starting()).toBe(false);
    worker().answer({ type: 'progress', loaded: 90, total: 90 });
    expect(service.starting()).toBe(true);

    worker().answer({ type: 'loaded', ms: 10, warmupMs: 5, isolated: true });
    expect(service.ready()).toBe(true);
  });

  it('loads again at start-up when it was left on', () => {
    localStorage.setItem(NATURAL_VOICE_STORAGE_KEY, 'on');
    expect(create().state()).toBe('loading');
    expect(worker().posted[0].type).toBe('load');
  });

  it('generates prefetched texts one at a time, in order, and keeps them', () => {
    const service = ready();
    service.prefetch(['One.', 'Two.', ' One. ']);
    expect(worker().speaks().map((m) => m.text)).toEqual(['One.']);
    expect(worker().speaks()[0]).toEqual({ type: 'speak', id: 1, text: 'One.' });
    expect(service.cached('One.')).toBeNull();

    audio(worker().speaks()[0].id);
    expect(service.cached('One.')).toEqual({ url: 'blob:1', seconds: 2 });
    expect(worker().speaks().map((m) => m.text)).toEqual(['One.', 'Two.']);
    audio(worker().speaks()[1].id);

    // Already generated: not asked again.
    service.prefetch(['One.', 'Two.']);
    expect(worker().speaks().length).toBe(2);
  });

  it('generates a text someone asked for before the prefetched ones', () => {
    const service = ready();
    service.prefetch(['A.', 'B.', 'C.']);
    service.want('C.');
    service.want('Z.');
    audio(worker().speaks()[0].id);
    audio(worker().speaks()[1].id);
    audio(worker().speaks()[2].id);
    expect(worker().speaks().map((m) => m.text)).toEqual(['A.', 'Z.', 'C.', 'B.']);
  });

  it('does not keep a text the model fails on, and tries it again when asked', () => {
    const service = ready();
    service.want('Hard.');
    worker().answer({ type: 'error', id: worker().speaks()[0].id, message: 'boom' });
    expect(service.cached('Hard.')).toBeNull();
    service.want('Hard.');
    expect(worker().speaks().length).toBe(2);
  });

  it('ignores texts before the model is ready', () => {
    const service = create();
    service.enable();
    service.prefetch(['Hi.']);
    service.want('Hi.');
    expect(worker().speaks()).toEqual([]);
    expect(service.cached('Hi.')).toBeNull();
  });

  it('remembers that the browser voice is off, and is exclusive while the natural voice is on, even failing', () => {
    const service = create();
    service.setOnly(true);
    expect(localStorage.getItem(NATURAL_VOICE_ONLY_STORAGE_KEY)).toBe('on');
    expect(service.exclusive()).toBe(false);
    service.enable();
    expect(service.exclusive()).toBe(true);
    worker().answer({ type: 'error', message: 'no network' });
    expect(service.exclusive()).toBe(true);
    service.disable();
    expect(service.exclusive()).toBe(false);
    service.setOnly(false);
    expect(localStorage.getItem(NATURAL_VOICE_ONLY_STORAGE_KEY)).toBeNull();
  });

  it('generates a requested text first, even while loading, and resolves when it is ready', async () => {
    const service = create();
    service.enable();
    service.prefetch(['Later.']);
    const clip = service.request('Now.');
    expect(worker().speaks()).toEqual([]);
    worker().answer({ type: 'loaded', ms: 10, warmupMs: 5, isolated: true });
    expect(worker().speaks().map((m) => m.text)).toEqual(['Now.']);
    audio(worker().speaks()[0].id);
    expect(await clip).toEqual({ url: 'blob:1', seconds: 2 });
    expect(await service.request('Now.')).toEqual({ url: 'blob:1', seconds: 2 });
  });

  it('resolves a request with null when the model fails on it or is turned off', async () => {
    const service = ready();
    const hard = service.request('Hard.');
    const other = service.request('Other.');
    worker().answer({ type: 'error', id: worker().speaks()[0].id, message: 'boom' });
    expect(await hard).toBeNull();
    service.disable();
    expect(await other).toBeNull();
    expect(await service.request('Off.')).toBeNull();
  });

  it('reports a model that cannot load, and turns off on request', () => {
    const service = create();
    service.enable();
    worker().answer({ type: 'error', message: 'no network' });
    expect(service.state()).toBe('error');
    expect(service.error()).toBe('no network');
    expect(worker().terminated).toBe(true);

    service.disable();
    expect(service.state()).toBe('off');
    expect(localStorage.getItem(NATURAL_VOICE_STORAGE_KEY)).toBeNull();
  });
});
