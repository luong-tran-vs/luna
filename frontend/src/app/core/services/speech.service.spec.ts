import { DOCUMENT } from '@angular/common';
import { TestBed } from '@angular/core/testing';

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
