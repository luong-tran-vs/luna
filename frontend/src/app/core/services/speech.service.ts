import { DOCUMENT } from '@angular/common';
import { inject, Injectable, signal } from '@angular/core';

import { estimateSeconds } from '../../shared/utils/fake-waveform';
import { Clip, NaturalVoiceService } from '../natural-voice/natural-voice.service';

/** What a caller hears about one utterance; errors from stopping it on purpose are not reported. */
export interface SpeakHandlers {
  started?: () => void;
  /** The browser reached the word starting at charIndex (not every browser reports this). */
  boundary?: (charIndex: number) => void;
  ended?: () => void;
  failed?: (error: string) => void;
  /**
   * Share already read (0–1) and seconds elapsed, about every PROGRESS_MS: estimated from the text
   * length and the rate, moved ahead by word events, capped below 1 until the end.
   */
  progress?: (share: number, seconds: number) => void;
}

/** How often progress is reported, in milliseconds. */
const PROGRESS_MS = 100;
/** Progress never reaches 1 before the browser says the reading ended. */
const PROGRESS_CAP = 0.95;

/**
 * Reads English text aloud, so listening needs no prepared audio files (F3, F4, F5, F17), and
 * always starts at once. With the natural voice on (NaturalVoiceService), a text it has already
 * generated plays in that voice at the chosen speed; any other text is read by the browser's own
 * voice (Web Speech API) while the natural voice generates it for next time — unless the learner
 * turned the browser voice off, then the text plays once generated. One utterance at a
 * time: speaking again stops the previous one.
 */
@Injectable({ providedIn: 'root' })
export class SpeechService {
  private readonly synth = inject(DOCUMENT).defaultView?.speechSynthesis ?? null;
  private readonly voice = inject(NaturalVoiceService);
  private readonly audio: HTMLAudioElement | null = typeof Audio === 'undefined' ? null : new Audio();
  private current: SpeechSynthesisUtterance | null = null;
  private timer: ReturnType<typeof setInterval> | null = null;
  /** Bumped by every speak and stop, so events of a clip that was stopped are ignored. */
  private token = 0;
  /** An online browser voice failed (no network): use voices on the device only. */
  private onlineFailed = false;

  private readonly browserSupported = this.synth !== null && typeof SpeechSynthesisUtterance !== 'undefined';

  private readonly _preparing = signal<string | null>(null);
  /**
   * The text (trimmed) waiting for the natural voice before it plays, while the browser voice is
   * off; null otherwise. Play buttons show it as "đang chuẩn bị" (see SpeakButton).
   */
  readonly preparing = this._preparing.asReadonly();
  private readonly _playing = signal<string | null>(null);
  /** The text (trimmed) being read aloud, by either voice; null when silent. */
  readonly playing = this._playing.asReadonly();

  /** True while the natural voice is loaded (texts can be generated ahead). */
  readonly natural = this.voice.ready;

  /** False when the browser has no speech synthesis and the natural voice is off. */
  readonly supported = this.browserSupported || this.voice.enabled();

  speak(text: string, rate: number, handlers: SpeakHandlers = {}): void {
    const clip = this.audio && this.voice.ready() ? this.voice.cached(text) : null;
    if (clip) {
      this.stop();
      this.play(text, clip, rate, handlers);
      return;
    }
    if (this.audio && this.voice.exclusive()) {
      // The browser voice is off: wait for the natural voice to generate the text, then play it.
      this.stop();
      const token = this.token;
      this._preparing.set(text.trim());
      void this.voice.request(text).then((ready) => {
        if (token !== this.token) {
          return; // stopped or replaced meanwhile
        }
        this._preparing.set(null);
        if (ready) {
          this.play(text, ready, rate, handlers);
        } else {
          handlers.failed?.('natural-voice');
        }
      });
      return;
    }
    this.voice.want(text);
    this.speakBrowser(text, rate, handlers);
  }

  /** Texts likely to be heard soon: the natural voice generates them ahead (no-op when it is off). */
  prefetch(texts: readonly string[]): void {
    this.voice.prefetch(texts);
  }

  /** Plays a generated clip; the speed changes the playback rate (the pitch is kept). */
  private play(text: string, clip: Clip, rate: number, handlers: SpeakHandlers): void {
    const audio = this.audio!;
    this._playing.set(text.trim());
    const token = this.token;
    const own = () => token === this.token;
    audio.onplaying = () => own() && handlers.started?.();
    audio.onended = () => {
      if (own()) {
        this.finish();
        handlers.progress?.(1, clip.seconds / rate);
        handlers.ended?.();
      }
    };
    audio.onerror = () => {
      if (own()) {
        this.finish();
        handlers.failed?.('audio');
      }
    };
    if (handlers.progress) {
      this.timer = setInterval(
        () =>
          handlers.progress?.(
            Math.min(PROGRESS_CAP, audio.currentTime / Math.max(0.1, clip.seconds)),
            audio.currentTime / rate,
          ),
        PROGRESS_MS,
      );
    }
    // Loading a new source resets playbackRate to defaultPlaybackRate, so set both.
    audio.defaultPlaybackRate = rate;
    audio.src = clip.url;
    audio.playbackRate = rate;
    audio.play().catch((e: unknown) => {
      if (own()) {
        this.finish();
        handlers.failed?.(String(e));
      }
    });
  }

  /** localOnly: skip the online voices (one of them just failed). */
  private speakBrowser(text: string, rate: number, handlers: SpeakHandlers, localOnly = this.onlineFailed): void {
    if (!this.synth || !this.browserSupported) {
      handlers.failed?.('unsupported');
      return;
    }
    this.stop();
    const u = new SpeechSynthesisUtterance(text);
    u.lang = 'en-US';
    u.rate = rate;
    const voice = this.englishVoice(localOnly);
    if (voice) {
      u.voice = voice;
      u.lang = voice.lang;
    }
    // Progress: estimated time, moved ahead by word events.
    const length = Math.max(0.1, estimateSeconds(text) / rate);
    let start = performance.now();
    let share = 0;
    const report = (next: number) => {
      share = Math.max(share, Math.min(PROGRESS_CAP, next));
      handlers.progress?.(share, (performance.now() - start) / 1000);
    };
    if (handlers.progress) {
      this.timer = setInterval(() => report((performance.now() - start) / 1000 / length), PROGRESS_MS);
    }

    u.onstart = () => {
      start = performance.now();
      handlers.started?.();
    };
    u.onboundary = (e) => {
      report(e.charIndex / Math.max(1, text.length));
      handlers.boundary?.(e.charIndex);
    };
    u.onend = () => {
      if (this.current === u) {
        this.finish();
        handlers.progress?.(1, (performance.now() - start) / 1000);
        handlers.ended?.();
      }
    };
    u.onerror = (e) => {
      if (this.current !== u) {
        return; // stopped or replaced on purpose
      }
      this.finish();
      // An online voice (Edge "Natural", Chrome "Google") needs the network: read with a voice on
      // the device instead, and keep to those for the rest of the visit.
      if (voice && !voice.localService && e.error !== 'interrupted' && e.error !== 'canceled') {
        this.onlineFailed = true;
        this.speakBrowser(text, rate, handlers, true);
        return;
      }
      handlers.failed?.(e.error);
    };
    this.current = u;
    this._playing.set(text.trim());
    this.synth.speak(u);
  }

  stop(): void {
    this.token++;
    this._preparing.set(null);
    this.finish();
    if (this.audio && !this.audio.paused) {
      this.audio.pause();
    }
    this.synth?.cancel();
  }

  private finish(): void {
    this.current = null;
    this._playing.set(null);
    if (this.timer !== null) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  /** The best English voice of the browser (see voiceScore); null lets the browser choose. */
  private englishVoice(localOnly: boolean): SpeechSynthesisVoice | null {
    let best: SpeechSynthesisVoice | null = null;
    let bestScore = -1;
    for (const v of this.synth?.getVoices() ?? []) {
      if (!v.lang.toLowerCase().startsWith('en') || (localOnly && !v.localService)) {
        continue;
      }
      const score = voiceScore(v);
      if (score > bestScore) {
        best = v;
        bestScore = score;
      }
    }
    return best;
  }
}


/**
 * Ranks an English browser voice: the neural ones first (Edge "Microsoft … Online (Natural)",
 * Chrome "Google US English", Apple "Enhanced"/"Premium"), then US English, then voices on the
 * device. The neural ones sound far less robotic; the online ones need the network.
 */
export function voiceScore(v: Pick<SpeechSynthesisVoice, 'name' | 'lang' | 'localService'>): number {
  const name = v.name.toLowerCase();
  let score = 0;
  if (/natural|neural/.test(name)) {
    score += 8;
  }
  if (name.includes('google') || /enhanced|premium/.test(name)) {
    score += 4;
  }
  if (v.lang.replace('_', '-') === 'en-US') {
    score += 2;
  }
  if (v.localService) {
    score += 1;
  }
  return score;
}
