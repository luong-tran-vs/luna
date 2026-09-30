import { ChangeDetectionStrategy, Component, effect, ElementRef, input, output, viewChild } from '@angular/core';

/**
 * Plays one audio file at a chosen speed (F4). Changing the source stops the previous audio,
 * so two sentences never play over each other.
 */
@Component({
  selector: 'lu-audio-player',
  template: `<audio #audio hidden preload="none" (ended)="finished.emit()"></audio>`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AudioPlayer {
  readonly src = input<string | null>(null);
  readonly rate = input(1);

  readonly started = output<void>();
  readonly finished = output<void>();
  readonly failed = output<void>();

  private readonly audio = viewChild.required<ElementRef<HTMLAudioElement>>('audio');

  constructor() {
    effect(() => {
      // A new source stops the previous audio, unless play(src) already switched to it.
      const src = this.src();
      const audio = this.audio().nativeElement;
      if ((audio.getAttribute('src') ?? null) !== src) {
        audio.pause();
      }
    });
    effect(() => {
      // Browsers reset playbackRate when the source changes; play() applies it again.
      this.audio().nativeElement.playbackRate = this.rate();
    });
  }

  /** Plays from the current position; `src` overrides the input (used right after changing it). */
  play(src: string | null = this.src()): void {
    if (!src) {
      return;
    }
    const audio = this.audio().nativeElement;
    if (audio.getAttribute('src') !== src) {
      audio.src = src;
    }
    audio.playbackRate = this.rate();
    audio.play().then(
      () => this.started.emit(),
      () => this.failed.emit(),
    );
  }

  replay(src: string | null = this.src()): void {
    try {
      this.audio().nativeElement.currentTime = 0;
    } catch {
      // some environments cannot seek before metadata loads; playing again is enough
    }
    this.play(src);
  }

  stop(): void {
    this.audio().nativeElement.pause();
  }
}
