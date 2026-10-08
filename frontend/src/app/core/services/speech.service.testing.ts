import { signal } from '@angular/core';

import { SpeakHandlers, SpeechService } from './speech.service';

/** A SpeechService for specs: records what the page asks the browser to read. */
export class FakeSpeech implements Pick<SpeechService, 'supported' | 'speak' | 'stop' | 'prefetch' | 'natural' | 'preparing' | 'playing'> {
  supported = true;
  /** Set to true to act as if the natural voice were ready. */
  readonly natural = signal(false);
  /** Set to a text to act as if it were waiting for the natural voice. */
  readonly preparing = signal<string | null>(null);
  /** Set to a text to act as if it were being read aloud. */
  readonly playing = signal<string | null>(null);
  /** Every prefetch() call. */
  readonly prefetched: (readonly string[])[] = [];
  readonly spoken: { text: string; rate: number; handlers: SpeakHandlers; voice: number }[] = [];
  /** Number of stop() calls. */
  stops = 0;

  speak(text: string, rate: number, handlers: SpeakHandlers = {}, voice = 0): void {
    this.spoken.push({ text, rate, handlers, voice });
  }

  stop(): void {
    this.stops++;
    this.playing.set(null);
  }

  prefetch(texts: readonly string[]): void {
    this.prefetched.push(texts);
  }

  last(): { text: string; rate: number; handlers: SpeakHandlers; voice: number } {
    return this.spoken[this.spoken.length - 1];
  }

  texts(): string[] {
    return this.spoken.map((s) => s.text);
  }
}

/** Provider replacing SpeechService with `fake`. */
export function provideFakeSpeech(fake: FakeSpeech) {
  return { provide: SpeechService, useValue: fake };
}
