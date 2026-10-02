import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  model,
  signal,
} from '@angular/core';

import { PracticeDialogue } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { Waveform } from '../../../../shared/components/waveform/waveform';
import { clock } from '../practice-logic';

const SPEEDS = [0.75, 1, 1.25];

type Mode = 'idle' | 'all' | 'one';

/**
 * Step 2: the sample dialogue, read by the browser's voice. "Play all" reads the turns one after
 * another (a turn the browser fails to read is skipped); each turn can also be heard alone. The
 * words start hidden so the learner listens first; each turn shows an illustrative waveform filled
 * as it is read.
 */
@Component({
  selector: 'lu-dialogue-step',
  imports: [Icon, Waveform],
  templateUrl: './dialogue-step.html',
  styleUrls: ['../practice.css', './dialogue-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DialogueStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  private readonly speech = inject(SpeechService);

  readonly dialogue = input.required<PracticeDialogue>();
  /** Reading speed, shared with the page. */
  readonly rate = model(1);

  protected readonly canSpeak = this.speech.supported;
  protected readonly speeds = SPEEDS;
  protected readonly mode = signal<Mode>('idle');
  /** Turn being read, or null. */
  protected readonly current = signal<number | null>(null);
  protected readonly showMeaning = signal(true);
  /** Turns whose words are shown; the words start hidden so the learner listens first. */
  protected readonly revealed = signal<ReadonlySet<number>>(new Set());
  protected readonly allRevealed = computed(() =>
    this.dialogue().turns.every((_, i) => this.isRevealed(i)),
  );
  /** Seconds of the turns already read in "play all", and of the current one. */
  private readonly done = signal(0);
  private readonly now = signal(0);
  /** Share of the current turn already read. */
  private readonly share = signal(0);

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

  /** Without a voice nothing can be heard, so every turn shows its words. */
  protected isRevealed(index: number): boolean {
    return !this.canSpeak || this.revealed().has(index);
  }

  protected toggleTurn(index: number): void {
    this.revealed.update((set) => {
      const next = new Set(set);
      if (!next.delete(index)) {
        next.add(index);
      }
      return next;
    });
  }

  protected toggleAll(): void {
    this.revealed.set(
      this.allRevealed() ? new Set() : new Set(this.dialogue().turns.map((_, i) => i)),
    );
  }

  /**
   * Part of a turn's waveform to fill: the read share of the current turn, and full for turns
   * already read in "play all".
   */
  protected progress(index: number): number {
    const current = this.current();
    if (current === null) {
      return 0;
    }
    if (index < current && this.mode() === 'all') {
      return 1;
    }
    return index === current ? this.share() : 0;
  }

  protected playAll(): void {
    this.mode.set('all');
    this.done.set(0);
    this.readFrom(0);
  }

  protected playOne(index: number): void {
    this.mode.set('one');
    this.read(index);
  }

  protected stop(): void {
    this.speech.stop();
    this.reset();
  }

  protected setRate(rate: number): void {
    this.rate.set(rate);
  }

  protected onSpeedKeydown(event: KeyboardEvent): void {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const i = (SPEEDS.indexOf(this.rate()) + step + SPEEDS.length) % SPEEDS.length;
    this.rate.set(SPEEDS[i]);
    const group = (event.currentTarget as HTMLElement).closest('[role="radiogroup"]');
    group?.querySelectorAll<HTMLElement>('[role="radio"]')[i]?.focus();
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
