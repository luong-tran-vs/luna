import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  input,
  model,
  signal,
} from '@angular/core';

import { PracticeDialogue } from '../../../../core/models/practice';
import { AudioPlayer } from '../../../../shared/components/audio-player/audio-player';
import { Icon } from '../../../../shared/components/icon/icon';
import { clock } from '../practice-logic';

const SPEEDS = [0.75, 1, 1.25];

type Mode = 'idle' | 'all' | 'one';

/**
 * Step 2: the sample dialogue. "Play all" reads the turns one after another through the page's
 * player (turns without audio, or failing to play, are skipped); each turn can also be heard alone.
 */
@Component({
  selector: 'lu-dialogue-step',
  imports: [Icon],
  templateUrl: './dialogue-step.html',
  styleUrls: ['../practice.css', './dialogue-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DialogueStep {
  readonly dialogue = input.required<PracticeDialogue>();
  /** The page's single audio player. */
  readonly player = input.required<AudioPlayer>();
  /** Playback speed, shared with the player. */
  readonly rate = model(1);

  protected readonly speeds = SPEEDS;
  protected readonly mode = signal<Mode>('idle');
  /** Turn being played, or null. */
  protected readonly current = signal<number | null>(null);
  protected readonly showMeaning = signal(true);
  /** Seconds of the turns already played in "play all", and the position in the current one. */
  private readonly done = signal(0);
  private readonly now = signal(0);

  protected readonly elapsed = computed(() =>
    clock(this.mode() === 'all' ? this.done() + this.now() : 0),
  );

  constructor() {
    effect((onCleanup) => {
      const p = this.player();
      const subs = [
        p.finished.subscribe(() => this.onEnd(true)),
        p.failed.subscribe(() => this.onEnd(false)),
        p.timeChange.subscribe((t) => this.now.set(t)),
      ];
      onCleanup(() => {
        subs.forEach((s) => s.unsubscribe());
        if (this.mode() !== 'idle') {
          p.stop();
        }
      });
    });
  }

  protected speaker(index: number): string {
    return this.dialogue().speakers[index] ?? '';
  }

  protected playAll(): void {
    this.mode.set('all');
    this.done.set(0);
    this.playFrom(0);
  }

  protected playOne(index: number): void {
    const url = this.dialogue().turns[index]?.audioUrl;
    if (!url) {
      return;
    }
    this.mode.set('one');
    this.current.set(index);
    this.now.set(0);
    this.player().replay(url);
  }

  protected stop(): void {
    this.player().stop();
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

  /** Plays the first turn with audio from `index` on; stops after the last one. */
  private playFrom(index: number): void {
    const turns = this.dialogue().turns;
    let i = index;
    while (i < turns.length && !turns[i].audioUrl) {
      i++;
    }
    const url = turns[i]?.audioUrl;
    if (!url) {
      this.reset();
      return;
    }
    this.current.set(i);
    this.now.set(0);
    this.player().replay(url);
  }

  private onEnd(played: boolean): void {
    const mode = this.mode();
    if (mode === 'all') {
      if (played) {
        this.done.update((d) => d + this.now());
      }
      this.playFrom((this.current() ?? -1) + 1);
    } else if (mode === 'one') {
      this.reset();
    }
  }

  private reset(): void {
    this.mode.set('idle');
    this.current.set(null);
    this.done.set(0);
    this.now.set(0);
  }
}
