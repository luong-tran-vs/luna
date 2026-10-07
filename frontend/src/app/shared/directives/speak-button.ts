import { computed, DestroyRef, Directive, ElementRef, inject, input } from '@angular/core';

import { SpeechService } from '../../core/services/speech.service';

/**
 * Shows on a play button what happens to the text it reads (styles/motion.css):
 * - preparing: waiting for the natural voice (browser voice turned off), a spinner;
 * - speaking: being read aloud, a stop square; pressing the button then stops the reading
 *   instead of reading again.
 * Give it the text the button reads. Left empty, it follows any text and only shows preparing,
 * for buttons that already show their own playing state.
 */
@Directive({
  selector: '[luSpeak]',
  host: {
    '[class.preparing]': 'preparing()',
    '[class.speaking]': 'speaking()',
    '[attr.aria-busy]': 'preparing() || null',
    '[attr.aria-pressed]': 'speaking() || null',
    '[attr.title]': "preparing() ? 'Đang chuẩn bị giọng đọc…' : speaking() ? 'Dừng đọc' : null",
  },
})
export class SpeakButton {
  private readonly speech = inject(SpeechService);

  /** The text the button reads; empty: any text, preparing only. */
  readonly luSpeak = input<string | null | undefined>('');

  private readonly text = computed(() => this.luSpeak()?.trim() ?? '');
  protected readonly preparing = computed(() => {
    const waiting = this.speech.preparing();
    return waiting !== null && (!this.text() || waiting === this.text());
  });
  protected readonly speaking = computed(() => !!this.text() && this.speech.playing() === this.text());

  constructor() {
    const button = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    // Captured, so it runs before the button's own (click) and can keep it from reading again.
    const onClick = (event: Event) => {
      if (this.speaking() || (this.text() && this.preparing())) {
        event.stopImmediatePropagation();
        this.speech.stop();
      }
    };
    button.addEventListener('click', onClick, { capture: true });
    inject(DestroyRef).onDestroy(() => button.removeEventListener('click', onClick, { capture: true }));
  }
}
