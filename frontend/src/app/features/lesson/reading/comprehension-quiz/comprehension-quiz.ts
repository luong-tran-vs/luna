import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  Injector,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { Quiz, QuizAnswer, QuizKind, QuizQuestion } from '../../../../core/models/reading';
import { Asker } from '../../../../shared/components/asker/asker';
import { Icon } from '../../../../shared/components/icon/icon';
import { ReadingApiService } from '../../reading-api.service';

export interface QuizScore {
  correct: number;
  total: number;
}

const SEND_FAILED = 'Không gửi được câu trả lời, vui lòng thử lại.';

/** The kind's short name on the tabs and in the result; '' (older questions): just "Câu". */
const KIND_SHORT: Record<QuizKind, string> = {
  gist: 'Ý chính',
  detail: 'Chi tiết',
  tfn: 'Đ/S/K',
  vocab: 'Nghĩa từ',
  '': 'Câu',
};

/** What the kind asks, over the question; '' for none. */
const KIND_TITLE: Record<QuizKind, string> = {
  gist: 'Ý chính của bài',
  detail: 'Chi tiết trong bài',
  tfn: 'Đúng, sai hay không có trong bài?',
  vocab: 'Nghĩa từ trong bài',
  '': '',
};

const kindOf = (q: QuizQuestion): QuizKind => q.kind ?? '';

/**
 * The order the options are shown in, as indexes into q.options. True / false / not given keeps
 * its fixed order, and so do questions written before kinds existed (their options may refer to
 * each other); the others are shuffled so the right answer is not always at the same place.
 */
export function optionOrder(q: QuizQuestion, random: () => number = Math.random): number[] {
  const order = q.options.map((_, i) => i);
  const kind = kindOf(q);
  if (kind === 'tfn' || kind === '') {
    return order;
  }
  for (let i = order.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [order[i], order[j]] = [order[j], order[i]];
  }
  return order;
}

/** What the main button of the bar does. */
type MainAction = 'check' | 'next' | 'missing' | 'result' | 'finish';

/**
 * Comprehension questions of the Reading step (F15; design, 2026-10-15): one question at a time,
 * asked by a boy in a message bubble, a row of tabs (number and kind, 2026-10-22) to jump to any
 * of them. The options are shuffled (see optionOrder) but answers use their server index. Pick an
 * option then Kiểm tra: the bubble's colour and label say right or wrong, the options are marked
 * where they are and the explanation shows under them, so nothing on screen moves. ← and "Câu tiếp
 * theo" go through the questions in order; after the last one comes the result (the score, each
 * question, doing the wrong ones or all of them again) and the button on to the next step.
 * Answers come from the server: reloading resumes at the first unanswered question, or on the
 * result once every question is answered.
 */
@Component({
  selector: 'lu-comprehension-quiz',
  imports: [Asker, Icon],
  templateUrl: './comprehension-quiz.html',
  styleUrl: './comprehension-quiz.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ComprehensionQuiz {
  private readonly api = inject(ReadingApiService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);

  readonly lessonId = input.required<string>();
  readonly quiz = input.required<Quiz>();
  /** Review mode never reports completion. */
  readonly review = input(false);
  /** Whether ← on the first question goes back to the text. */
  readonly canBack = input(false);
  /** The button on the result, e.g. "Sang bước Nghe"; empty for none. */
  readonly finishLabel = input('');

  /** Emitted when the last open question gets answered (the result can be shown). */
  readonly allAnswered = output<QuizScore>();
  /** The questions changed on the server; the page should load them again. */
  readonly reload = output<void>();
  readonly back = output<void>();
  /** The result's button: on to the next step. */
  readonly finished = output<void>();
  /** Whether the result is on screen, each time it changes (for the page's title, "Đọc · Kết quả"). */
  readonly resultChange = output<boolean>();

  protected readonly answers = linkedSignal(() => [...this.quiz().answers]);
  protected readonly sending = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly changed = signal(false);
  protected readonly announcement = signal('');
  /** The option picked on each question not checked yet; kept while going back and forth. */
  private readonly picks = signal<ReadonlyMap<number, number>>(new Map());
  protected readonly letters = ['A', 'B', 'C', 'D', 'E', 'F'];

  protected readonly byIndex = computed(() => new Map(this.answers().map((a) => [a.questionIndex, a])));
  protected readonly total = computed(() => this.quiz().questions.length);
  /** The questions not answered yet, in order. */
  protected readonly open = computed(() => {
    const answered = this.byIndex();
    return this.quiz().questions.map((_, i) => i).filter((i) => !answered.has(i));
  });
  protected readonly correct = computed(() => this.answers().filter((a) => a.correct).length);
  protected readonly wrong = computed(() => this.answers().filter((a) => !a.correct).map((a) => a.questionIndex));

  /** The question on screen; it opens on the first one not answered. */
  protected readonly shown = linkedSignal({
    source: this.quiz,
    computation: (q) => Math.max(0, q.questions.findIndex((_, i) => !q.answers.some((a) => a.questionIndex === i))),
  });
  /** The result instead of a question: at once when every question was answered before. */
  private readonly onResult = linkedSignal({
    source: this.quiz,
    computation: (q) => q.questions.length > 0 && q.answers.length >= q.questions.length,
  });
  /** The result is on screen. */
  readonly resultShown = computed(() => this.onResult() && this.open().length === 0);

  /** The option picked on the question on screen, as an index into its options (not its place). */
  protected readonly picked = computed(() => this.picks().get(this.shown()) ?? null);

  /**
   * Each question's options in the order shown, made again each time the questions are opened;
   * index is the option's place on the server, which is what an answer sends and gets back.
   */
  protected readonly shownOptions = computed(() =>
    this.quiz().questions.map((q) => optionOrder(q).map((index) => ({ index, text: q.options[index] }))),
  );

  /** The kind of the question on screen: what it asks, and the language of its texts. */
  protected readonly kind = computed(() => {
    const q = this.quiz().questions[this.shown()];
    const kind = q ? kindOf(q) : '';
    return {
      title: KIND_TITLE[kind],
      // A vocab question is asked in Vietnamese; tfn and vocab options are Vietnamese.
      promptLang: kind === 'vocab' ? 'vi' : 'en',
      optionLang: kind === 'tfn' || kind === 'vocab' ? 'vi' : 'en',
    };
  });

  /** The result by kind, in the order the kinds come: "Ý chính ✓ · Chi tiết ✓✓ …"; empty for older questions. */
  protected readonly byKind = computed(() => {
    const answered = this.byIndex();
    const groups = new Map<QuizKind, boolean[]>();
    this.quiz().questions.forEach((q, i) => {
      const kind = kindOf(q);
      const a = answered.get(i);
      if (kind && a) {
        groups.set(kind, [...(groups.get(kind) ?? []), a.correct]);
      }
    });
    return [...groups].map(([kind, marks]) => ({
      name: KIND_SHORT[kind],
      marks: marks.map((ok) => (ok ? '✓' : '✗')).join(''),
      label: `${KIND_SHORT[kind]}: ${marks.filter(Boolean).length}/${marks.length} đúng`,
    }));
  });

  /** One tab per question: its number and kind, and right or wrong once answered. */
  protected readonly tabs = computed(() => {
    const answered = this.byIndex();
    return this.quiz().questions.map((q, i) => {
      const a = answered.get(i);
      const state: 'open' | 'right' | 'wrong' = !a ? 'open' : a.correct ? 'right' : 'wrong';
      const kind = kindOf(q) ? ` · ${KIND_SHORT[kindOf(q)]}` : '';
      const label = `Câu ${i + 1}${kind}: ${{ open: 'chưa trả lời', right: 'đúng', wrong: 'sai' }[state]}`;
      return { state, label, prompt: q.prompt, kind: KIND_SHORT[kindOf(q)], vi: kindOf(q) === 'vocab' };
    });
  });

  /** The bar's main button on a question. */
  protected readonly main = computed<{ action: MainAction; label: string; disabled: boolean }>(() => {
    const i = this.shown();
    if (!this.byIndex().has(i)) {
      return {
        action: 'check',
        label: this.sending() ? 'Đang kiểm tra…' : 'Kiểm tra',
        // Not disabled while sending: the button keeps the focus (choose ignores a second press).
        disabled: this.picked() === null || this.changed(),
      };
    }
    if (i < this.total() - 1) {
      return { action: 'next', label: 'Câu tiếp theo', disabled: false };
    }
    const left = this.open().length;
    if (left > 0) {
      return { action: 'missing', label: `Làm câu còn thiếu (${left})`, disabled: false };
    }
    return { action: 'result', label: 'Xem kết quả', disabled: false };
  });

  /** "Bỏ qua, làm sau": on an open question, to the next one, else to another open one. */
  protected readonly skipTo = computed(() => {
    const i = this.shown();
    if (this.byIndex().has(i)) {
      return -1;
    }
    if (i < this.total() - 1) {
      return i + 1;
    }
    return this.open().find((k) => k !== i) ?? -1;
  });

  constructor() {
    effect(() => this.resultChange.emit(this.resultShown()));
  }

  protected pick(choice: number): void {
    const i = this.shown();
    this.picks.update((m) => new Map(m).set(i, choice));
  }

  protected onMain(): void {
    const m = this.main();
    switch (m.action) {
      case 'check':
        void this.choose(this.shown(), this.picked()!);
        break;
      case 'next':
        this.show(this.shown() + 1);
        break;
      case 'missing':
        this.show(this.open()[0]);
        break;
      case 'result':
        this.showResult();
        break;
    }
  }

  /** ← the question before; from the first one, back to the text; from the result, the last question. */
  protected goBack(): void {
    if (this.resultShown()) {
      this.show(this.total() - 1);
    } else if (this.shown() > 0) {
      this.show(this.shown() - 1);
    } else {
      this.back.emit();
    }
  }

  protected show(index: number): void {
    if (index < 0 || index >= this.total()) {
      return;
    }
    this.onResult.set(false);
    this.shown.set(index);
    this.moved(`#quiz-q-${index}`);
  }

  protected showResult(): void {
    this.onResult.set(true);
    this.moved('#quiz-result');
  }

  protected async choose(questionIndex: number, choice: number): Promise<void> {
    if (this.sending() || this.byIndex().has(questionIndex)) {
      return;
    }
    this.sending.set(true);
    this.error.set(null);
    try {
      const res = await firstValueFrom(
        this.api.answer(this.lessonId(), { version: this.quiz().version, questionIndex, choice }),
      );
      this.record(res.answer);
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { error?: string; answer?: QuizAnswer } | null) : null;
      if (body?.error === 'already_answered' && body.answer) {
        this.record(body.answer);
      } else if (body?.error === 'quiz_changed') {
        this.changed.set(true);
      } else {
        this.error.set(SEND_FAILED);
      }
    } finally {
      this.sending.set(false);
    }
  }

  /** Trả lời lại: the question opens again; the new answer replaces the stored one. */
  protected redo(index: number): void {
    this.forget([index]);
    this.show(index);
  }

  /** Làm lại các câu sai: the wrong ones open again, from the first of them. */
  protected redoWrong(): void {
    const wrong = this.wrong();
    if (wrong.length) {
      this.forget(wrong);
      this.announcement.set(`Làm lại ${wrong.length} câu sai.`);
      this.show(wrong[0]);
    }
  }

  /** Làm lại tất cả: every answer is forgotten on the server too, back to the first question. */
  protected async restart(): Promise<void> {
    if (this.sending()) {
      return;
    }
    this.sending.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(this.api.resetAnswers(this.lessonId()));
      this.answers.set([]);
      this.picks.set(new Map());
      this.announcement.set('Đã xoá các câu trả lời, làm lại từ câu 1.');
      this.show(0);
    } catch {
      this.error.set('Chưa làm lại được, vui lòng thử lại.');
    } finally {
      this.sending.set(false);
    }
  }

  private forget(indexes: number[]): void {
    const drop = new Set(indexes);
    this.answers.update((list) => list.filter((a) => !drop.has(a.questionIndex)));
    this.picks.update((m) => new Map([...m].filter(([i]) => !drop.has(i))));
  }

  /** After moving to another question or the result: back to its top, focus on what it asks. */
  private moved(selector: string): void {
    afterNextRender(
      () => {
        const target = this.host.nativeElement.querySelector<HTMLElement>(selector);
        this.host.nativeElement.ownerDocument.scrollingElement?.scrollTo?.({ top: 0 });
        target?.focus({ preventScroll: true });
      },
      { injector: this.injector },
    );
  }

  private record(answer: QuizAnswer): void {
    this.picks.update((m) => new Map([...m].filter(([i]) => i !== answer.questionIndex)));
    this.answers.update((list) =>
      [...list.filter((a) => a.questionIndex !== answer.questionIndex), answer].sort(
        (a, b) => a.questionIndex - b.questionIndex,
      ),
    );
    const done = this.open().length === 0;
    this.announcement.set(
      `Câu ${answer.questionIndex + 1}: ${answer.correct ? 'Đúng' : 'Sai'}.` +
        (done ? ' Đã trả lời hết, bấm Xem kết quả.' : ''),
    );
    if (done && !this.review()) {
      this.allAnswered.emit({ correct: this.correct(), total: this.total() });
    }
  }
}
