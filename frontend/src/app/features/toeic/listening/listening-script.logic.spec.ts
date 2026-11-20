import { PaperGroup } from '../../../core/models/toeic';
import { SpeakHandlers } from '../../../core/services/speech.service';
import { ListeningClock, ListeningPlayer, ListeningSpeech } from './listening-player';
import { accentLang, buildScript, groupStart, PAUSE, scriptSeconds, segmentStart } from './listening-script.logic';

function q(id: string, number: number, prompt?: string, options = ['a', 'b', 'c', 'd']) {
  return { id, no: 1, number, prompt, options };
}

const p1: PaperGroup = { key: 'p1', part: 1, number: 1, imageUrl: null, graphic: null, questions: [q('p1#1', 1)] };
const p2: PaperGroup = {
  key: 'p2',
  part: 2,
  number: 7,
  imageUrl: null,
  graphic: null,
  script: [{ speaker: 'w', accent: 'gb', text: 'When is the shipment arriving?' }],
  questions: [q('p2#1', 7, 'When is the shipment arriving?', ['By Thursday.', 'Yes.', 'To the office.'])],
};
const p3: PaperGroup = {
  key: 'p3',
  part: 3,
  number: 32,
  imageUrl: null,
  graphic: { kind: 'table', columns: ['A'], rows: [['1']] },
  script: [
    { speaker: 'm', accent: 'us', text: 'Hi there.' },
    { speaker: 'w', accent: 'au', text: 'Hello.' },
    { speaker: 'm2', accent: 'ca', text: 'Morning.' },
  ],
  questions: [
    { id: 'p3#1', no: 1, number: 32, prompt: 'Where are they?', options: ['a', 'b', 'c', 'd'] },
    { id: 'p3#2', no: 2, number: 33, prompt: 'Who is she?', options: ['a', 'b', 'c', 'd'] },
    { id: 'p3#3', no: 3, number: 34, prompt: 'What next?', options: ['a', 'b', 'c', 'd'] },
  ],
};
const p5: PaperGroup = { key: 'p5', part: 5, number: 101, imageUrl: null, graphic: null, questions: [q('p5#1', 101, 'x ___')] };

describe('buildScript', () => {
  const script = buildScript([p1, p2, p3, p5]);

  it('puts the directions first in each part and leaves out the reading', () => {
    expect(script.filter((s) => s.kind === 'directions').map((s) => s.part)).toEqual([1, 2, 3]);
    expect(script.some((s) => s.group === 'p5')).toBe(false);
  });

  it('reads Part 1: number, the four statements, then 5 seconds', () => {
    const part1 = script.filter((s) => s.group === 'p1');
    expect(part1[0].text).toBe('Number 1. Look at the picture marked number 1 in your test book.');
    expect(part1.slice(1).map((s) => s.text)).toEqual(['(A) a', '(B) b', '(C) c', '(D) d']);
    expect(part1.at(-1)?.pauseAfterMs).toBe(PAUSE.afterItem12);
  });

  it('reads Part 2: the question in its accent, the three responses by someone else', () => {
    const part2 = script.filter((s) => s.group === 'p2');
    expect(part2.map((s) => s.kind)).toEqual(['lead', 'turn', 'option', 'option', 'option']);
    expect(part2[1].lang).toBe('en-GB');
    expect(part2[2].lang).toBe('en-US');
  });

  it('reads Part 3 with each speaker in their accent and a pause after each question', () => {
    const part3 = script.filter((s) => s.group === 'p3');
    expect(part3[0].text).toBe('Questions 32 through 34 refer to the following conversation with three speakers and graphic.');
    const turns = part3.filter((s) => s.kind === 'turn');
    expect(turns.map((t) => t.lang)).toEqual(['en-US', 'en-AU', 'en-CA']);
    expect(turns.map((t) => t.voice)).toEqual([0, 1, 2]);
    const qs = part3.filter((s) => s.kind === 'question');
    expect(qs.map((s) => s.text)).toEqual(['Number 32. Where are they?', 'Number 33. Who is she?', 'Number 34. What next?']);
    expect(qs.every((s) => s.pauseAfterMs === PAUSE.afterQuestion34)).toBe(true);
  });

  it('can leave the directions out (practice)', () => {
    expect(buildScript([p1, p2], { directions: false }).some((s) => s.kind === 'directions')).toBe(false);
  });

  it('finds the start of a group and of a segment', () => {
    const at = groupStart(script, 'p3');
    expect(script[at].kind).toBe('lead');
    expect(segmentStart(script, at + 3)).toBe(at);
    expect(segmentStart(script, 0)).toBe(0);
    expect(groupStart(script, 'nope')).toBe(-1);
  });

  it('estimates the time, shorter when faster', () => {
    expect(scriptSeconds(script, 1.25)).toBeLessThan(scriptSeconds(script, 1));
    expect(scriptSeconds(script)).toBeGreaterThan(20);
  });

  it('maps accents to voice languages', () => {
    expect(['us', 'gb', 'au', 'ca', undefined].map((a) => accentLang(a))).toEqual(['en-US', 'en-GB', 'en-AU', 'en-CA', 'en-US']);
  });
});

class FakeSpeech implements ListeningSpeech {
  calls: { text: string; rate: number; handlers: SpeakHandlers; lang: string; voice: number }[] = [];
  stops = 0;
  speakAs(text: string, rate: number, handlers: SpeakHandlers, opts: { lang: string; voice: number }): void {
    this.calls.push({ text, rate, handlers, ...opts });
  }
  stop(): void {
    this.stops++;
  }
  end(): void {
    this.calls.at(-1)?.handlers.ended?.();
  }
}

class FakeClock implements ListeningClock {
  pending: { fn: () => void; ms: number; id: number }[] = [];
  private id = 0;
  set(fn: () => void, ms: number): unknown {
    const id = ++this.id;
    this.pending.push({ fn, ms, id });
    return id;
  }
  clear(handle: unknown): void {
    this.pending = this.pending.filter((p) => p.id !== handle);
  }
  fire(): void {
    const p = this.pending.shift();
    p?.fn();
  }
}

describe('ListeningPlayer', () => {
  const script = buildScript([p1, p2, p3], { directions: false });
  let speech: FakeSpeech;
  let clock: FakeClock;
  let player: ListeningPlayer;

  beforeEach(() => {
    speech = new FakeSpeech();
    clock = new FakeClock();
    player = new ListeningPlayer(script, speech, clock);
  });

  it('reads each segment, waits its pause, then goes on by itself', () => {
    player.play();
    expect(speech.calls[0].text).toContain('Number 1');
    speech.end();
    expect(clock.pending[0].ms).toBe(script[0].pauseAfterMs);
    clock.fire();
    expect(player.index()).toBe(1);
    expect(speech.calls[1].text).toBe('(A) a');
  });

  it('pauses, then reads the segment again from its start', () => {
    player.play();
    speech.end();
    clock.fire();
    player.pause();
    expect(player.playing()).toBe(false);
    expect(clock.pending.length).toBe(0);
    player.resume();
    expect(speech.calls.at(-1)?.text).toBe('(A) a');
    expect(player.playing()).toBe(true);
  });

  it('hears the group again and jumps to another group', () => {
    const p3at = groupStart(script, 'p3');
    player.play(p3at + 2);
    player.replayGroup();
    expect(player.index()).toBe(p3at);
    player.jumpToGroup('p2');
    expect(player.group()).toBe('p2');
    expect(speech.calls.at(-1)?.text).toBe('Number 7.');
  });

  it('reads at the chosen speed and in each voice', () => {
    player.setRate(1.25);
    player.play(groupStart(script, 'p3') + 2);
    expect(speech.calls.at(-1)).toEqual(expect.objectContaining({ rate: 1.25, lang: 'en-AU', voice: 1 }));
  });

  it('ignores the end of a segment it stopped, and stops when nothing can be heard', () => {
    player.play();
    const old = speech.calls[0];
    player.play(3);
    old.handlers.ended?.();
    expect(clock.pending.length).toBe(0);
    speech.calls.at(-1)?.handlers.failed?.('not-allowed');
    expect(player.playing()).toBe(false);
    expect(player.error()).toBe('not-allowed');
  });

  it('finishes after the last segment', () => {
    player.play(script.length - 1);
    speech.end();
    clock.fire();
    expect(player.finished()).toBe(true);
    expect(player.playing()).toBe(false);
  });
});
