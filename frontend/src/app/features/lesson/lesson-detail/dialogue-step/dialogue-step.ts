import { NgTemplateOutlet } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  model,
  output,
  signal,
} from '@angular/core';

import { PracticeDialogue } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';
import { clock } from '../practice-logic';

const SPEEDS = [0.75, 1, 1.25];

type Mode = 'idle' | 'all' | 'one';

/**
 * The sample dialogue as a chat (updated 2026-10-14): every turn on screen, one bubble each, the two
 * speakers on either side. Tapping a bubble reads that turn; one button reads them all one after
 * another (a turn the browser fails to read is skipped), the bubble being read marked and filled as
 * it goes. The words start hidden so the learner listens first; two switches show the words and
 * the meanings. One speed button cycles 0.75x → 1x → 1.25x.
 */
@Component({
  selector: 'lu-dialogue-step',
  imports: [Icon, NgTemplateOutlet, SpeakButton],
  templateUrl: './dialogue-step.html',
  styleUrls: ['../practice.css', './dialogue-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DialogueStep {
  /** Position of this step on the page, shown in its title; 0 for none (Luyện thêm). */
  readonly number = input(1);
  private readonly speech = inject(SpeechService);

  readonly dialogue = input.required<PracticeDialogue>();
  /** Reading speed, shared with the page. */
  readonly rate = model(1);
  /** The button under the dialogue, e.g. "Sang bước Viết" or "Xong, về bài học". */
  readonly finishLabel = input('Hoàn thành');
  /** The finishing button: the page moves on. */
  readonly finished = output<void>();

  protected readonly canSpeak = this.speech.supported;
  protected readonly mode = signal<Mode>('idle');
  /** Turn being read, or null. */
  protected readonly current = signal<number | null>(null);
  protected readonly showMeaning = signal(true);
  /** Whether the words are shown; hidden at first so the learner listens first. */
  protected readonly showWords = signal(false);
  /** Without a voice nothing can be heard, so the words are always shown. */
  protected readonly wordsShown = computed(() => !this.canSpeak || this.showWords());
  /** Seconds of the turns already read in "play all", and of the current one. */
  private readonly done = signal(0);
  private readonly now = signal(0);
  /** Share of the current turn already read (0–1), filling its bubble. */
  protected readonly share = signal(0);

  protected readonly elapsed = computed(() =>
    clock(this.mode() === 'all' ? this.done() + this.now() : 0),
  );

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      if (this.mode() !== 'idle') {
        this.speech.stop();
      }
    });
  }

  protected speaker(index: number): string {
    return this.dialogue().speakers[index] ?? '';
  }

  /** The speed button: 0.75x → 1x → 1.25x → 0.75x. */
  protected cycleRate(): void {
    const i = SPEEDS.indexOf(this.rate());
    this.rate.set(SPEEDS[(i + 1) % SPEEDS.length]);
  }

  protected playAll(): void {
    this.mode.set('all');
    this.done.set(0);
    this.readFrom(0);
  }

  /** A bubble: stops that turn when it is being read, else reads it alone. */
  protected toggleOne(index: number): void {
    if (this.current() === index) {
      this.stop();
    } else {
      this.mode.set('one');
      this.read(index);
    }
  }

  protected stop(): void {
    this.speech.stop();
    this.reset();
  }

  protected finish(): void {
    if (this.mode() !== 'idle') {
      this.stop();
    }
    this.finished.emit();
  }

  /** Reads turn `index` on; stops after the last one. */
  private readFrom(index: number): void {
    if (index >= this.dialogue().turns.length) {
      this.reset();
      return;
    }
    this.read(index);
  }

  private read(index: number): void {
    const turn = this.dialogue().turns[index];
    if (!turn) {
      return;
    }
    this.current.set(index);
    this.now.set(0);
    this.share.set(0);
    this.speech.speak(turn.text, this.rate(), {
      progress: (share, seconds) => {
        this.share.set(share);
        this.now.set(seconds);
      },
      ended: () => this.onEnd(true),
      failed: () => this.onEnd(false),
    });
  }

  private onEnd(read: boolean): void {
    const mode = this.mode();
    if (mode === 'all') {
      if (read) {
        this.done.update((d) => d + this.now());
      }
      this.readFrom((this.current() ?? -1) + 1);
    } else if (mode === 'one') {
      this.reset();
    }
  }

  private reset(): void {
    this.mode.set('idle');
    this.current.set(null);
    this.done.set(0);
    this.now.set(0);
    this.share.set(0);
  }
}
