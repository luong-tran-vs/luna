import { InjectionToken } from '@angular/core';

/** Sample rate the speech models expect. */
export const SAMPLE_RATE = 16000;
/** A recording never runs longer than this: the Speaking step reads one short sentence. */
export const MAX_SECONDS = 15;

/** Louder than this (RMS of the samples, 0–1) counts as speech. */
const SPEECH_LEVEL = 0.02;
/** Quiet this long after speaking ends the recording. */
const PAUSE_SECONDS = 1;
/** Nothing said this long: give up. */
const NO_SPEECH_SECONDS = 6;
/** How often the level is measured, in milliseconds. */
const TICK_MS = 50;

export interface Recording {
  /** What the browser recorded (webm or mp4), to listen to again. */
  blob: Blob;
  /** Mono samples at SAMPLE_RATE, for the model. */
  audio: Float32Array;
  seconds: number;
}

/**
 * Decides when to stop listening: after the learner spoke and then stayed quiet for PAUSE_SECONDS,
 * or when nothing was said for NO_SPEECH_SECONDS. Feed it each level with the time since the start.
 */
export class EndOfSpeech {
  private spoke = false;
  private lastLoud = 0;

  /** True when the recording should stop. */
  update(level: number, seconds: number): boolean {
    if (level >= SPEECH_LEVEL) {
      this.spoke = true;
      this.lastLoud = seconds;
    }
    return this.spoke ? seconds - this.lastLoud >= PAUSE_SECONDS : seconds >= NO_SPEECH_SECONDS;
  }
}

/** Root mean square of the samples: how loud they are, 0–1. */
export function rms(samples: Float32Array): number {
  let sum = 0;
  for (const s of samples) {
    sum += s * s;
  }
  return Math.sqrt(sum / Math.max(1, samples.length));
}

/** Whether this page may record: a microphone API and a secure context (HTTPS or localhost). */
export function canRecord(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.isSecureContext &&
    typeof navigator.mediaDevices?.getUserMedia === 'function' &&
    typeof MediaRecorder !== 'undefined'
  );
}

/** Records one utterance; VoiceRecorder in the browser, a fake in specs. */
export interface Recorder {
  record(onTick: (level: number, seconds: number) => void): Promise<Recording>;
  stop(): void;
}

/** Where recorders come from, and whether this page can record at all. */
export const RECORDING = new InjectionToken<{ available: () => boolean; create: () => Recorder }>('RECORDING', {
  providedIn: 'root',
  factory: () => ({ available: canRecord, create: () => new VoiceRecorder() }),
});

/**
 * Records one utterance from the microphone: starts at once, stops by itself after a pause (see
 * EndOfSpeech) or MAX_SECONDS, or when stop() is called. The level and the elapsed time are reported
 * as it goes.
 */
export class VoiceRecorder implements Recorder {
  private stopNow: (() => void) | null = null;

  async record(onTick: (level: number, seconds: number) => void): Promise<Recording> {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
    });
    const context = new AudioContext();
    const analyser = context.createAnalyser();
    analyser.fftSize = 2048;
    context.createMediaStreamSource(stream).connect(analyser);
    const recorder = new MediaRecorder(stream);
    const chunks: Blob[] = [];
    recorder.ondataavailable = (e) => chunks.push(e.data);
    const stopped = new Promise<void>((resolve) => (recorder.onstop = () => resolve()));
    this.stopNow = () => recorder.state !== 'inactive' && recorder.stop();

    recorder.start();
    const started = performance.now();
    const samples = new Float32Array(analyser.fftSize);
    const end = new EndOfSpeech();
    const timer = setInterval(() => {
      analyser.getFloatTimeDomainData(samples);
      const level = rms(samples);
      const seconds = (performance.now() - started) / 1000;
      onTick(level, seconds);
      if (end.update(level, seconds) || seconds >= MAX_SECONDS) {
        this.stop();
      }
    }, TICK_MS);

    try {
      await stopped;
    } finally {
      clearInterval(timer);
      this.stopNow = null;
      stream.getTracks().forEach((t) => t.stop());
      void context.close();
    }
    const blob = new Blob(chunks, { type: recorder.mimeType });
    const audio = await toMono16k(blob);
    return { blob, audio, seconds: audio.length / SAMPLE_RATE };
  }

  stop(): void {
    this.stopNow?.();
  }
}

/** Decodes a recording to mono samples at SAMPLE_RATE (the browser resamples while decoding). */
async function toMono16k(blob: Blob): Promise<Float32Array> {
  const context = new AudioContext({ sampleRate: SAMPLE_RATE });
  try {
    const decoded = await context.decodeAudioData(await blob.arrayBuffer());
    if (decoded.numberOfChannels === 1) {
      return decoded.getChannelData(0);
    }
    const mono = new Float32Array(decoded.length);
    for (let c = 0; c < decoded.numberOfChannels; c++) {
      decoded.getChannelData(c).forEach((s, i) => (mono[i] += s / decoded.numberOfChannels));
    }
    return mono;
  } finally {
    void context.close();
  }
}
