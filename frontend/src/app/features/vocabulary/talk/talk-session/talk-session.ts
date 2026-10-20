import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { TalkSession, TalkTurn } from '../../../../core/models/talk';
import { SpeechService } from '../../../../core/services/speech.service';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import { SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { MAX_SECONDS, RECORDING } from '../../../../core/stt/voice-recorder';
import { Icon } from '../../../../shared/components/icon/icon';
import { Loading } from '../../../../shared/components/loading/loading';
import { recordError } from '../../../lesson/lesson-detail/speak-step/speak-step';
import { talkError } from '../talk-error';

type Busy = 'none' | 'sending' | 'ending' | 'restarting';
type Mic = 'idle' | 'recording' | 'scoring';

/**
 * One conversation with the AI (added 2026-10-20). The AI's lines are read aloud and can show their
 * Vietnamese; the learner says their line into the microphone (turned into text on the device,
 * then sent with Gửi so a misheard word can be fixed) or types it. Gợi ý shows an answer the AI
 * suggested with its last line. Kết thúc asks for the review, shown under the conversation; an
 * ended session is read-only.
 */
@Component({
  selector: 'lu-talk-session',
  imports: [Icon, Loading, RouterLink],
  templateUrl: './talk-session.html',
  styleUrl: './talk-session.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkSessionPage {
  private readonly api = inject(TalkApiService);
  private readonly router = inject(Router);
  private readonly speech = inject(SpeechService);
  protected readonly recognizer = inject(SpeechRecognizerService);
  private readonly recording = inject(RECORDING);
  private readonly recorder = this.recording.create();
  private readonly chatEnd = viewChild<ElementRef<HTMLElement>>('chatEnd');

  protected readonly canSpeak = this.speech.supported;
  protected readonly micAllowed = this.recording.available() && this.recognizer.supported;
  protected readonly maxSeconds = MAX_SECONDS;

  protected readonly session = signal<TalkSession | null>(null);
  protected readonly loadError = signal(false);
  protected readonly busy = signal<Busy>('none');
  protected readonly error = signal<string | null>(null);
  /** The line being sent, shown in the chat until the answer comes. */
  protected readonly pending = signal<string | null>(null);
  protected readonly text = signal('');
  /** Whether the text came from the microphone. */
  private readonly spoken = signal(false);
  protected readonly mic = signal<Mic>('idle');
  protected readonly elapsed = signal(0);
  protected readonly showHint = signal(false);
  protected readonly showMeaning = signal(false);
  /** The AI turn being read aloud. */
  protected readonly reading = signal<number | null>(null);

  protected readonly learnerTurns = computed(
    () => this.session()?.turns.filter((t) => t.role === 'learner').length ?? 0,
  );
  protected readonly open = computed(() => this.session()?.status === 'open');
  protected readonly turnsLeft = computed(
    () => (this.session()?.maxTurns ?? 0) - this.learnerTurns(),
  );
  protected readonly canSay = computed(() => this.open() && this.turnsLeft() > 0);
  protected readonly canSend = computed(
    () =>
      this.canSay() && this.busy() === 'none' && this.mic() === 'idle' && this.text().trim() !== '',
  );
  protected readonly micStatus = computed(() => {
    switch (this.mic()) {
      case 'recording':
        return 'Đang ghi âm... bấm lần nữa để dừng';
      case 'scoring':
        return 'Đang nghe lại câu của bạn...';
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
        return 'Bấm micro rồi nói câu của bạn';
    }
  });

  /** The session id, from the route. */
  private id = '';

  constructor() {
    inject(ActivatedRoute)
      .paramMap.pipe(takeUntilDestroyed())
      .subscribe((p) => {
        const id = p.get('id') ?? '';
        if (id !== this.session()?.id) {
          this.id = id;
          this.load();
        }
      });
    inject(DestroyRef).onDestroy(() => {
      this.recorder.stop();
      this.speech.stop();
    });
  }

  protected load(): void {
    this.loadError.set(false);
    this.session.set(null);
    this.api.get(this.id).subscribe({
      next: (s) => {
        this.session.set(s);
        if (s.status === 'open') {
          this.readLast(s);
        }
      },
      error: () => this.loadError.set(true),
    });
  }

  protected onInput(event: Event): void {
    this.text.set((event.target as HTMLTextAreaElement).value);
    if (this.text().trim() === '') {
      this.spoken.set(false);
    }
  }

  protected send(): void {
    const s = this.session();
    const text = this.text().trim();
    if (!s || !this.canSend()) {
      return;
    }
    this.speech.stop();
    this.busy.set('sending');
    this.error.set(null);
    this.pending.set(text);
    this.text.set('');
    this.showHint.set(false);
    this.scrollDown();
    this.api.say(s.id, text, this.spoken()).subscribe({
      next: (next) => {
        this.session.set(next);
        this.pending.set(null);
        this.spoken.set(false);
        this.busy.set('none');
        this.readLast(next);
        this.scrollDown();
      },
      error: (err: unknown) => {
        // Nothing was stored: the line goes back to the box to send again.
        this.pending.set(null);
        this.text.set(text);
        this.busy.set('none');
        this.error.set(talkError(err, 'AI chưa trả lời được, vui lòng gửi lại.'));
      },
    });
  }

  protected end(): void {
    const s = this.session();
    if (!s || this.busy() !== 'none' || this.mic() !== 'idle') {
      return;
    }
    this.speech.stop();
    this.busy.set('ending');
    this.error.set(null);
    this.api.end(s.id).subscribe({
      next: (next) => {
        this.session.set(next);
        this.busy.set('none');
      },
      error: (err: unknown) => {
        this.busy.set('none');
        this.error.set(talkError(err, 'AI chưa nhận xét được, vui lòng bấm Kết thúc lần nữa.'));
      },
    });
  }

  /** Luyện lại: a new conversation about the same topic. */
  protected again(): void {
    const s = this.session();
    if (!s || this.busy() !== 'none') {
      return;
    }
    this.busy.set('restarting');
    this.error.set(null);
    this.api.start(s.topicId).subscribe({
      next: (next) => {
        this.busy.set('none');
        this.session.set(next);
        this.id = next.id;
        void this.router.navigate(['/vocabulary/talk', next.id]);
        this.readLast(next);
      },
      error: (err: unknown) => {
        this.busy.set('none');
        this.error.set(talkError(err, 'Chưa bắt đầu được, vui lòng thử lại.'));
      },
    });
  }

  /** Reads one line aloud (the AI's, or the better line of a fix). */
  protected listen(text: string, turn: number | null = null): void {
    if (!this.canSpeak) {
      return;
    }
    this.reading.set(turn);
    this.speech.speak(text, 1, {
      ended: () => this.reading.set(null),
      failed: () => this.reading.set(null),
    });
  }

  protected toggleHint(): void {
    this.showHint.update((v) => !v);
  }

  /** Dùng micro: downloads the recogniser (asked, since it is about 60MB). */
  protected loadRecognizer(): void {
    this.recognizer.load();
  }

  /** The microphone button: records, then puts what was heard in the box. */
  protected async toggleRecord(): Promise<void> {
    if (this.mic() === 'recording') {
      this.recorder.stop();
      return;
    }
    if (
      !this.canSay() ||
      this.mic() !== 'idle' ||
      this.busy() !== 'none' ||
      this.recognizer.state() !== 'ready'
    ) {
      return;
    }
    this.speech.stop();
    this.error.set(null);
    this.mic.set('recording');
    try {
      const rec = await this.recorder.record((_level, seconds) => this.elapsed.set(seconds));
      this.mic.set('scoring');
      const heard = (await this.recognizer.transcribe(rec.audio)).trim();
      if (heard === '') {
        this.error.set('Máy chưa nghe rõ. Hãy nói lại to và rõ hơn.');
      } else {
        this.text.set(heard);
        this.spoken.set(true);
      }
    } catch (err) {
      this.error.set(recordError(err));
    } finally {
      this.mic.set('idle');
    }
  }

  protected isAI(t: TalkTurn): boolean {
    return t.role === 'ai';
  }

  private readLast(s: TalkSession): void {
    const i = s.turns.length - 1;
    const last = s.turns[i];
    if (last?.role === 'ai') {
      this.listen(last.text, i);
    }
  }

  private scrollDown(): void {
    setTimeout(() =>
      this.chatEnd()?.nativeElement.scrollIntoView?.({ block: 'end', behavior: 'smooth' }),
    );
  }
}
