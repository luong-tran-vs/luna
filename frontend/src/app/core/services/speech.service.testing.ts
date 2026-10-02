import { SpeakHandlers, SpeechService } from './speech.service';

/** A SpeechService for specs: records what the page asks the browser to read. */
export class FakeSpeech implements Pick<SpeechService, 'supported' | 'speak' | 'stop'> {
  supported = true;
  readonly spoken: { text: string; rate: number; handlers: SpeakHandlers }[] = [];
  /** Number of stop() calls. */
  stops = 0;

  speak(text: string, rate: number, handlers: SpeakHandlers = {}): void {
    this.spoken.push({ text, rate, handlers });
  }

  stop(): void {
    this.stops++;
  }

  last(): { text: string; rate: number; handlers: SpeakHandlers } {
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
