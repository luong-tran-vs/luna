import { DOCUMENT } from '@angular/common';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { Clip, NaturalVoiceService } from '../natural-voice/natural-voice.service';

import { SpeechService } from './speech.service';

/** Minimal SpeechSynthesisUtterance for jsdom, which has none. */
class FakeUtterance {
  lang = '';
  rate = 1;
  voice: { lang: string } | null = null;
  onstart: (() => void) | null = null;
  onboundary: ((e: { charIndex: number }) => void) | null = null;
  onend: (() => void) | null = null;
  onerror: ((e: { error: string }) => void) | null = null;
  constructor(readonly text: string) {}
}

describe('SpeechService', () => {
  let synth: { speak: ReturnType<typeof vi.fn>; cancel: ReturnType<typeof vi.fn>; getVoices: () => unknown[] };
  let voices: { lang: string; localService: boolean; name: string }[];

  const create = (withSynth = true) => {
    TestBed.configureTestingModule({
      providers: [{ provide: DOCUMENT, useValue: { defaultView: withSynth ? { speechSynthesis: synth } : {} } }],
    });
    return TestBed.inject(SpeechService);
  };
  const spoken = () => synth.speak.mock.calls.at(-1)![0] as FakeUtterance;

  beforeEach(() => {
    voices = [];
    synth = { speak: vi.fn(), cancel: vi.fn(), getVoices: () => voices };
    vi.stubGlobal('SpeechSynthesisUtterance', FakeUtterance);
  });

  afterEach(() => vi.unstubAllGlobals());

  it('reads the text at the given rate with an English voice', () => {
    voices = [
      { lang: 'vi-VN', localService: true, name: 'An' },
      { lang: 'en-GB', localService: true, name: 'Daniel' },
      { lang: 'en-US', localService: false, name: 'Online' },
      { lang: 'en-US', localService: true, name: 'Samantha' },
    ];
    const speech = create();
    expect(speech.supported).toBe(true);
    speech.speak('Hello there.', 0.75);
    const u = spoken();
    expect(u.text).toBe('Hello there.');
    expect(u.rate).toBe(0.75);
    expect((u.voice as unknown as { name: string }).name).toBe('Samantha');
    expect(u.lang).toBe('en-US');
  });

  it('falls back to en-US without English voices', () => {
    voices = [{ lang: 'vi-VN', localService: true, name: 'An' }];
    create().speak('Hi.', 1);
    expect(spoken().voice).toBeNull();
    expect(spoken().lang).toBe('en-US');
  });

  it('prefers a neural voice (Edge "Natural", Chrome "Google") over the plain ones', () => {
    voices = [
      { lang: 'en-US', localService: true, name: 'Microsoft David - English (United States)' },
      { lang: 'en-US', localService: false, name: 'Google US English' },
      { lang: 'en-GB', localService: false, name: 'Microsoft Sonia Online (Natural) - English (United Kingdom)' },
      { lang: 'en-US', localService: false, name: 'Microsoft Aria Online (Natural) - English (United States)' },
    ];
    create().speak('Hi.', 1);
    expect((spoken().voice as unknown as { name: string }).name).toContain('Aria Online (Natural)');
  });

  it('reads with a voice on the device when an online voice fails, and keeps to those', () => {
    voices = [
      { lang: 'en-US', localService: true, name: 'Microsoft David' },
      { lang: 'en-US', localService: false, name: 'Microsoft Aria Online (Natural)' },
    ];
    const failed = vi.fn();
    const speech = create();
    speech.speak('Hi.', 1, { failed });
    spoken().onerror!({ error: 'network' });
    expect(failed).not.toHaveBeenCalled();
    expect((spoken().voice as unknown as { name: string }).name).toBe('Microsoft David');
    expect(spoken().text).toBe('Hi.');

    speech.speak('Again.', 1);
    expect((spoken().voice as unknown as { name: string }).name).toBe('Microsoft David');
  });

  it('reports start, words and end', () => {
    const events: string[] = [];
    create().speak('Hi there.', 1, {
      started: () => events.push('start'),
      boundary: (i) => events.push('word ' + i),
      ended: () => events.push('end'),
    });
    const u = spoken();
    u.onstart!();
    u.onboundary!({ charIndex: 3 });
    u.onend!();
    expect(events).toEqual(['start', 'word 3', 'end']);
  });

  it('reports progress from word events, capped until the end, then 1', () => {
    const shares: number[] = [];
    create().speak('Hello there my friend.', 1, { progress: (share) => shares.push(share) });
    const u = spoken();
    u.onboundary!({ charIndex: 11 }); // half of the text
    u.onboundary!({ charIndex: 3 }); // never goes back
    u.onboundary!({ charIndex: 22 });
    u.onend!();
    expect(shares[0]).toBeCloseTo(0.5);
    expect(shares[1]).toBeCloseTo(0.5);
    expect(shares[2]).toBe(0.95);
    expect(shares.at(-1)).toBe(1);
  });

  it('advances progress with time while reading, and stops when stopped', () => {
    vi.useFakeTimers();
    try {
      const shares: number[] = [];
      const speech = create();
      speech.speak('Hi.', 1, { progress: (share) => shares.push(share) });
      vi.advanceTimersByTime(300);
      expect(shares.length).toBe(3);
      speech.stop();
      vi.advanceTimersByTime(300);
      expect(shares.length).toBe(3);
    } finally {
      vi.useRealTimers();
    }
  });

  it('reports errors, but not those of an utterance stopped or replaced on purpose', () => {
    const speech = create();
    const failed = vi.fn();
    speech.speak('One.', 1, { failed });
    const first = spoken();
    speech.speak('Two.', 1, { failed }); // replaces "One."
    expect(synth.cancel).toHaveBeenCalled();
    first.onerror!({ error: 'interrupted' });
    first.onend!();
    expect(failed).not.toHaveBeenCalled();

    spoken().onerror!({ error: 'synthesis-failed' });
    expect(failed).toHaveBeenCalledWith('synthesis-failed');

    speech.speak('Three.', 1, { failed });
    speech.stop();
    spoken().onerror!({ error: 'canceled' });
    expect(failed).toHaveBeenCalledTimes(1);
  });

  it('is unsupported without speech synthesis and fails at once', () => {
    const speech = create(false);
    const failed = vi.fn();
    expect(speech.supported).toBe(false);
    speech.speak('Hi.', 1, { failed });
    expect(failed).toHaveBeenCalledWith('unsupported');
  });
});

/** Minimal Audio for jsdom (whose play() is not implemented). */
class FakeAudio {
  static last: FakeAudio;
  src = '';
  paused = true;
  currentTime = 0;
  playbackRate = 1;
  defaultPlaybackRate = 1;
  readonly played: { src: string; rate: number }[] = [];
  onplaying: (() => void) | null = null;
  onended: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor() {
    FakeAudio.last = this;
  }
  play(): Promise<void> {
    this.paused = false;
    this.played.push({ src: this.src, rate: this.playbackRate });
    return Promise.resolve();
  }
  pause(): void {
    this.paused = true;
  }
}

describe('SpeechService with the natural voice', () => {
  let synth: { speak: ReturnType<typeof vi.fn>; cancel: ReturnType<typeof vi.fn>; getVoices: () => unknown[] };
  let clips: Map<string, Clip>;
  const voice = {
    ready: signal(true),
    enabled: signal(true),
    exclusive: signal(false),
    cached: (text: string) => clips.get(text) ?? null,
    request: vi.fn(),
    want: vi.fn(),
    prefetch: vi.fn(),
  };

  const create = () => {
    TestBed.configureTestingModule({
      providers: [
        { provide: DOCUMENT, useValue: { defaultView: { speechSynthesis: synth } } },
        { provide: NaturalVoiceService, useValue: voice },
      ],
    });
    return TestBed.inject(SpeechService);
  };

  beforeEach(() => {
    clips = new Map([['Hello.', { url: 'blob:hello', seconds: 2 }]]);
    voice.ready.set(true);
    voice.exclusive.set(false);
    voice.request.mockReset();
    voice.want.mockClear();
    voice.prefetch.mockClear();
    synth = { speak: vi.fn(), cancel: vi.fn(), getVoices: () => [] };
    vi.stubGlobal('SpeechSynthesisUtterance', FakeUtterance);
    vi.stubGlobal('Audio', FakeAudio);
  });

  afterEach(() => vi.unstubAllGlobals());

  it('plays a generated text at once, at the chosen speed, and reports start, progress and end', () => {
    const events: string[] = [];
    const speech = create();
    expect(speech.natural()).toBe(true);
    speech.speak('Hello.', 0.75, {
      started: () => events.push('start'),
      ended: () => events.push('end'),
      progress: (share, seconds) => events.push(`${share} ${seconds.toFixed(2)}`),
    });
    const audio = FakeAudio.last;
    expect(audio.played).toEqual([{ src: 'blob:hello', rate: 0.75 }]);
    expect(speech.playing()).toBe('Hello.');
    audio.onplaying!();
    audio.onended!();
    expect(speech.playing()).toBeNull();
    // 2 s of audio at 0.75× lasts 2.67 s.
    expect(events).toEqual(['start', '1 2.67', 'end']);
    expect(synth.speak).not.toHaveBeenCalled();
    expect(voice.want).not.toHaveBeenCalled();
  });

  it('reads a text not generated yet with the browser voice at once, and asks for it', () => {
    const speech = create();
    speech.speak('Not yet.', 1);
    expect((synth.speak.mock.calls.at(-1)![0] as FakeUtterance).text).toBe('Not yet.');
    expect(speech.playing()).toBe('Not yet.');
    speech.stop();
    expect(speech.playing()).toBeNull();
    expect(voice.want).toHaveBeenCalledWith('Not yet.');
  });

  it('ignores the events of a clip that was stopped', () => {
    const ended = vi.fn();
    const speech = create();
    speech.speak('Hello.', 1, { ended });
    speech.stop();
    FakeAudio.last.onended!();
    expect(ended).not.toHaveBeenCalled();
    expect(FakeAudio.last.paused).toBe(true);
  });

  it('with the browser voice off, waits for the text to be generated and plays it', async () => {
    voice.exclusive.set(true);
    let done!: (clip: Clip | null) => void;
    voice.request.mockReturnValue(new Promise((resolve) => (done = resolve)));
    const speech = create();
    speech.speak(' Not yet. ', 1.25);
    expect(voice.request).toHaveBeenCalledWith(' Not yet. ');
    expect(speech.preparing()).toBe('Not yet.');
    done({ url: 'blob:later', seconds: 1 });
    await Promise.resolve();
    expect(speech.preparing()).toBeNull();
    expect(FakeAudio.last.played).toEqual([{ src: 'blob:later', rate: 1.25 }]);
    expect(synth.speak).not.toHaveBeenCalled();
  });

  it('with the browser voice off, drops a text stopped while waiting and reports one that failed', async () => {
    voice.exclusive.set(true);
    const speech = create();
    voice.request.mockResolvedValue({ url: 'blob:late', seconds: 1 });
    speech.speak('Stopped.', 1);
    speech.stop();
    expect(speech.preparing()).toBeNull();
    await Promise.resolve();
    expect(FakeAudio.last.played).toEqual([]);

    const failed = vi.fn();
    voice.request.mockResolvedValue(null);
    speech.speak('Failed.', 1, { failed });
    await Promise.resolve();
    expect(failed).toHaveBeenCalledWith('natural-voice');
    expect(synth.speak).not.toHaveBeenCalled();
  });

  it('uses the browser voice while the model is not ready, and passes prefetches on', () => {
    voice.ready.set(false);
    const speech = create();
    speech.speak('Hello.', 1);
    expect(synth.speak).toHaveBeenCalled();
    speech.prefetch(['A.']);
    expect(voice.prefetch).toHaveBeenCalledWith(['A.']);
  });
});
