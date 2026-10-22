import { NgOptimizedImage } from '@angular/common';
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
import { Router } from '@angular/router';

import {
  ReadAloud,
  SpeakAnswer,
  SpeakQuestion,
  SpeakResult,
} from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import { SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { MAX_SECONDS, RECORDING } from '../../../../core/stt/voice-recorder';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';
import { compareDictation, normalizeWords } from '../../../../shared/utils/dictation-compare';
import { DiffPart, wordDiff } from '../../../../shared/utils/word-diff';
import { ReadingApiService } from '../../reading-api.service';
import { recordError } from '../speak-step/speak-step';

type Phase = 'read' | 'answer' | 'summary';
type RecState = 'idle' | 'recording' | 'scoring';

/** How a word of the sentence was heard: green, yellow or red (design, section 5a). */
export type WordMark = 'ok' | 'close' | 'miss';

export interface MarkedWord {
  text: string;
  mark: WordMark;
}

/** The learner's last try at a sentence to repeat. */
interface ReadTry {
  heard: string;
  words: MarkedWord[];
  /** The recording, to listen to again ("Giọng tôi"). */
  url: string;
}

/** The AI check of the answers, for one set of answers (key). */
interface Check {
  key: string;
  state: 'loading' | 'done' | 'error';
  results: SpeakResult[];
  note: string;
}

/** One answer of the summary. */
export interface SummaryItem {
  question: string;
  said: string;
  /** null: no correction to show (fine, unchecked or empty). */
  diff: DiffPart[] | null;
  ok: boolean;
  noteVi: string;
}

/** At most this many sentences to repeat (design: "rút còn 2 câu"). */
const MAX_SENTENCES = 2;
/** The server refuses longer answers. */
export const MAX_ANSWER = 300;

/**
 * Colours the words of the sentence by how they were heard: each word written in the sentence
 * (with its capitals and punctuation) gets the marks of the words it normalises to.
 */
export function markSentence(sentence: string, heard: string): MarkedWord[] {
  const compared = compareDictation(sentence, heard, { slips: false }).words.filter(
    (w) => w.expected !== undefined,
  );
  let k = 0;
  return sentence
    .split(/\s+/)
    .filter((t) => t !== '')
    .map((text) => {
      const n = normalizeWords(text).length;
      const part = compared.slice(k, k + n);
      k += n;
      const okCount = part.filter((w) => w.status === 'ok' || w.status === 'close').length;
      const missed = part.filter((w) => w.status === 'missing').length;
      const mark: WordMark =
        okCount === part.length ? 'ok' : missed === part.length ? 'miss' : 'close';
      return { text, mark };
    });
}

/**
 * The sentence before it is read, coloured like the design (5a): the word the tip is about red (the
 * sound to work on), the longest other word yellow (worth care), the rest green. A word counts as
 * the tip's when it appears whole in the tip before its ":", or in the whole tip when it has none.
 * Reading it then colours the words by how they were heard.
 */
export function previewMarks(sentence: string, tip: string): MarkedWord[] {
  const head = (tip.includes(':') ? tip.slice(0, tip.indexOf(':')) : tip).toLowerCase();
  const named = new Set(head.match(/[a-z']+/g) ?? []);
  const words = sentence.split(/\s+/).filter((t) => t !== '');
  const bare = words.map((t) => t.toLowerCase().replace(/[^a-z']/g, ''));
  const hard = bare.findIndex((w) => w.length > 1 && named.has(w));
  let care = -1;
  if (hard >= 0) {
    bare.forEach((w, i) => {
      if (i !== hard && w.length >= 4 && (care < 0 || w.length > bare[care].length)) {
        care = i;
      }
    });
  }
  return words.map((text, i) => ({
    text,
    mark: i === hard ? 'miss' : i === care ? 'close' : 'ok',
  }));
}

/** The parts of a frame line around its "___" blanks. */
export function frameLines(frame: string): string[][] {
  return frame
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l !== '')
    .map((l) => l.split(/_{2,}/));
}

/** The short note shown when the answers could not be checked. */
function checkNote(err: unknown): string {
  const e = err as { status?: number; body?: unknown; error?: unknown } | null;
  // ApiError carries body, a bare HttpErrorResponse (no interceptor) carries error.
  const code = ((e?.body ?? e?.error) as { error?: string } | null)?.error;
  if (code === 'speak_limit') {
    return 'Hôm nay bạn đã nhờ nhận xét bài này đủ lượt, nên lần này chưa có nhận xét.';
  }
  if (code === 'ai_quota' || e?.status === 503) {
    return 'AI đang bận nên chưa nhận xét được. Câu trả lời của bạn vẫn được giữ ở đây.';
  }
  return 'Chưa nhận xét được lúc này. Câu trả lời của bạn vẫn được giữ ở đây.';
}

/**
 * The Speaking step of a lesson being studied (2026-10-22, design section 5): repeat two sentences
 * with a tip each, answer the questions aloud (or typed) with an answer frame, then the answers
 * checked by the AI in one request and an offer to talk with Leo about the topic. Speech is
 * recognised in the browser (SpeechRecognizerService); without a microphone everything can be typed.
 */
@Component({
  selector: 'lu-speak-lesson',
  imports: [Icon, NgOptimizedImage, SpeakButton],
  templateUrl: './speak-lesson.html',
  styleUrls: ['../practice.css', './speak-lesson.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SpeakLesson {
  readonly lessonId = input.required<string>();
  /** The topic of the goal, to open the talk with Leo on it; '' when unknown. */
  readonly topicId = input('');
  readonly topicName = input('');
  /** The sentences to repeat with their tips; empty for older lessons (then fallbackSentences). */
  readonly readAloud = input<ReadAloud[]>([]);
  readonly fallbackSentences = input<string[]>([]);
  /** Questions to answer aloud; empty for older lessons (the step is then the repeating only). */
  readonly questions = input<SpeakQuestion[]>([]);
  readonly number = input(0);
  readonly finishLabel = input('Tiếp theo');
  readonly canBack = input(false);
  readonly back = output<void>();
  readonly done = output<void>();

  protected readonly speech = inject(SpeechService);
  protected readonly recognizer = inject(SpeechRecognizerService);
  private readonly api = inject(ReadingApiService);
  private readonly talk = inject(TalkApiService);
  private readonly router = inject(Router);
  private readonly recording = inject(RECORDING);
  private readonly recorder = this.recording.create();
  private readonly player = typeof Audio === 'undefined' ? null : new Audio();

  /** False without a microphone API or over plain http: everything is then typed or skipped. */
  protected readonly micAllowed = this.recording.available() && this.recognizer.supported;
  protected readonly maxSeconds = MAX_SECONDS;
  protected readonly maxAnswer = MAX_ANSWER;

  protected readonly sentences = computed<ReadAloud[]>(() => {
    const given = this.readAloud().filter((r) => r.text.trim() !== '');
    return given.length
      ? given.slice(0, MAX_SENTENCES)
      : this.fallbackSentences()
          .slice(0, MAX_SENTENCES)
          .map((text) => ({ text, tipVi: '' }));
  });

  protected readonly phase = signal<Phase>('read');
  protected readonly readIndex = signal(0);
  protected readonly qIndex = signal(0);
  private readonly tries = signal<ReadonlyMap<number, ReadTry>>(new Map());
  protected readonly answers = linkedSignal<SpeakQuestion[], string[]>({
    source: this.questions,
    computation: (qs) => qs.map(() => ''),
  });
  /** The typing box is open; always when there is no microphone. */
  protected readonly typing = signal(!this.micAllowed);
  protected readonly rec = signal<RecState>('idle');
  protected readonly level = signal(0);
  protected readonly elapsed = signal(0);
  protected readonly error = signal<string | null>(null);
  protected readonly check = signal<Check | null>(null);
  protected readonly openingTalk = signal(false);

  protected readonly sentence = computed(() => this.sentences()[this.readIndex()] ?? null);
  protected readonly readTry = computed(() => this.tries().get(this.readIndex()) ?? null);
  /** The sentence on screen, coloured: by the last try, else as the design shows it before reading. */
  protected readonly marks = computed<MarkedWord[]>(() => {
    const s = this.sentence();
    return this.readTry()?.words ?? (s ? previewMarks(s.text, s.tipVi) : []);
  });
  protected readonly question = computed(() => this.questions()[this.qIndex()] ?? null);
  protected readonly answer = computed(() => this.answers()[this.qIndex()] ?? '');
  protected readonly frame = computed(() => frameLines(this.question()?.frame ?? ''));
  protected readonly spokenCount = computed(
    () => this.answers().filter((a) => a.trim() !== '').length,
  );
  protected readonly busy = computed(() => this.rec() !== 'idle');
  protected readonly canRecordNow = computed(
    () => this.micAllowed && this.recognizer.state() === 'ready' && this.rec() !== 'scoring',
  );
  protected readonly hasBack = computed(
    () => this.canBack() || this.phase() !== 'read' || this.readIndex() > 0,
  );

  protected readonly mainLabel = computed(() => {
    switch (this.phase()) {
      case 'read':
        if (this.readIndex() < this.sentences().length - 1) {
          return 'Câu tiếp';
        }
        return this.questions().length ? 'Sang phần trả lời' : this.finishLabel();
      case 'answer':
        return this.qIndex() < this.questions().length - 1 ? 'Câu tiếp' : 'Xem nhận xét';
      default:
        return this.finishLabel();
    }
  });

  protected readonly status = computed(() => {
    switch (this.rec()) {
      case 'recording':
        return 'Đang ghi âm...';
      case 'scoring':
        return 'Đang nghe lại...';
      default:
        break;
    }
    if (!this.micAllowed) {
      return '';
    }
    switch (this.recognizer.state()) {
      case 'loading':
        return this.recognizer.starting()
          ? 'Đang khởi động bộ nhận giọng nói...'
          : `Đang tải bộ nhận giọng nói (một lần, khoảng 60MB)... ${this.recognizer.percent()}%`;
      case 'error':
        return 'Chưa tải được bộ nhận giọng nói. Bạn vẫn có thể gõ.';
      default:
        return '';
    }
  });

  protected readonly summary = computed<SummaryItem[]>(() => {
    const c = this.check();
    return this.questions().map((q, i) => {
      const said = this.answers()[i]?.trim() ?? '';
      const r = c?.state === 'done' ? c.results[i] : undefined;
      const changed = !!r && !!said && r.corrected.trim() !== '' && r.corrected !== said;
      return {
        question: q.question,
        said,
        diff: changed ? wordDiff(said, r.corrected) : null,
        ok: !!r && !!said && !changed,
        noteVi: changed ? r.noteVi : '',
      };
    });
  });

  constructor() {
    if (this.micAllowed) {
      this.recognizer.load();
    }
    inject(DestroyRef).onDestroy(() => {
      this.recorder.stop();
      this.player?.pause();
      this.tries().forEach((t) => URL.revokeObjectURL(t.url));
    });
  }

  /** Records one utterance and returns what was heard, or null when it failed (error shown). */
  private async listen(): Promise<{ heard: string; blob: Blob } | null> {
    this.speech.stop();
    this.player?.pause();
    this.error.set(null);
    this.rec.set('recording');
    try {
      const r = await this.recorder.record((level, seconds) => {
        this.level.set(level);
        this.elapsed.set(seconds);
      });
      this.rec.set('scoring');
      const heard = (await this.recognizer.transcribe(r.audio)).trim();
      return { heard, blob: r.blob };
    } catch (err) {
      this.error.set(recordError(err));
      return null;
    } finally {
      this.rec.set('idle');
      this.level.set(0);
    }
  }

  /** The microphone of "Đọc theo": record, then colour the sentence. */
  protected async recordSentence(): Promise<void> {
    if (this.rec() === 'recording') {
      this.recorder.stop();
      return;
    }
    const s = this.sentence();
    if (!s || !this.canRecordNow()) {
      return;
    }
    const index = this.readIndex();
    const got = await this.listen();
    if (!got) {
      return;
    }
    const url = URL.createObjectURL(got.blob);
    this.tries.update((m) => {
      const next = new Map(m);
      const old = next.get(index);
      if (old) {
        URL.revokeObjectURL(old.url);
      }
      return next.set(index, { heard: got.heard, words: markSentence(s.text, got.heard), url });
    });
  }

  /** The microphone of "Trả lời" (and ↻ Nói lại): what is heard replaces the answer. */
  protected async recordAnswer(): Promise<void> {
    if (this.rec() === 'recording') {
      this.recorder.stop();
      return;
    }
    if (!this.canRecordNow()) {
      return;
    }
    const index = this.qIndex();
    const got = await this.listen();
    if (got) {
      this.setAnswer(index, got.heard.slice(0, MAX_ANSWER));
    }
  }

  protected playBack(): void {
    const t = this.readTry();
    if (t && this.player) {
      this.speech.stop();
      this.player.src = t.url;
      void this.player.play();
    }
  }

  protected retryLoad(): void {
    this.recognizer.load();
  }

  protected toggleTyping(): void {
    this.typing.update((v) => !v || !this.micAllowed);
  }

  protected typed(event: Event): void {
    this.setAnswer(this.qIndex(), (event.target as HTMLTextAreaElement).value);
  }

  private setAnswer(index: number, text: string): void {
    this.answers.update((a) => a.map((v, i) => (i === index ? text : v)));
  }

  private stopAll(): void {
    this.speech.stop();
    this.player?.pause();
    this.error.set(null);
  }

  protected next(): void {
    if (this.busy()) {
      return;
    }
    this.stopAll();
    switch (this.phase()) {
      case 'read':
        if (this.readIndex() < this.sentences().length - 1) {
          this.readIndex.update((i) => i + 1);
        } else if (this.questions().length) {
          this.qIndex.set(0);
          this.phase.set('answer');
        } else {
          this.done.emit();
        }
        break;
      case 'answer':
        if (this.qIndex() < this.questions().length - 1) {
          this.qIndex.update((i) => i + 1);
        } else {
          this.phase.set('summary');
          this.runCheck();
        }
        break;
      default:
        this.done.emit();
    }
  }

  /** ← goes back inside the step first, then to the step before. */
  protected prev(): void {
    if (this.busy()) {
      return;
    }
    this.stopAll();
    if (this.phase() === 'summary') {
      this.phase.set('answer');
    } else if (this.phase() === 'answer' && this.qIndex() > 0) {
      this.qIndex.update((i) => i - 1);
    } else if (this.phase() === 'answer') {
      this.readIndex.set(Math.max(0, this.sentences().length - 1));
      this.phase.set('read');
    } else if (this.readIndex() > 0) {
      this.readIndex.update((i) => i - 1);
    } else {
      this.back.emit();
    }
  }

  /** Checks the answers in one request; the same answers are not sent twice (5 checks a day). */
  private runCheck(): void {
    const payload: SpeakAnswer[] = this.questions().map((q, i) => ({
      question: q.question,
      said: this.answers()[i]?.trim() ?? '',
    }));
    const key = JSON.stringify(payload);
    const old = this.check();
    if (old?.key === key && old.state !== 'error') {
      return;
    }
    if (payload.every((a) => a.said === '')) {
      // Nothing to correct: no request.
      this.check.set({
        key,
        state: 'done',
        results: payload.map(() => ({ said: '', corrected: '', noteVi: '' })),
        note: '',
      });
      return;
    }
    this.check.set({ key, state: 'loading', results: [], note: '' });
    this.api.checkSpeaking(this.lessonId(), payload).subscribe({
      next: (results) => {
        if (this.check()?.key === key) {
          this.check.set({ key, state: 'done', results, note: '' });
        }
      },
      error: (err: unknown) => {
        if (this.check()?.key === key) {
          this.check.set({ key, state: 'error', results: [], note: checkNote(err) });
        }
      },
    });
  }

  /**
   * The card "Nói chuyện với Leo": the step counts as done (the learner leaves the lesson from here),
   * then a talk on this topic starts, or the talk page opens.
   */
  protected openTalk(): void {
    if (this.openingTalk()) {
      return;
    }
    this.done.emit();
    const topicId = this.topicId();
    if (!topicId) {
      void this.router.navigate(['/vocabulary/talk']);
      return;
    }
    this.openingTalk.set(true);
    this.talk.start(topicId).subscribe({
      next: (s) => void this.router.navigate(['/vocabulary/talk', s.id]),
      // The topic page shows what went wrong and lets the learner pick again.
      error: () => {
        this.openingTalk.set(false);
        void this.router.navigate(['/vocabulary/talk']);
      },
    });
  }
}
