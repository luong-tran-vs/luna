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
  untracked,
} from '@angular/core';

import { ListenQuestion, PracticeView } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { AudioBar } from '../../../../shared/components/audio-bar/audio-bar';
import { Icon } from '../../../../shared/components/icon/icon';
import { estimateSeconds } from '../../../../shared/utils/fake-waveform';
import { Listening } from '../../listening/listening';
import { newSeed, shuffle } from '../practice-logic';

/** Lượt 1 (the gist), lượt 2 (the details), then chép lại (the dictation). */
export type ListenPhase = 'gist' | 'detail' | 'dictation';

const SPEEDS = [0.75, 1];
const LETTERS = 'ABCDEFGH';

/**
 * The order the options of a question are shown in: shown position → index in question.options.
 * Shuffled with the seed, except a tfn question, whose Đúng / Sai / Không nói stay in their order.
 */
export function optionOrder(q: ListenQuestion, seed: number): number[] {
  const order = q.options.map((_, i) => i);
  return q.kind === 'tfn' ? order : shuffle(order, seed);
}

/**
 * The Listening step of the six-step lesson (design 2026-10-22, section 2). Lượt 1: the dialogue is
 * read aloud without its text (one browser voice per speaker) and the learner answers the gist
 * question. Lượt 2: the detail and true/false/not-given questions, with each turn playable alone and
 * the turn holding the answer shown after the check; the whole text with its meaning can be shown
 * once every question is answered. Then the short dictation sentences (practice.dictation) with
 * lu-listening, which records them. The questions are checked here only, nothing is sent.
 * Without the new content (practice.steps null, an older lesson) it goes straight to the dictation.
 */
@Component({
  selector: 'lu-listen-step',
  imports: [AudioBar, Icon, Listening],
  templateUrl: './listen-step.html',
  styleUrls: ['../practice.css', './listen-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ListenStep {
  protected readonly speech = inject(SpeechService);

  readonly lessonId = input.required<string>();
  /** The lesson's practice: dialogue, steps.listeningQuestions, dictation. null while loading or without one. */
  readonly practice = input<PracticeView | null>(null);
  /** study: answering for the lesson being studied; review: a done step looked at again. */
  readonly mode = input<'study' | 'review'>('study');
  /** Dictation sentence saved last time, to start from. */
  readonly startSentence = input<number | null>(null);
  readonly number = input(0);
  readonly canBack = input(false);
  /** "Để sau, sang bước Đọc" when the step may be left for later; '' otherwise. */
  readonly laterLabel = input('');
  /** The button once done ("Sang bước Đọc"). */
  readonly nextLabel = input('');
  readonly back = output<void>();
  /** Leaves the step (later, or on once done/reviewed). */
  readonly leave = output<void>();
  /** Study mode: the dictation is finished; the page records the step. */
  readonly completed = output<void>();
  /** Study mode: the dictation sentence the learner is on, to save. */
  readonly position = output<number>();

  protected readonly speeds = SPEEDS;
  protected readonly letters = LETTERS;
  /** A new option order every time the step is opened. */
  private readonly seed = newSeed();

  protected readonly dialogue = computed(() => {
    const d = this.practice()?.dialogue;
    return d && d.turns.length ? d : null;
  });
  protected readonly questions = computed(() =>
    this.dialogue() ? (this.practice()?.steps?.listeningQuestions ?? []) : [],
  );
  protected readonly gist = computed(() => this.questions().filter((q) => q.kind === 'gist'));
  protected readonly detail = computed(() => this.questions().filter((q) => q.kind !== 'gist'));
  /** The dictation sentences; null to dictate the lesson's own sentences (no practice). */
  protected readonly items = computed(() => {
    const d = this.practice()?.dictation;
    return d && d.length ? d : null;
  });
  /** Short sentences written for the step, rather than every sentence of an older lesson. */
  protected readonly shortDictation = computed(() => !!this.practice()?.steps && this.items() !== null);

  /** Where the step opens: the dictation when there are no questions or it was reached before. */
  private readonly firstPhase = computed<ListenPhase>(() => {
    if (!this.questions().length) {
      return 'dictation';
    }
    // Read once: the saved sentence changes as the learner moves on, which must not move the phase.
    if (untracked(() => this.mode() === 'study' && (this.startSentence() ?? 0) > 0)) {
      return 'dictation';
    }
    return this.gist().length ? 'gist' : 'detail';
  });
  protected readonly phase = linkedSignal(() => this.firstPhase());
  /** The question on screen, within the phase. */
  protected readonly qIndex = linkedSignal(() => {
    this.phase();
    return 0;
  });
  protected readonly list = computed(() => (this.phase() === 'gist' ? this.gist() : this.detail()));
  protected readonly question = computed<ListenQuestion | null>(() => this.list()[this.qIndex()] ?? null);
  /** The question's key: its position among all questions. */
  private readonly key = computed(() => {
    const q = this.question();
    return q ? this.questions().indexOf(q) : -1;
  });
  /** Shown options: text and index in question.options. */
  protected readonly options = computed(() => {
    const q = this.question();
    if (!q) {
      return [];
    }
    return optionOrder(q, this.seed + this.key()).map((i) => ({ text: q.options[i], index: i }));
  });
  /** Chosen option (index in question.options) of every checked question, by key. */
  private readonly answers = signal<ReadonlyMap<number, number>>(new Map());
  /** The option picked for the question on screen, not checked yet. */
  protected readonly picked = linkedSignal<number | null>(() => {
    this.key();
    return null;
  });
  protected readonly answer = computed(() => this.answers().get(this.key()) ?? null);
  protected readonly checked = computed(() => this.answer() !== null);
  protected readonly right = computed(() => this.answer() !== null && this.answer() === this.question()?.answerIndex);
  /** The turn holding the answer, shown once the question is checked. */
  protected readonly evidence = computed(() => {
    const q = this.question();
    const turn = q && q.turnIndex >= 0 ? this.dialogue()?.turns[q.turnIndex] : undefined;
    return turn ? { index: q!.turnIndex, turn } : null;
  });
  /** Every question answered: the text of the dialogue can be shown. */
  protected readonly allAnswered = computed(() => this.answers().size >= this.questions().length);
  protected readonly showText = signal(false);
  protected readonly isLast = computed(() => this.qIndex() >= this.list().length - 1);
  protected readonly mainLabel = computed(() => {
    if (!this.checked()) {
      return 'Kiểm tra';
    }
    if (!this.isLast()) {
      return 'Câu tiếp';
    }
    return this.phase() === 'gist' && this.detail().length ? 'Sang lượt 2' : 'Sang phần chép lại';
  });
  /** ← goes from lượt 2 to lượt 1, from lượt 1 to the step before. */
  protected readonly canGoBack = computed(
    () => (this.phase() === 'detail' && this.gist().length > 0) || this.canBack(),
  );

  // --- the player ---

  protected readonly rate = signal(1);
  /** 'all': the whole dialogue, turn after turn; 'one': a single turn. */
  protected readonly playing = signal<'all' | 'one' | null>(null);
  /** The turn being read. */
  protected readonly turn = signal<number | null>(null);
  /** Share of the turn being read (0–1). */
  private readonly share = signal(0);
  /** Times the whole dialogue was heard to the end. */
  protected readonly heard = signal(0);
  private readonly turnSeconds = computed(() =>
    (this.dialogue()?.turns ?? []).map((t) => estimateSeconds(t.text)),
  );
  protected readonly duration = computed(() => {
    const secs = this.turnSeconds();
    const t = this.turn();
    const total = this.playing() === 'one' && t !== null ? secs[t] : secs.reduce((a, b) => a + b, 0);
    return total / this.rate();
  });
  protected readonly progress = computed(() => {
    const t = this.turn();
    if (t === null) {
      return 0;
    }
    if (this.playing() === 'one') {
      return this.share();
    }
    const secs = this.turnSeconds();
    const total = secs.reduce((a, b) => a + b, 0) || 1;
    return (secs.slice(0, t).reduce((a, b) => a + b, 0) + secs[t] * this.share()) / total;
  });
  protected readonly status = computed(() => {
    const t = this.turn();
    const n = this.dialogue()?.turns.length ?? 0;
    if (this.playing() === 'all' && t !== null) {
      return `Đang phát · lượt ${t + 1}/${n}`;
    }
    if (this.playing() === 'one' && t !== null) {
      return `Đang phát lượt ${t + 1}`;
    }
    return this.heard() ? 'Đã nghe xong · nghe lại bao nhiêu lần cũng được' : `Bấm để nghe cả đoạn (${n} lượt)`;
  });

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      if (this.playing()) {
        this.speech.stop();
      }
    });
  }

  protected speaker(index: number): string {
    return this.dialogue()?.speakers[index] ?? '';
  }

  protected initial(speaker: number): string {
    return (this.speaker(speaker)[0] ?? '').toUpperCase();
  }

  /** "M3": the speaker's initial and the turn's number. */
  protected chipLabel(index: number): string {
    const t = this.dialogue()?.turns[index];
    return t ? `${this.initial(t.speaker)}${index + 1}` : '';
  }

  protected togglePlayAll(): void {
    if (this.playing() === 'all') {
      this.stop();
    } else {
      this.playing.set('all');
      this.read(0);
    }
  }

  /** A turn chip or 🔈: reads that turn alone, or stops it when it is being read. */
  protected toggleTurn(index: number): void {
    if (this.playing() === 'one' && this.turn() === index) {
      this.stop();
    } else {
      this.playing.set('one');
      this.read(index);
    }
  }

  protected stop(): void {
    this.speech.stop();
    this.reset();
  }

  private read(index: number): void {
    const turn = this.dialogue()?.turns[index];
    if (!turn) {
      return;
    }
    this.turn.set(index);
    this.share.set(0);
    // One browser voice per speaker, so the two people sound different.
    this.speech.speak(
      turn.text,
      this.rate(),
      {
        progress: (share) => this.share.set(share),
        ended: () => this.onEnd(index),
        failed: () => this.onEnd(index),
      },
      turn.speaker,
    );
  }

  private onEnd(index: number): void {
    if (this.playing() === 'all' && index + 1 < (this.dialogue()?.turns.length ?? 0)) {
      this.read(index + 1);
      return;
    }
    if (this.playing() === 'all') {
      this.heard.update((n) => n + 1);
    }
    this.reset();
  }

  private reset(): void {
    this.playing.set(null);
    this.turn.set(null);
    this.share.set(0);
  }

  // --- the questions ---

  protected pick(index: number): void {
    if (!this.checked()) {
      this.picked.set(index);
    }
  }

  /** The main button: Kiểm tra, Câu tiếp, or on to the next part. */
  protected main(): void {
    if (!this.checked()) {
      const p = this.picked();
      if (p !== null) {
        this.answers.update((m) => new Map(m).set(this.key(), p));
      }
      return;
    }
    if (!this.isLast()) {
      this.qIndex.update((i) => i + 1);
      return;
    }
    this.stopPlayer();
    this.phase.set(this.phase() === 'gist' && this.detail().length ? 'detail' : 'dictation');
  }

  protected goBack(): void {
    this.stopPlayer();
    if (this.phase() === 'detail' && this.gist().length) {
      this.phase.set('gist');
      this.qIndex.set(this.gist().length - 1);
    } else if (this.canBack()) {
      this.back.emit();
    }
  }

  /** ← on the first dictation sentence: back to the questions, or to the step before. */
  protected backFromDictation(): void {
    const last = this.detail().length ? 'detail' : this.gist().length ? 'gist' : null;
    if (last) {
      this.phase.set(last);
      this.qIndex.set(this.list().length - 1);
    } else {
      this.back.emit();
    }
  }

  private stopPlayer(): void {
    if (this.playing()) {
      this.stop();
    }
  }
}
