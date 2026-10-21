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
import { NgOptimizedImage } from '@angular/common';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { TalkHint, TalkReaction, TalkSession } from '../../../../core/models/talk';
import { SpeechService } from '../../../../core/services/speech.service';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import { SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { MAX_SECONDS, RECORDING, SPEECH_LEVEL } from '../../../../core/stt/voice-recorder';
import { Icon } from '../../../../shared/components/icon/icon';
import { Loading } from '../../../../shared/components/loading/loading';
import { recordError } from '../../../lesson/lesson-detail/speak-step/speak-step';
import { TalkEndConfirm } from '../talk-end-confirm/talk-end-confirm';
import { talkError } from '../talk-error';
import { TalkHistory } from '../talk-history/talk-history';
import { TalkReview } from '../talk-review/talk-review';

type Busy = 'none' | 'sending' | 'ending' | 'restarting';
type Mic = 'idle' | 'recording' | 'scoring';

/** Speed of Đọc chậm. */
const SLOW_RATE = 0.7;
/** Whether Leo's lines are read aloud by themselves, kept in this browser. */
const AUTO_READ_KEY = 'lu.talk.autoRead';
/** Bars of the wave in the talk button while recording. */
const WAVE_BARS = 22;
/** How far the bars rise when breathing in silence, in px. */
const WAVE_BREATH = 3;
/** How long Leo's face stays up after a new line, in ms. */
const REACTION_MS = 2600;
/** Scenes behind Leo (talk-session.css): the topic picks one, so a topic keeps its own. */
const SCENES = 4;

/** Leo's face for each reaction. */
const FACES: Record<TalkReaction, string> = {
  happy: '😊',
  laugh: '😄',
  surprised: '😮',
  love: '🥰',
  curious: '🤔',
  thumbs: '👍',
};

function savedAutoRead(): boolean {
  try {
    return localStorage.getItem(AUTO_READ_KEY) !== 'off';
  } catch {
    return true;
  }
}

/** Whether the answers show by themselves each turn: asked once, then kept in this browser. */
const HINTS_KEY = 'lu.talk.hints';
type HintsChoice = 'always' | 'on-tap';

function savedHints(): HintsChoice | null {
  try {
    const v = localStorage.getItem(HINTS_KEY);
    return v === 'always' || v === 'on-tap' ? v : null;
  } catch {
    return null;
  }
}

/** A word of Leo's line with where it starts, for the karaoke. */
interface Word {
  text: string;
  start: number;
}

/**
 * One conversation with the AI, Leo, laid out as a video call (added 2026-10-20, reworked
 * 2026-10-21 from design/de-xuat-tro-chuyen-tu-nhien.png). Leo stands in a scene picked by the
 * topic and speaks in a bubble whose words light up as they are read; he leans in while the
 * learner speaks, thinks while the AI answers and shows a small face before his next line. What the
 * device heard is the learner's own bubble, to send, fix or say again. Answers Leo suggests stay
 * behind a pill until asked for; a tapped one is read aloud for the learner to say it, never sent
 * for them. The talk button shows a live wave of the voice while recording; Kết thúc in the top
 * bar ends the call (after a confirm while turns are left); an ended session shows only the review.
 */
@Component({
  selector: 'lu-talk-session',
  imports: [Icon, Loading, NgOptimizedImage, RouterLink, TalkEndConfirm, TalkHistory, TalkReview],
  templateUrl: './talk-session.html',
  styleUrls: ['./talk-session.css', './talk-session-stage.css', './talk-session-dock.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkSessionPage {
  private readonly api = inject(TalkApiService);
  private readonly router = inject(Router);
  private readonly speech = inject(SpeechService);
  protected readonly recognizer = inject(SpeechRecognizerService);
  private readonly recording = inject(RECORDING);
  private readonly recorder = this.recording.create();
  private readonly textBox = viewChild<ElementRef<HTMLTextAreaElement>>('textBox');

  protected readonly canSpeak = this.speech.supported;
  protected readonly micAllowed = this.recording.available() && this.recognizer.supported;
  protected readonly maxSeconds = MAX_SECONDS;

  protected readonly session = signal<TalkSession | null>(null);
  protected readonly loadError = signal(false);
  protected readonly busy = signal<Busy>('none');
  protected readonly error = signal<string | null>(null);
  /** The line being sent, in the learner's bubble until Leo answers. */
  protected readonly pending = signal<string | null>(null);
  /** Speaking (the microphone) or typing (the keyboard). */
  protected readonly typing = signal(!this.micAllowed);
  protected readonly text = signal('');
  /** Whether the text came from the microphone. */
  private readonly spoken = signal(false);
  /** What the device heard, waiting to be sent, fixed or said again. */
  protected readonly heard = signal<string | null>(null);
  protected readonly mic = signal<Mic>('idle');
  protected readonly elapsed = signal(0);
  /** Loudness of the voice being recorded (0–1), the newest last, for the wave in the button. */
  private readonly levels = signal<number[]>([]);
  /** Set by ✕ while recording: what was said is dropped instead of heard. */
  private cancelled = false;
  /** The words heard so far while the learner speaks, in their bubble. */
  protected readonly liveText = signal('');
  /** Whether the voice was loud enough for speech yet in this recording. */
  private spoke = false;
  /** Whether a live reading is under way. */
  private partialBusy = false;
  protected readonly showMeaning = signal(false);
  /** Whether the answers show by themselves; null until the learner answered the question. */
  protected readonly hintsChoice = signal(savedHints());
  protected readonly hintsOpen = signal(this.hintsChoice() === 'always');
  protected readonly showScenario = signal(false);
  /** The answer tapped to say it. */
  protected readonly picked = signal<number | null>(null);
  protected readonly historyOpen = signal(false);
  protected readonly confirmEnd = signal(false);
  protected readonly autoRead = signal(savedAutoRead());
  /** The turn being read aloud (Leo's lines only). */
  protected readonly reading = signal<number | null>(null);
  /** Share of that turn already read (0–1). */
  protected readonly readShare = signal(0);
  /** Leo's face after a new line, for a moment. */
  protected readonly face = signal<string | null>(null);
  private faceTimer: ReturnType<typeof setTimeout> | undefined;

  protected readonly learnerTurns = computed(
    () => this.session()?.turns.filter((t) => t.role === 'learner').length ?? 0,
  );
  protected readonly open = computed(() => this.session()?.status === 'open');
  protected readonly turnsLeft = computed(
    () => (this.session()?.maxTurns ?? 0) - this.learnerTurns(),
  );
  protected readonly canSay = computed(() => this.open() && this.turnsLeft() > 0);
  protected readonly idle = computed(() => this.busy() === 'none' && this.mic() === 'idle');
  protected readonly canSend = computed(
    () => this.canSay() && this.idle() && this.text().trim() !== '',
  );
  /** Leo's last line, the one to answer. */
  protected readonly line = computed(() => {
    const turns = this.session()?.turns ?? [];
    for (let i = turns.length - 1; i >= 0; i--) {
      if (turns[i].role === 'ai') {
        return { index: i, turn: turns[i] };
      }
    }
    return null;
  });
  /** The answer tapped, shown over the microphone to say it. */
  protected readonly pickedText = computed(() => {
    const i = this.picked();
    return i === null ? null : (this.session()?.hints[i]?.text ?? null);
  });
  /** Leo's line in words, each lit once the reading passed its start. */
  protected readonly words = computed<Word[]>(() => {
    const text = this.line()?.turn.text ?? '';
    const out: Word[] = [];
    for (const m of text.matchAll(/\S+\s*/g)) {
      out.push({ text: m[0], start: m.index });
    }
    return out;
  });
  /** How far the karaoke got, in characters; -1 while Leo's line is not being read. */
  protected readonly readTo = computed(() => {
    const line = this.line();
    return line && this.reading() === line.index ? this.readShare() * line.turn.text.length : -1;
  });
  /** The turns ring: the share of the turns said, in degrees. */
  protected readonly progressDeg = computed(() => {
    const s = this.session();
    return s ? Math.min(360, (this.learnerTurns() / s.maxTurns) * 360) : 0;
  });
  /** Minutes from the start to the last line, for the closing card. */
  protected readonly minutes = computed(() => {
    const s = this.session();
    const last = s?.turns[s.turns.length - 1]?.at;
    if (!s || !last) {
      return 0;
    }
    return Math.max(1, Math.round((Date.parse(last) - Date.parse(s.createdAt)) / 60000));
  });
  protected readonly scene = computed(() => {
    const id = this.session()?.topicId ?? '';
    let h = 0;
    for (const c of id) {
      h = (h * 31 + c.charCodeAt(0)) % 997;
    }
    return h % SCENES;
  });
  /** What Leo is doing: his pose and the label under the microphone follow it. */
  protected readonly pose = computed(() => {
    if (this.busy() === 'sending') {
      return 'thinking';
    }
    if (this.mic() !== 'idle') {
      return 'listening';
    }
    return this.reading() !== null ? 'talking' : 'idle';
  });
  protected readonly micLabel = computed(() => {
    switch (this.mic()) {
      case 'recording':
        return 'Nói xong bấm vào nút đỏ để dừng';
      case 'scoring':
        return 'Đang nghe lại câu của bạn...';
      default:
        break;
    }
    if (this.busy() === 'sending') {
      return 'Leo đang nghĩ...';
    }
    if (this.reading() !== null) {
      return 'Leo đang nói...';
    }
    return 'Bấm nút rồi nói câu trả lời của bạn';
  });
  /** The learner asked for less motion: the wave stays flat in silence. */
  private readonly calm =
    typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
  /**
   * The wave in the button while recording: one bar per recent level, in px, oldest first. In
   * silence the bars breathe a little (a slow ripple driven by the recording clock) so the learner
   * sees the microphone is on; a voice louder than the ripple shows as it is.
   */
  protected readonly bars = computed(() => {
    const levels = this.levels();
    const t = this.elapsed();
    return Array.from({ length: WAVE_BARS }, (_, i) => {
      const level = levels[levels.length - WAVE_BARS + i] ?? 0;
      const voice = Math.min(1, level * 8) * 20;
      const breath = this.calm ? 0 : WAVE_BREATH * (0.5 + 0.5 * Math.sin(t * 5 - i * 0.55));
      return Math.round(4 + Math.max(voice, breath));
    });
  });
  /** Time recorded, as 0:06. */
  protected readonly elapsedText = computed(() => {
    const s = Math.floor(this.elapsed());
    return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
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
      clearTimeout(this.faceTimer);
    });
  }

  protected load(): void {
    this.loadError.set(false);
    this.session.set(null);
    this.api.get(this.id).subscribe({
      next: (s) => this.begin(s),
      error: () => this.loadError.set(true),
    });
  }

  protected onInput(event: Event): void {
    this.text.set((event.target as HTMLTextAreaElement).value);
    if (this.text().trim() === '') {
      this.spoken.set(false);
    }
  }

  /** Gửi from the keyboard. */
  protected sendTyped(): void {
    if (this.canSend()) {
      this.send(this.text().trim(), this.spoken());
    }
  }

  /** Gửi from what the device heard. */
  protected sendHeard(): void {
    const heard = this.heard();
    if (heard && this.canSay() && this.idle()) {
      this.send(heard, true);
    }
  }

  /** Sửa: the heard line goes to the keyboard, still counted as spoken. */
  protected fixHeard(): void {
    this.text.set(this.heard() ?? '');
    this.spoken.set(true);
    this.heard.set(null);
    this.useKeyboard();
  }

  /** Nói lại: drops the heard line and records again. */
  protected sayAgain(): void {
    this.heard.set(null);
    void this.toggleRecord();
  }

  protected useKeyboard(): void {
    this.typing.set(true);
    setTimeout(() => this.textBox()?.nativeElement.focus());
  }

  protected useMic(): void {
    this.typing.set(false);
  }

  /** A tapped answer: read aloud for the learner to say it, or put in the box when typing. */
  protected pick(hint: TalkHint, i: number): void {
    if (this.typing()) {
      this.text.set(hint.text);
      this.spoken.set(false);
      this.textBox()?.nativeElement.focus();
      return;
    }
    this.picked.set(i);
    this.listen(hint.text);
  }

  protected toggleAutoRead(): void {
    const on = !this.autoRead();
    this.autoRead.set(on);
    if (!on) {
      this.speech.stop();
      this.reading.set(null);
    }
    try {
      localStorage.setItem(AUTO_READ_KEY, on ? 'on' : 'off');
    } catch {
      // Storage blocked: the choice lasts until the page is left.
    }
  }

  /** Answers the question "show the answers each turn?", or changes the answer later. */
  protected chooseHints(choice: HintsChoice): void {
    this.hintsChoice.set(choice);
    this.hintsOpen.set(choice === 'always');
    try {
      localStorage.setItem(HINTS_KEY, choice);
    } catch {
      // Storage blocked: the choice lasts until the page is left.
    }
  }

  /** Kết thúc: asks first while turns are left and something was said. */
  protected askEnd(): void {
    if (this.learnerTurns() > 0 && this.turnsLeft() > 0) {
      this.confirmEnd.set(true);
    } else {
      this.end();
    }
  }

  protected end(): void {
    const s = this.session();
    this.confirmEnd.set(false);
    if (!s || !this.idle()) {
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
        this.error.set(talkError(err, 'AI chưa nhận xét được, vui lòng bấm lần nữa.'));
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
        this.id = next.id;
        void this.router.navigate(['/vocabulary/talk', next.id]);
        this.begin(next);
      },
      error: (err: unknown) => {
        this.busy.set('none');
        this.error.set(talkError(err, 'Chưa bắt đầu được, vui lòng thử lại.'));
      },
    });
  }

  /** Reads a line aloud: Leo's (with its turn, for the karaoke and his pose) or an answer. */
  protected listen(text: string, turn: number | null = null, rate = 1): void {
    if (!this.canSpeak) {
      return;
    }
    this.reading.set(turn);
    this.readShare.set(0);
    this.speech.speak(text, rate, {
      progress: (share) => this.readShare.set(share),
      ended: () => this.reading.set(null),
      failed: () => this.reading.set(null),
    });
  }

  protected listenSlowly(text: string, turn: number): void {
    this.listen(text, turn, SLOW_RATE);
  }

  /** The microphone before the recogniser is ready: downloads it (about 60MB, asked first). */
  protected loadRecognizer(): void {
    this.recognizer.load();
  }

  /** ✕ while recording: stops and drops what was said. */
  protected cancelRecord(): void {
    if (this.mic() === 'recording') {
      this.cancelled = true;
      this.recorder.stop();
    }
  }

  /** The talk button: records, then shows what was heard. */
  protected async toggleRecord(): Promise<void> {
    if (this.mic() === 'recording') {
      this.recorder.stop();
      return;
    }
    if (!this.canSay() || !this.idle() || this.recognizer.state() !== 'ready') {
      return;
    }
    this.speech.stop();
    this.reading.set(null);
    this.error.set(null);
    this.hintsOpen.set(false);
    this.elapsed.set(0);
    this.levels.set([]);
    this.liveText.set('');
    this.cancelled = false;
    this.spoke = false;
    this.mic.set('recording');
    try {
      const rec = await this.recorder.record(
        (level, seconds) => {
          this.levels.update((l) => [...l.slice(1 - WAVE_BARS), level]);
          this.elapsed.set(seconds);
          this.spoke ||= level >= SPEECH_LEVEL;
        },
        (audio) => this.transcribeSoFar(audio),
      );
      if (this.cancelled) {
        return;
      }
      this.mic.set('scoring');
      const heard = (await this.recognizer.transcribe(rec.audio)).trim();
      if (heard === '') {
        this.error.set('Máy chưa nghe rõ. Hãy nói lại to và rõ hơn.');
      } else {
        this.heard.set(heard);
      }
    } catch (err) {
      this.error.set(recordError(err));
    } finally {
      this.levels.set([]);
      this.liveText.set('');
      this.mic.set('idle');
    }
  }

  /**
   * The live text: what was recorded so far, turned into text while the learner goes on speaking.
   * Only once they spoke (the model makes up words out of silence), and one at a time: a second
   * that comes while the last one is still being read is skipped. The text sent is still read
   * from the whole recording when it stops.
   */
  private transcribeSoFar(audio: Float32Array): void {
    if (!this.spoke || this.partialBusy || this.mic() !== 'recording') {
      return;
    }
    this.partialBusy = true;
    this.recognizer
      .transcribe(audio)
      .then((text) => {
        if (this.mic() === 'recording' && !this.cancelled) {
          this.liveText.set(text.trim());
        }
      })
      .catch(() => undefined) // the live text is only a preview; the final reading reports errors
      .finally(() => (this.partialBusy = false));
  }

  private begin(s: TalkSession): void {
    this.session.set(s);
    this.showScenario.set(s.status === 'open' && this.learnerTurns() === 0);
    this.resetTurn();
    if (s.status === 'open') {
      this.readLast();
    }
  }

  private send(text: string, spoken: boolean): void {
    const s = this.session();
    if (!s) {
      return;
    }
    this.speech.stop();
    this.reading.set(null);
    this.busy.set('sending');
    this.error.set(null);
    this.pending.set(text);
    this.text.set('');
    this.heard.set(null);
    this.hintsOpen.set(false);
    this.showScenario.set(false);
    this.api.say(s.id, text, spoken).subscribe({
      next: (next) => {
        this.session.set(next);
        this.pending.set(null);
        this.busy.set('none');
        this.resetTurn();
        this.react();
        this.readLast();
      },
      error: (err: unknown) => {
        // Nothing was stored: the line goes back where it came from, to send again.
        this.pending.set(null);
        this.busy.set('none');
        if (this.typing()) {
          this.text.set(text);
          this.spoken.set(spoken);
        } else {
          this.heard.set(text);
        }
        this.error.set(talkError(err, 'AI chưa trả lời được, vui lòng gửi lại.'));
      },
    });
  }

  /** Shows Leo's face for his new line, for a moment. */
  private react(): void {
    const reaction = this.line()?.turn.reaction;
    clearTimeout(this.faceTimer);
    this.face.set(reaction ? FACES[reaction] : null);
    if (reaction) {
      this.faceTimer = setTimeout(() => this.face.set(null), REACTION_MS);
    }
  }

  /** A new line of Leo's: nothing picked or heard yet, the answers open if the learner wants. */
  private resetTurn(): void {
    this.picked.set(null);
    this.spoken.set(false);
    this.heard.set(null);
    this.hintsOpen.set(this.hintsChoice() === 'always');
  }

  private readLast(): void {
    const line = this.line();
    if (this.autoRead() && line && line.index === (this.session()?.turns.length ?? 0) - 1) {
      this.listen(line.turn.text, line.index);
    }
  }
}
