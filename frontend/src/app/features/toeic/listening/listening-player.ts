import { computed, signal } from '@angular/core';

import { SpeakHandlers } from '../../../core/services/speech.service';
import { groupStart, Segment, segmentStart } from './listening-script.logic';

/** What the player needs from SpeechService (a fake in specs). */
export interface ListeningSpeech {
  speakAs(text: string, rate: number, handlers: SpeakHandlers, opts: { lang: string; voice: number }): void;
  stop(): void;
}

/** A timer the player uses for the pauses (setTimeout; a fake in specs). */
export interface ListeningClock {
  set(fn: () => void, ms: number): unknown;
  clear(handle: unknown): void;
}

const realClock: ListeningClock = {
  set: (fn, ms) => setTimeout(fn, ms),
  clear: (h) => clearTimeout(h as ReturnType<typeof setTimeout>),
};

/** Speeds the learner may choose while practising. */
export const RATES = [0.75, 1, 1.25] as const;

/**
 * Plays a listening script as on the test (FR-018): each segment read with its voice, then its
 * pause, then the next one, on its own. While practising the learner may pause (the segment is read
 * again from its start on resume), hear the group again, change the speed, and jump to a group.
 */
export class ListeningPlayer {
  /** The segment being read or waited after; -1 before the start. */
  readonly index = signal(-1);
  readonly playing = signal(false);
  readonly finished = signal(false);
  readonly rate = signal(1);
  /** The last speech error ("unsupported", "not-allowed"...), "" when fine. */
  readonly error = signal('');

  readonly segment = computed<Segment | null>(() => this.script[this.index()] ?? null);
  /** The group being played, null on the directions or before the start. */
  readonly group = computed(() => this.segment()?.group ?? null);

  private timer: unknown = null;
  /** Bumped on every start and stop, so a late callback of a stopped segment is ignored. */
  private token = 0;

  constructor(
    readonly script: readonly Segment[],
    private readonly speech: ListeningSpeech,
    private readonly clock: ListeningClock = realClock,
  ) {}

  /** Starts (or goes on) at index: the current segment from its start, else the beginning. */
  play(index = Math.max(0, this.index())): void {
    this.halt();
    if (index >= this.script.length) {
      this.finish();
      return;
    }
    this.finished.set(false);
    this.playing.set(true);
    this.error.set('');
    this.run(index);
  }

  /** Stops where it is; resume() reads the current segment again from its start. */
  pause(): void {
    this.halt();
    this.playing.set(false);
  }

  resume(): void {
    this.play(Math.max(0, this.index()));
  }

  /** Hears the current group (or directions) again from its start. */
  replayGroup(): void {
    this.play(segmentStart(this.script, Math.max(0, this.index())));
  }

  /** Goes to the start of a group; plays on if it was playing. */
  jumpToGroup(key: string): void {
    const at = groupStart(this.script, key);
    if (at < 0 || this.group() === key) {
      return;
    }
    if (this.playing()) {
      this.play(at);
    } else {
      this.index.set(at);
    }
  }

  /** Skips the directions being read: on to the next segment. */
  skipDirections(): void {
    const s = this.segment();
    if (s?.kind === 'directions') {
      this.play(this.index() + 1);
    }
  }

  setRate(rate: number): void {
    this.rate.set(rate);
  }

  /** Puts the player at index without playing (resuming an attempt). */
  seek(index: number): void {
    this.halt();
    this.index.set(Math.min(Math.max(0, index), this.script.length - 1));
  }

  destroy(): void {
    this.halt();
    this.playing.set(false);
  }

  private run(i: number): void {
    const token = ++this.token;
    this.index.set(i);
    const s = this.script[i];
    const next = () => {
      if (token !== this.token) {
        return;
      }
      this.timer = this.clock.set(() => {
        if (token === this.token) {
          this.timer = null;
          if (i + 1 >= this.script.length) {
            this.finish();
          } else {
            this.run(i + 1);
          }
        }
      }, s.pauseAfterMs);
    };
    this.speech.speakAs(
      s.text,
      this.rate(),
      {
        ended: next,
        failed: (e) => {
          if (token !== this.token) {
            return;
          }
          if (e === 'unsupported' || e === 'not-allowed') {
            // Nothing can be heard (no voice, or the browser wants a tap first): stop and say so.
            this.error.set(e);
            this.halt();
            this.playing.set(false);
            return;
          }
          next();
        },
      },
      { lang: s.lang, voice: s.voice },
    );
  }

  private halt(): void {
    this.token++;
    if (this.timer !== null) {
      this.clock.clear(this.timer);
      this.timer = null;
    }
    this.speech.stop();
  }

  private finish(): void {
    this.halt();
    this.playing.set(false);
    this.finished.set(true);
  }
}
