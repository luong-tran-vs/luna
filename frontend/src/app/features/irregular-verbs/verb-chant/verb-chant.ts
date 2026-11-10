import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  signal,
} from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spokenForm, threeForms } from '../verb-study.logic';

type Speed = 'slow' | 'mid' | 'fast';

const RATES: Record<Speed, number> = { slow: 0.7, mid: 0.85, fast: 1 };
/** Pause after each word, in ms; "Đọc theo" leaves room to say it aloud. */
const GAP_MS = 250;
const ECHO_MS = 1300;

/**
 * "Đọc theo nhịp" (design/de-xuat-dong-tu.png part 7, added 2026-11-10): the verbs of a group read
 * V1 – V2 – V3 one line after another on a steady beat, the line and the form being read lit. The
 * learner picks the speed, may have each line read twice, and with "Đọc theo" gets a pause after
 * each word to say it aloud.
 */
@Component({
  selector: 'lu-verb-chant',
  imports: [Icon],
  templateUrl: './verb-chant.html',
  styleUrl: './verb-chant.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbChant {
  private readonly speech = inject(SpeechService);

  readonly verbs = input.required<readonly IrregularVerb[]>();

  protected readonly canSpeak = this.speech.supported;
  protected readonly speeds: readonly { value: Speed; label: string }[] = [
    { value: 'slow', label: 'Chậm' },
    { value: 'mid', label: 'Vừa' },
    { value: 'fast', label: 'Nhanh' },
  ];
  protected readonly speed = signal<Speed>('mid');
  protected readonly twice = signal(false);
  protected readonly echo = signal(false);
  protected readonly playing = signal(false);
  /** The line and the form (0–2) being read; -1 when none. */
  protected readonly line = signal(-1);
  protected readonly beat = signal(-1);

  protected readonly lines = computed(() =>
    this.verbs().map((v) => ({ verb: v, forms: threeForms(v) })),
  );
  protected readonly current = computed(() => this.lines()[this.line()] ?? this.lines()[0] ?? null);

  private run = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stop());
  }

  protected toggleTwice(): void {
    this.twice.update((t) => !t);
  }

  protected toggleEcho(): void {
    this.echo.update((e) => !e);
  }

  /** Starts from the line lit (or the first), or goes on from where it was paused. */
  protected play(): void {
    this.stop();
    const run = this.run;
    this.playing.set(true);
    const steps: { line: number; beat: number }[] = [];
    const repeats = this.twice() ? 2 : 1;
    this.lines().forEach((_, l) => {
      for (let r = 0; r < repeats; r++) {
        for (let b = 0; b < 3; b++) {
          steps.push({ line: l, beat: b });
        }
      }
    });
    const startAt = Math.max(
      0,
      steps.findIndex((s) => s.line === Math.max(0, this.line())),
    );
    const rate = RATES[this.speed()];
    const gap = this.echo() ? ECHO_MS / rate : GAP_MS;
    const next = (i: number) => {
      if (run !== this.run) {
        return;
      }
      if (i >= steps.length) {
        this.playing.set(false);
        this.line.set(-1);
        this.beat.set(-1);
        return;
      }
      const s = steps[i];
      const v = this.lines()[s.line];
      this.line.set(s.line);
      this.beat.set(s.beat);
      this.speech.speak(spokenForm(v.verb.base, v.forms[s.beat], s.beat), rate, {
        ended: () => (this.timer = setTimeout(() => next(i + 1), gap)),
        failed: () => this.pause(),
      });
    };
    next(startAt);
  }

  /** Stops reading; the line lit stays, so "Đọc" goes on from it. */
  protected pause(): void {
    this.stop();
    this.beat.set(-1);
  }

  protected pick(i: number): void {
    this.stop();
    this.line.set(i);
    this.beat.set(-1);
  }

  private stop(): void {
    this.run++;
    clearTimeout(this.timer);
    this.playing.set(false);
  }
}
