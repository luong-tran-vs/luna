import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  output,
  signal,
} from '@angular/core';

import { PracticeDialogue } from '../../../../../core/models/practice';
import { SpeechService } from '../../../../../core/services/speech.service';
import { SpeechRecognizerService } from '../../../../../core/stt/speech-recognizer.service';
import { MAX_SECONDS, RECORDING } from '../../../../../core/stt/voice-recorder';
import { Icon } from '../../../../../shared/components/icon/icon';
import { recordError } from '../../speak-step/speak-step';
import {
  compareAnswer,
  hintText,
  linesOf,
  RoleAnswer,
  summarize,
  Verdict,
  verdictOf,
  wordCount,
} from './roleplay-logic';

type Phase = 'idle' | 'recording' | 'scoring';

const VERDICT_LABELS: Record<Verdict, string> = {
  ok: 'Chính xác!',
  close: 'Gần đúng',
  bad: 'Chưa đúng',
};

/**
 * Role play in the sample dialogue (added 2026-10-20, step 1 of the role play, no AI): the learner
 * picks a speaker, the app reads the other speaker's lines, and at each of their own lines the
 * learner sees its Vietnamese and says it in English (or types it). The answer is compared with the
 * line like Luyện nói; Gợi ý shows the line one word at a time. The speech recogniser (about 60MB)
 * is downloaded only when the learner asks for the microphone; typing works without it.
 */
@Component({
  selector: 'lu-roleplay',
  imports: [Icon],
  templateUrl: './roleplay.html',
  styleUrls: ['../../practice.css', '../dialogue-step.css', './roleplay.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Roleplay {
  readonly dialogue = input.required<PracticeDialogue>();
  /** Reading speed of the other speaker's lines. */
  readonly rate = input(1);
  /** Thoát: back to listening to the dialogue. */
  readonly closed = output<void>();

  private readonly speech = inject(SpeechService);
  protected readonly recognizer = inject(SpeechRecognizerService);
  private readonly recording = inject(RECORDING);
  private readonly recorder = this.recording.create();

  protected readonly canSpeak = this.speech.supported;
  /** False without a microphone API, over plain http or without the recogniser: typing only. */
  protected readonly micAllowed = this.recording.available() && this.recognizer.supported;
  protected readonly maxSeconds = MAX_SECONDS;

  /** The learner's speaker, or null while choosing. */
  protected readonly role = signal<number | null>(null);
  /** The turn being played; turns before it are done, the dialogue's length when it is over. */
  protected readonly step = signal(0);
  private readonly answers = signal<ReadonlyMap<number, RoleAnswer>>(new Map());
  /** Words of the current line shown by Gợi ý. */
  protected readonly hints = signal(0);
  protected readonly typed = signal('');
  protected readonly phase = signal<Phase>('idle');
  protected readonly elapsed = signal(0);
  protected readonly error = signal<string | null>(null);
  /** The other speaker's turn being read, or null. */
  protected readonly reading = signal<number | null>(null);

  protected readonly over = computed(() => this.step() >= this.dialogue().turns.length);
  /** Turns on screen: the ones done and the current one. */
  protected readonly shown = computed(() =>
    this.dialogue()
      .turns.slice(0, this.step() + 1)
      .map((turn, index) => ({ turn, index })),
  );
  protected readonly myTurn = computed(() => {
    const t = this.dialogue().turns[this.step()];
    return t !== undefined && t.speaker === this.role();
  });
  protected readonly line = computed(() => this.dialogue().turns[this.step()]?.text ?? '');
  protected readonly answer = computed(() => this.answers().get(this.step()) ?? null);
  protected readonly canHint = computed(() => this.hints() < wordCount(this.line()));
  protected readonly myLines = computed(() => {
    const r = this.role();
    return r === null ? [] : linesOf(this.dialogue(), r);
  });
  /** "Lượt 2/4" of the learner's lines. */
  protected readonly lineNumber = computed(() => this.myLines().indexOf(this.step()) + 1);
  protected readonly summary = computed(() => summarize(this.myLines(), this.answers()));
  protected readonly canRecordNow = computed(
    () => this.micAllowed && this.recognizer.state() === 'ready' && this.phase() === 'idle',
  );
  protected readonly micStatus = computed(() => {
    switch (this.phase()) {
      case 'recording':
        return 'Đang ghi âm...';
      case 'scoring':
        return 'Đang chấm...';
      default:
        break;
    }
    switch (this.recognizer.state()) {
      case 'loading':
        return this.recognizer.starting()
          ? 'Đang khởi động bộ nhận giọng nói...'
          : `Đang tải bộ nhận giọng nói... ${this.recognizer.percent()}%`;
      case 'error':
        return 'Chưa tải được bộ nhận giọng nói. Bạn vẫn gõ được câu trả lời.';
      default:
        return this.answer() ? 'Bấm micro để nói lại' : 'Bấm micro rồi nói câu của bạn';
    }
  });

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      this.recorder.stop();
      this.speech.stop();
    });
  }

  protected speaker(index: number): string {
    return this.dialogue().speakers[index] ?? '';
  }

  protected answerOf(index: number): RoleAnswer | null {
    return this.answers().get(index) ?? null;
  }

  protected verdict(a: RoleAnswer): Verdict {
    return verdictOf(a.comparison);
  }

  protected verdictLabel(a: RoleAnswer): string {
    return VERDICT_LABELS[verdictOf(a.comparison)];
  }

  protected hint(): string {
    return hintText(this.line(), this.hints());
  }

  /** Picks a speaker and starts from the first turn. */
  protected start(role: number): void {
    this.speech.stop();
    this.role.set(role);
    this.answers.set(new Map());
    this.enter(0);
  }

  /** Back to choosing a speaker. */
  protected changeRole(): void {
    this.speech.stop();
    this.reading.set(null);
    this.role.set(null);
  }

  protected close(): void {
    this.recorder.stop();
    this.speech.stop();
    this.closed.emit();
  }

  /** Lượt tiếp / Bỏ qua at the learner's line. */
  protected next(): void {
    if (this.phase() === 'idle') {
      this.enter(this.step() + 1);
    }
  }

  protected showHint(): void {
    this.hints.update((h) => Math.min(h + 1, wordCount(this.line())));
  }

  /** Reads one of the turns on screen again (the other speaker's, or the learner's once answered). */
  protected listen(index: number): void {
    const turn = this.dialogue().turns[index];
    if (!turn || !this.canSpeak || this.phase() !== 'idle') {
      return;
    }
    this.reading.set(index);
    this.speech.speak(turn.text, this.rate(), {
      ended: () => this.reading.set(null),
      failed: () => this.reading.set(null),
    });
  }

  /** Dùng micro: downloads the recogniser (asked, since it is about 60MB). */
  protected loadRecognizer(): void {
    this.recognizer.load();
  }

  protected onTyped(event: Event): void {
    this.typed.set((event.target as HTMLInputElement).value);
  }

  protected check(): void {
    const said = this.typed().trim();
    if (said !== '' && this.phase() === 'idle') {
      this.save(said, true);
    }
  }

  /** The microphone button: starts recording, or stops it early. */
  protected async toggleRecord(): Promise<void> {
    if (this.phase() === 'recording') {
      this.recorder.stop();
      return;
    }
    if (!this.canRecordNow()) {
      return;
    }
    const index = this.step();
    this.speech.stop();
    this.error.set(null);
    this.phase.set('recording');
    try {
      const rec = await this.recorder.record((_level, seconds) => this.elapsed.set(seconds));
      this.phase.set('scoring');
      const heard = await this.recognizer.transcribe(rec.audio);
      if (this.step() === index) {
        this.save(heard, false);
      }
    } catch (err) {
      this.error.set(recordError(err));
    } finally {
      this.phase.set('idle');
    }
  }

  private save(said: string, typed: boolean): void {
    const index = this.step();
    const comparison = compareAnswer(this.line(), said, typed);
    this.answers.update((m) => new Map(m).set(index, { said, comparison, typed }));
  }

  /**
   * Moves to turn `index`: the other speaker's line is read, then the next turn comes by itself
   * (at once without a voice); the learner's line waits for them.
   */
  private enter(index: number): void {
    this.step.set(index);
    this.hints.set(0);
    this.typed.set('');
    this.error.set(null);
    this.reading.set(null);
    const turn = this.dialogue().turns[index];
    if (!turn || turn.speaker === this.role()) {
      return;
    }
    if (!this.canSpeak) {
      this.enter(index + 1);
      return;
    }
    this.reading.set(index);
    const after = () => {
      if (this.step() === index) {
        this.enter(index + 1);
      }
    };
    this.speech.speak(turn.text, this.rate(), { ended: after, failed: after });
  }
}
