import { NgOptimizedImage } from '@angular/common';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  signal,
  viewChild,
  viewChildren,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AttemptResult, GrammarDetail, GrammarListItem, STATUS_LABELS } from '../../../core/models/grammar-study';
import { GrammarApiService } from '../../../core/services/grammar-api.service';
import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { SpeakButton } from '../../../shared/directives/speak-button';
import {
  autoMark,
  breakAnswer,
  breakAnswerText,
  guessStructure,
  PatternPart,
  patternRows,
  sentenceParts,
  splitLines,
  structureTone,
  tabLabels,
} from '../grammar-logic';
import { ExampleLine, ExampleLines } from '../example-lines/example-lines';
import { GrammarQuiz } from '../grammar-quiz/grammar-quiz';
import { MistakeCheck } from '../mistake-check/mistake-check';
import { Pattern } from '../pattern/pattern';
import { savedStep, saveStep, stepIndex, STEPS } from '../grammar-steps';

/** One way back into a point already studied: a button of the "Học tiếp từ chỗ dở" card. */
interface ResumeAction {
  label: string;
  step: number;
  /** Starts the round over the practice exercises answered wrong last time. */
  weak?: boolean;
}

/**
 * An example as shown: one line, or a question then its answer ("Are you twenty? – No, I'm not.")
 * each with its own meaning. `en` is the whole sentence, read aloud.
 */
interface ShownExample {
  en: string;
  lines: ExampleLine[];
}

/**
 * Coloured from its marks, or else from the words its structure's pattern names; without a pattern
 * the sentence stays plain.
 */
function shownExample(en: string, vi: string, marked?: string, parts: readonly PatternPart[] = []): ShownExample {
  const runs = marked ? sentenceParts(en, marked, parts) : autoMark(en, parts);
  const english = splitLines(breakAnswer(runs));
  const meanings = breakAnswerText(vi).split('\n');
  // The meanings go with their lines when both split alike, else all under the last line.
  const paired = meanings.length === english.length;
  return {
    en,
    lines: english.map((sentence, i) => ({
      sentence,
      vi: paired ? meanings[i] : i === english.length - 1 ? meanings.join(' ') : '',
    })),
  };
}

/** One card of Lý thuyết: the quick summary, one paragraph of the explanation, or the uses. */
type TheoryCard = { kind: 'quick' } | { kind: 'explain'; text: string; n: number } | { kind: 'usage' };

interface Resume {
  mastered: boolean;
  title: string;
  text: string;
  actions: ResumeAction[];
}

/**
 * One grammar point (F20) in four steps, framed like a lesson: Lý thuyết, Cấu trúc & ví dụ,
 * Luyện tập, then the mastery test. Nothing is locked: the "i" overview opens any step. A link may
 * open a step (?step=practice, with &redo=weak to redo the wrong answers at once); a point already
 * studied opens with a card leading straight to what is left to do.
 */
@Component({
  selector: 'lu-grammar-lesson',
  imports: [NgOptimizedImage, RouterLink, Icon, Loading, ExampleLines, GrammarQuiz, MistakeCheck, Pattern, SpeakButton],
  templateUrl: './grammar-lesson.html',
  styleUrl: './grammar-lesson.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLesson {
  private readonly api = inject(GrammarApiService);
  private readonly speech = inject(SpeechService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);

  private readonly stepTitles = viewChildren<ElementRef<HTMLElement>>('stepTitle');
  private readonly cardTitle = viewChild<ElementRef<HTMLElement>>('cardTitle');
  private readonly practiceQuiz = viewChild<GrammarQuiz>('practiceQuiz');
  private readonly mistakeCheck = viewChild(MistakeCheck);

  protected readonly steps = STEPS;
  protected readonly statusLabels = STATUS_LABELS;
  protected readonly canSpeak = this.speech.supported;
  protected readonly detail = signal<GrammarDetail | null>(null);
  /** 'notFound' when the point has no published lesson; 'failed' for any other error. */
  protected readonly error = signal<'notFound' | 'failed' | null>(null);
  /** Index of the step on screen in STEPS. */
  protected readonly step = signal(0);
  protected readonly overview = signal(false);
  protected readonly pointId = signal('');
  /** The point to study after this one, known once the mastery test is passed. */
  protected readonly nextPoint = signal<GrammarListItem | null>(null);
  /** Index of the Lý thuyết card on screen. */
  protected readonly card = signal(0);
  /** Index of the structure tab on screen in Cấu trúc & ví dụ. */
  protected readonly tab = signal(0);
  /** The step last opened on this point in this browser (0 = none), read when it opens. */
  private readonly leftAt = signal(0);

  /** Practice exercises answered wrong in the last practice round. */
  private readonly weakCount = computed(() => {
    const d = this.detail();
    return d ? d.content.practice.filter((e) => d.progress.weak.includes(e.id)).length : 0;
  });

  /** "Học tiếp từ chỗ dở" at the top of Lý thuyết for a point already practised, tested or opened. */
  protected readonly resume = computed<Resume | null>(() => {
    const p = this.detail()?.progress;
    if (!p) {
      return null;
    }
    const weak = this.weakCount();
    if (p.mastered) {
      return {
        mastered: true,
        title: `Đã nắm vững · ${p.bestMastery}%`,
        text: 'Muốn ôn nhanh? Làm lại bài kiểm tra hoặc luyện lại cả bài.',
        actions: [
          { label: 'Làm lại kiểm tra', step: 3 },
          { label: 'Luyện tập', step: 2 },
        ],
      };
    }
    if (p.practiceAttempts > 0 || p.masteryAttempts > 0) {
      const practice = p.practiceAttempts > 0 ? `Lần trước luyện tập: ${p.lastPractice}%` : 'Chưa luyện tập';
      const left = weak > 0 ? ` · còn ${weak} câu sai` : '';
      const test =
        p.masteryAttempts > 0 ? `Kiểm tra tốt nhất: ${p.bestMastery}% (cần 80%).` : 'Chưa làm bài kiểm tra.';
      return {
        mastered: false,
        title: 'Học tiếp từ chỗ dở',
        text: `${practice}${left}. ${test}`,
        actions:
          weak > 0
            ? [
                { label: `Luyện lại ${weak} câu sai`, step: 2, weak: true },
                { label: 'Làm kiểm tra', step: 3 },
              ]
            : [
                { label: 'Làm kiểm tra', step: 3 },
                { label: 'Luyện tập', step: 2 },
              ],
      };
    }
    const at = this.leftAt();
    return at > 0
      ? {
          mastered: false,
          title: 'Học tiếp từ chỗ dở',
          text: `Lần trước bạn dừng ở Bước ${at + 1} · ${STEPS[at].label}.`,
          actions: [{ label: `Tiếp tục Bước ${at + 1}`, step: at }],
        }
      : null;
  });

  protected readonly stepKey = computed(() => STEPS[this.step()].key);
  /**
   * Lý thuyết one card at a time: "Nhớ nhanh" (the first pattern and example) when there is one,
   * each paragraph of the explanation, then the uses.
   */
  protected readonly theoryCards = computed<TheoryCard[]>(() => {
    const c = this.detail()?.content;
    if (!c) {
      return [];
    }
    const cards: TheoryCard[] = [];
    if (this.quick()) {
      cards.push({ kind: 'quick' });
    }
    c.explanation.forEach((text, i) => cards.push({ kind: 'explain', text, n: i + 1 }));
    if (c.usage.length > 0) {
      cards.push({ kind: 'usage' });
    }
    return cards;
  });
  /**
   * The 1-based structure each example shows, 0 for none. Lessons written before 2026-10-30 do not
   * say: the structure is then guessed from the form of the sentence and the labels.
   */
  private readonly exampleStructures = computed(() => {
    const examples = this.detail()?.content.examples ?? [];
    const structures = this.structures();
    const named = examples.some((e) => e.structure);
    const labels = structures.map((s) => s.label);
    return examples.map((e) => {
      const n = named ? (e.structure ?? 0) : guessStructure(e.en, labels);
      return n <= structures.length ? n : 0;
    });
  });
  /**
   * Cấu trúc & ví dụ as one tab per structure, with its examples coloured like its pattern when
   * marked; the structure's own example stands in when none shows it.
   */
  protected readonly tabs = computed(() => {
    const content = this.detail()?.content;
    const of = this.exampleStructures();
    const labels = this.structures().map((s) => s.label);
    const short = tabLabels(labels);
    return this.structures().map((s, i) => {
      const own = (content?.examples ?? [])
        .filter((_, k) => of[k] === i + 1)
        .map((e) => shownExample(e.en, e.vi, e.marked, s.parts));
      // "Hay sai": the first common mistake whose right sentence has the form of this structure.
      const mistake = content?.mistakes.find((m) => guessStructure(m.right, labels) === i + 1) ?? null;
      return {
        ...s,
        short: short[i],
        examples: own.length > 0 ? own : [shownExample(s.example, '', undefined, s.parts)],
        mistake,
      };
    });
  });
  /**
   * Cấu trúc & ví dụ shows the tabs one at a time, then the common mistakes on their own screen
   * (`tab` one past the last structure).
   */
  protected readonly hasMistakes = computed(() => (this.detail()?.content.mistakes.length ?? 0) > 0);
  protected readonly onMistakes = computed(() => this.hasMistakes() && this.tab() >= this.tabs().length);
  /** The examples that show no structure, listed under the tabs. */
  protected readonly otherExamples = computed(() => {
    const of = this.exampleStructures();
    return (this.detail()?.content.examples ?? []).filter((_, k) => of[k] === 0).map((e) => shownExample(e.en, e.vi));
  });
  /** "Ví dụ khác" next to examples shown in the tabs, plain "Ví dụ" when all of them are below. */
  protected readonly otherTitle = computed(() =>
    this.otherExamples().length === this.detail()?.content.examples.length ? 'Ví dụ' : 'Ví dụ khác',
  );
  /**
   * The first structure and an example for "Nhớ nhanh": one of its own, coloured when marked, else
   * the first example; null with neither.
   */
  protected readonly quick = computed(() => {
    const s = this.structures()[0];
    if (s && this.exampleStructures().includes(1)) {
      return { rows: s.rows, ...this.tabs()[0].examples[0] };
    }
    const ex = this.detail()?.content.examples[0];
    if (!s && !ex) {
      return null;
    }
    return { rows: s?.rows ?? [], ...shownExample(ex?.en ?? s?.example ?? '', ex?.vi ?? '', undefined, s?.parts) };
  });
  protected readonly lastCard = computed(() => this.card() >= this.theoryCards().length - 1);
  /** On the last screen of Cấu trúc & ví dụ: the mistakes, or the last tab when there are none. */
  protected readonly lastTab = computed(() => this.tab() >= this.tabs().length - (this.hasMistakes() ? 0 : 1));
  /** On the mistakes screen, a mistake not answered yet, or answered but with others after it. */
  private readonly mistakeLeft = computed(() => {
    const check = this.mistakeCheck();
    return this.onMistakes() && !!check && (!check.answered() || !check.isLast());
  });
  protected readonly canGoBack = computed(() => this.step() > 0 || this.card() > 0);
  protected readonly backLabel = computed(() => {
    if (this.step() === 0) {
      return 'Thẻ trước';
    }
    return this.step() === 1 && this.tab() > 0 ? 'Cấu trúc trước' : 'Bước trước';
  });
  /** The bar's button on Lý thuyết and Cấu trúc (the quizzes have their own). */
  protected readonly barLabel = computed(() => {
    if (this.step() === 0 && !this.lastCard()) {
      return this.theoryCards()[this.card()]?.kind === 'quick' ? 'Đọc giải thích' : 'Thẻ tiếp';
    }
    if (this.step() === 1 && this.mistakeLeft()) {
      return 'Câu tiếp';
    }
    if (this.step() === 1 && !this.lastTab()) {
      const next = this.tabs()[this.tab() + 1];
      return next ? `Xem ${next.short}` : 'Ôn lỗi thường gặp';
    }
    const next = STEPS[this.step() + 1];
    return this.step() < 2 && next ? `Sang ${next.label}` : '';
  });
  /** The bar waits for a sentence to be picked on the mistakes screen. */
  protected readonly barDisabled = computed(() => this.mistakeLeft() && !this.mistakeCheck()?.answered());
  protected readonly testContinue = computed(() => {
    const n = this.nextPoint();
    return n ? `Học tiếp: ${n.titleVi}` : 'Về danh sách ngữ pháp';
  });
  /**
   * The structures with their pattern in lines of coloured pieces (`rows`), the pieces counted by
   * the marks (`parts`), and the tone of their label.
   */
  protected readonly structures = computed(() => {
    const list = this.detail()?.content.structures ?? [];
    const rows = patternRows(list.map((s) => s.pattern));
    return list.map((s, i) => ({
      ...s,
      rows: rows[i],
      parts: rows[i].filter((r) => !r.answer).flatMap((r) => r.parts),
      tone: structureTone(s.label),
    }));
  });

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((p) => {
      this.pointId.set(p.get('pointId') ?? '');
      this.load();
    });
  }

  protected load(): void {
    const query = this.route.snapshot?.queryParamMap;
    const linked = stepIndex(query?.get('step') ?? null);
    const redoWeak = linked === 2 && query?.get('redo') === 'weak';
    this.detail.set(null);
    this.error.set(null);
    this.step.set(linked ?? 0);
    this.card.set(0);
    this.tab.set(0);
    this.overview.set(false);
    this.nextPoint.set(null);
    this.leftAt.set(savedStep(this.pointId()));
    this.api.get(this.pointId()).subscribe({
      next: (d) => {
        this.detail.set(d);
        if (redoWeak) {
          afterNextRender(() => this.practiceQuiz()?.startWeak(), { injector: this.injector });
        }
      },
      error: (err: unknown) => this.error.set(err instanceof ApiError && err.status === 404 ? 'notFound' : 'failed'),
    });
  }

  /** Shows a step from its top, its title focused for screen readers. */
  protected goTo(index: number): void {
    this.step.set(index);
    this.overview.set(false);
    saveStep(this.pointId(), index);
    afterNextRender(
      () => {
        document.scrollingElement?.scrollTo?.({ top: 0 });
        this.stepTitles()[index]?.nativeElement.focus({ preventScroll: true });
      },
      { injector: this.injector },
    );
  }

  protected resumeWith(action: ResumeAction): void {
    this.goTo(action.step);
    if (action.weak) {
      afterNextRender(() => this.practiceQuiz()?.startWeak(), { injector: this.injector });
    }
  }

  /** Shows a Lý thuyết card, its title focused. */
  protected showCard(index: number): void {
    this.card.set(index);
    afterNextRender(() => this.cardTitle()?.nativeElement.focus(), { injector: this.injector });
  }

  /** Shows a structure tab; with `focus` (arrow keys) the tab itself takes the focus. */
  protected showTab(index: number, focus = false): void {
    this.tab.set(index);
    // The tabs scroll sideways on a narrow screen: keep the shown one in sight.
    afterNextRender(
      () => {
        const tab = document.getElementById(`gtab-${index}`);
        tab?.scrollIntoView?.({ block: 'nearest', inline: 'nearest' });
        if (focus) {
          tab?.focus();
        }
      },
      { injector: this.injector },
    );
  }

  /** Arrow keys, Home and End move between the structure tabs. */
  protected onTabKey(event: KeyboardEvent): void {
    const last = this.tabs().length - 1;
    const to: Record<string, number> = {
      ArrowRight: this.tab() < last ? this.tab() + 1 : 0,
      ArrowLeft: this.tab() > 0 ? this.tab() - 1 : last,
      Home: 0,
      End: last,
    };
    if (event.key in to) {
      event.preventDefault();
      this.showTab(to[event.key], true);
    }
  }

  /** Through the Lý thuyết cards and the structure tabs, then to the next step. */
  protected next(): void {
    if (this.step() === 0 && !this.lastCard()) {
      this.showCard(this.card() + 1);
      return;
    }
    if (this.step() === 1 && this.mistakeLeft()) {
      this.mistakeCheck()?.next();
      return;
    }
    if (this.step() === 1 && !this.lastTab()) {
      this.showTab(this.tab() + 1);
      return;
    }
    if (this.step() === 0) {
      this.tab.set(0);
    }
    this.goTo(Math.min(this.step() + 1, STEPS.length - 1));
  }

  /** Back through the cards and tabs; a step left backwards opens on its last card or tab. */
  protected back(): void {
    if (this.step() === 0) {
      this.showCard(Math.max(this.card() - 1, 0));
      return;
    }
    if (this.step() === 1 && this.tab() > 0) {
      this.showTab(this.tab() - 1);
      return;
    }
    if (this.step() === 1) {
      this.card.set(Math.max(this.theoryCards().length - 1, 0));
    }
    if (this.step() === 2) {
      this.tab.set(Math.max(this.tabs().length - (this.hasMistakes() ? 0 : 1), 0));
    }
    this.goTo(this.step() - 1);
  }

  protected toggleOverview(): void {
    this.overview.update((o) => !o);
  }

  protected listen(text: string): void {
    this.speech.speak(text, 0.9);
  }

  protected onRecorded(result: AttemptResult): void {
    this.detail.update((d) => (d ? { ...d, progress: result.progress } : d));
  }

  /** After a passed test, finds the next point of the level for the "Học tiếp" button. */
  protected onTestRecorded(result: AttemptResult): void {
    this.onRecorded(result);
    const d = this.detail();
    if (!result.passed || !d) {
      return;
    }
    this.api.points(d.point.level).subscribe({
      next: (list) => {
        const n = list.next && list.next !== d.point.id ? list.points.find((p) => p.id === list.next) : undefined;
        this.nextPoint.set(n ?? null);
      },
      // Without it the button leads back to the list.
      error: () => this.nextPoint.set(null),
    });
  }

  protected afterTest(): void {
    const n = this.nextPoint();
    void this.router.navigate(n ? ['/grammar', n.id] : ['/grammar']);
  }
}
