import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';

import { SpeechService } from '../../../../core/services/speech.service';
import { SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { MAX_SECONDS, RECORDING } from '../../../../core/stt/voice-recorder';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';
import { Comparison, compareDictation } from '../../../../shared/utils/dictation-compare';

/** The learner's last try at a sentence. */
interface Attempt {
  heard: string;
  comparison: Comparison;
  /** The recording, to listen to again. */
  url: string;
}

type Phase = 'idle' | 'recording' | 'scoring';
type Verdict = 'ok' | 'close' | 'bad';

const VERDICT_LABELS: Record<Verdict, string> = { ok: 'Chính xác!', close: 'Gần đúng', bad: 'Chưa đúng' };

/** Heights (%) of the decorative bars on each side of the microphone, at rest. */
const BARS = [30, 55, 80, 45, 65];

/**
 * Speaking practice (listen and repeat): one short sentence at a time, the learner hears it, records
 * it, and sees which words were heard right, wrong or missing. Speech is turned into text in the
 * browser (SpeechRecognizerService), so the recording never leaves the device. Like the other
 * practice steps, results stay on this page.
 */
@Component({
  selector: 'lu-speak-step',
  imports: [Icon, SpeakButton],
  templateUrl: './speak-step.html',
  styleUrls: ['../practice.css', './speak-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SpeakStep {
  /** Position of this step on the page, shown in its title. */
  readonly number = input(1);
  /** Short sentences to repeat. */
  readonly sentences = input.required<string[]>();
  /** Hoàn thành on the last sentence: go to the next step. */
  readonly done = output<void>();

  protected readonly speech = inject(SpeechService);
  protected readonly recognizer = inject(SpeechRecognizerService);
  private readonly recording = inject(RECORDING);
  private readonly recorder = this.recording.create();
  private readonly player = typeof Audio === 'undefined' ? null : new Audio();

  /** False without a microphone API or over plain http: the step explains and can be skipped. */
  protected readonly micAllowed = this.recording.available() && this.recognizer.supported;
  protected readonly maxSeconds = MAX_SECONDS;

  protected readonly index = linkedSignal({ source: this.sentences, computation: () => 0 });
  private readonly attempts = signal<ReadonlyMap<number, Attempt>>(new Map());
  protected readonly phase = signal<Phase>('idle');
  protected readonly level = signal(0);
  protected readonly elapsed = signal(0);
  protected readonly error = signal<string | null>(null);

  protected readonly sentence = computed(() => this.sentences()[this.index()] ?? '');
  protected readonly attempt = computed(() => this.attempts().get(this.index()) ?? null);
  protected readonly isFirst = computed(() => this.index() === 0);
  protected readonly isLast = computed(() => this.index() >= this.sentences().length - 1);
  protected readonly triedCount = computed(() => this.attempts().size);
  protected readonly verdict = computed<Verdict | null>(() => {
    const c = this.attempt()?.comparison;
    if (!c) {
      return null;
    }
    if (c.totalWords > 0 && c.correctWords === c.totalWords) {
      return 'ok';
    }
    return c.correctWords * 2 >= c.totalWords ? 'close' : 'bad';
  });
  protected readonly verdictLabel = computed(() => {
    const v = this.verdict();
    return v ? VERDICT_LABELS[v] : '';
  });
  /** The bars follow the voice while recording and rest otherwise. */
  protected readonly bars = computed(() => {
    if (this.phase() !== 'recording') {
      return BARS;
    }
    const boost = Math.min(1, this.level() * 8);
    return BARS.map((h) => Math.max(15, Math.min(100, h * (0.4 + boost))));
  });
  protected readonly status = computed(() => {
    switch (this.phase()) {
      case 'recording':
        return 'Đang ghi âm...';
      case 'scoring':
        return 'Đang chấm...';
      default:
        break;
    }
    if (!this.micAllowed) {
      return 'Micro chưa dùng được trên trang này';
    }
    switch (this.recognizer.state()) {
      case 'loading':
        return this.recognizer.starting()
          ? 'Đang khởi động bộ chấm phát âm...'
          : `Đang tải bộ chấm phát âm (một lần, khoảng 60MB)... ${this.recognizer.percent()}%`;
      case 'error':
        return 'Chưa tải được bộ chấm phát âm.';
      default:
        return this.attempt() ? 'Bấm micro để đọc lại' : 'Bấm micro rồi đọc theo câu trên';
    }
  });
  protected readonly canRecordNow = computed(
    () => this.micAllowed && this.recognizer.state() === 'ready' && this.phase() !== 'scoring',
  );

  constructor() {
    if (this.micAllowed) {
      this.recognizer.load();
    }
    inject(DestroyRef).onDestroy(() => {
      this.recorder.stop();
      this.player?.pause();
      this.attempts().forEach((a) => URL.revokeObjectURL(a.url));
    });
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
    const index = this.index();
    const sentence = this.sentence();
    this.speech.stop();
    this.player?.pause();
    this.error.set(null);
    this.phase.set('recording');
    try {
      const rec = await this.recorder.record((level, seconds) => {
        this.level.set(level);
        this.elapsed.set(seconds);
      });
      this.phase.set('scoring');
      const heard = await this.recognizer.transcribe(rec.audio);
      const url = URL.createObjectURL(rec.blob);
      this.attempts.update((m) => {
        const next = new Map(m);
        const old = next.get(index);
        if (old) {
          URL.revokeObjectURL(old.url);
        }
        return next.set(index, { heard, comparison: compareDictation(sentence, heard), url });
      });
    } catch (err) {
      this.error.set(recordError(err));
    } finally {
      this.phase.set('idle');
      this.level.set(0);
    }
  }

  protected playBack(): void {
    const a = this.attempt();
    if (a && this.player) {
      this.speech.stop();
      this.player.src = a.url;
      void this.player.play();
    }
  }

  protected retryLoad(): void {
    this.recognizer.load();
  }

  /** Làm lại bước này: every try forgotten, back to the first sentence. */
  protected restart(): void {
    if (this.phase() !== 'idle') {
      return;
    }
    this.player?.pause();
    this.attempts().forEach((a) => URL.revokeObjectURL(a.url));
    this.attempts.set(new Map());
    this.error.set(null);
    this.index.set(0);
  }

  protected go(index: number): void {
    if (index < 0 || index >= this.sentences().length || this.phase() !== 'idle') {
      return;
    }
    this.speech.stop();
    this.player?.pause();
    this.error.set(null);
    this.index.set(index);
  }
}

/** A message for a recording that could not be made or scored. */
export function recordError(err: unknown): string {
  const name = err instanceof DOMException ? err.name : '';
  if (name === 'NotAllowedError' || name === 'SecurityError') {
    return 'Chưa cho phép dùng micro. Hãy cho phép trong cài đặt trang của trình duyệt rồi thử lại.';
  }
  if (name === 'NotFoundError') {
    return 'Không tìm thấy micro trên máy này.';
  }
  return 'Chưa chấm được lần đọc này. Hãy thử lại.';
}
