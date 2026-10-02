import { DOCUMENT } from '@angular/common';
import { inject, Injectable } from '@angular/core';

import { estimateSeconds } from '../../shared/utils/fake-waveform';

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
 * Reads English text aloud with the browser's own voice (Web Speech API), so listening needs no
 * prepared audio files (F3, F4, F5, F17). One utterance at a time: speaking again stops the
 * previous one.
 */
@Injectable({ providedIn: 'root' })
export class SpeechService {
  private readonly synth = inject(DOCUMENT).defaultView?.speechSynthesis ?? null;
  private current: SpeechSynthesisUtterance | null = null;
  private timer: ReturnType<typeof setInterval> | null = null;

  /** False when the browser has no speech synthesis. */
  readonly supported = this.synth !== null && typeof SpeechSynthesisUtterance !== 'undefined';

  speak(text: string, rate: number, handlers: SpeakHandlers = {}): void {
    if (!this.synth || !this.supported) {
      handlers.failed?.('unsupported');
      return;
    }
    this.stop();
    const u = new SpeechSynthesisUtterance(text);
    u.lang = 'en-US';
    u.rate = rate;
    const voice = this.englishVoice();
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
      handlers.failed?.(e.error);
    };
    this.current = u;
    this.synth.speak(u);
  }

  stop(): void {
    this.finish();
    this.synth?.cancel();
  }

  private finish(): void {
    this.current = null;
    if (this.timer !== null) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  /** A local US voice when there is one, else any English voice; null lets the browser choose. */
  private englishVoice(): SpeechSynthesisVoice | null {
    const english = (this.synth?.getVoices() ?? []).filter((v) => v.lang.toLowerCase().startsWith('en'));
    return (
      english.find((v) => v.lang === 'en-US' && v.localService) ??
      english.find((v) => v.lang === 'en-US') ??
      english[0] ??
      null
    );
  }
}
