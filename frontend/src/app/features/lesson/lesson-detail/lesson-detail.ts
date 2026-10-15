import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  InjectionToken,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, firstValueFrom, forkJoin, Observable, of, Subscription } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { PracticeView } from '../../../core/models/practice';
import { ReadingLesson } from '../../../core/models/reading';
import { LessonStudy, Step, STEPS } from '../../../core/models/study';
import { LessonVocabulary } from '../../../core/models/vocab';
import { formatScore, Writing as SubmittedWriting } from '../../../core/models/writing';
import { SpeechService } from '../../../core/services/speech.service';
import { StudyApiService } from '../../../core/services/study-api.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Listening } from '../listening/listening';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { Reading } from '../reading/reading';
import { Writing } from '../writing/writing';
import { DialogueStep } from './dialogue-step/dialogue-step';
import { FillStep } from './fill-step/fill-step';
import {
  hasDialogue,
  hasFill,
  hasTranslations,
  levelLabel,
  newSeed,
  Score,
  shuffle,
  wordsIn,
} from './practice-logic';
import { PracticeList } from './practice-list/practice-list';
import { PracticeSummary } from './practice-summary/practice-summary';
import { ReadingText } from './reading-text/reading-text';
import { SpeakStep } from './speak-step/speak-step';
import { TranslateStep } from './translate-step/translate-step';
import { VocabStep } from './vocab-step/vocab-step';
import { Loading } from '../../../shared/components/loading/loading';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { ListeningApiService } from '../listening-api.service';

type Tab = 'lesson' | 'reading';

/** The practice steps (shown when the lesson has their content), then the lesson's own steps. */
type PracticeKey = 'words' | 'dialogue' | 'fill' | 'translate' | 'speak';
type StepKey = PracticeKey | Step;
/** The practice a lesson being studied keeps for later (Luyện thêm), outside its main steps. */
export type ExtraKey = 'dialogue' | 'fill' | 'translate';

const STEP_NAMES: Record<StepKey, string> = {
  words: 'Từ vựng',
  dialogue: 'Hội thoại',
  fill: 'Điền ô trống',
  translate: 'Dịch câu',
  speak: 'Luyện nói',
  read: 'Đọc',
  listen: 'Nghe',
  write: 'Viết',
};

/** About how long each step takes, in minutes, for the estimate under the lesson's title. */
const STEP_MINUTES: Record<StepKey, number> = {
  words: 3,
  dialogue: 2,
  fill: 3,
  translate: 3,
  speak: 2,
  read: 4,
  listen: 4,
  write: 4,
};

/** One practice of the lesson in a summary: what was done, or null when it was not done yet. */
export interface PracticeItem {
  key: StepKey;
  name: string;
  result: string | null;
}

/** Whether lessons have the Speaking step (Luyện nói); specs of the other steps turn it off. */
export const SPEAKING_PRACTICE = new InjectionToken<boolean>('SPEAKING_PRACTICE', {
  providedIn: 'root',
  factory: () => true,
});

/** Sentences of the Speaking step: short ones, a few of them. */
const SPEAK_MAX_WORDS = 15;
const SPEAK_COUNT = 6;

/** Cards due from which the done card says the backlog is large. */
const BACKLOG = 30;
/** About how long one card takes to review, in seconds. */
const SECONDS_PER_CARD = 20;

/** Where the learner stopped is kept on this device, since the server does not track the practice steps. */
const STEP_KEY = (lessonId: string) => `luna.lesson-step.${lessonId}`;

function readSavedStep(lessonId: string): string | null {
  try {
    return localStorage.getItem(STEP_KEY(lessonId));
  } catch {
    return null;
  }
}

function writeSavedStep(lessonId: string, step: string | null): void {
  try {
    if (step === null) {
      localStorage.removeItem(STEP_KEY(lessonId));
    } else {
      localStorage.setItem(STEP_KEY(lessonId), step);
    }
  } catch {
    // Storage can be blocked (private window): the lesson then resumes from the server state alone.
  }
}

/** Delay before saving the reading or listening position (debounce 1 s). */
const POSITION_DELAY = 1000;

function isStudyStep(key: StepKey | null): key is Step {
  return key !== null && (STEPS as readonly string[]).includes(key);
}

/**
 * A lesson's page (F17, L): header, then the Bài học tab with the practice steps that have content
 * (words, dialogue, fill-in, translation) or the Bài đọc tab with the text and grammar note.
 *
 * The lesson being studied is learnt here (updated 2026-10-14): five main steps, words → Đọc → Nghe →
 * Luyện nói → Viết (optional). Đọc, Nghe and Viết are recorded on the server. The practice on the
 * text (dialogue, fill-in, translation) is "Luyện thêm": optional, opened from the lesson's details
 * or the done card, back to where the learner was once finished. Each step moves on with its own
 * button, so there is one way forward; the details list every step to go back to one. Once done,
 * "Sang bài tiếp theo" opens the next lesson at once. Other lessons keep the practice only, ending
 * on its summary. Practice results stay on this page.
 */
@Component({
  selector: 'lu-lesson-detail',
  imports: [Loading, ConfirmDialog, 
    DialogueStep,
    FillStep,
    Icon,
    Listening,
    PracticeList,
    PracticeSummary,
    ReadingText,
    Reading,
    RouterLink,
    SpeakStep,
    TranslateStep,
    VocabStep,
    Writing,
  ],
  templateUrl: './lesson-detail.html',
  styleUrl: './lesson-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonDetail {
  private readonly reading = inject(ReadingApiService);
  private readonly studyApi = inject(StudyApiService);
  private readonly router = inject(Router);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly speech = inject(SpeechService);
  private readonly vocabApi = inject(VocabApiService);
  private readonly listeningApi = inject(ListeningApiService);
  private readonly speakingOn = inject(SPEAKING_PRACTICE);
  private readonly writingApi = inject(WritingApiService);

  protected readonly id = signal('');
  protected readonly lesson = signal<ReadingLesson | null>(null);
  /** Back to the path of the lesson's part, where the learner came from. */
  protected readonly backLink = computed(() => {
    const level = this.lesson()?.level;
    return level ? ['/lessons/levels', level] : ['/lessons'];
  });
  protected readonly vocabulary = signal<LessonVocabulary | null>(null);
  protected readonly practice = signal<PracticeView | null>(null);
  protected readonly study = signal<LessonStudy | null>(null);
  protected readonly loadError = signal<string | null>(null);
  /** Whether the page leads through Đọc → Nghe → Viết: set once, for the lesson being studied. */
  protected readonly studyFlow = signal(false);
  /**
   * A completed lesson learnt again from the start ("Học lại bài"): every step, the latest results
   * count, the lesson's progress does not change.
   */
  protected readonly relearn = signal(false);
  protected readonly relearnAsk = signal(false);
  protected readonly relearnError = signal(false);
  /** The lesson was completed on this page. */
  protected readonly lessonDone = signal(false);
  protected readonly saving = signal(false);
  /** Cards due now, shown once the lesson is done so the learner can review before the next one. */
  protected readonly dueCount = signal(0);
  protected readonly backlog = BACKLOG;
  /** The writing submitted in this lesson, linked from the done card; null when it was skipped. */
  protected readonly doneWriting = signal<SubmittedWriting | null>(null);
  protected readonly formatScore = formatScore;
  protected readonly dueMinutes = computed(() =>
    Math.max(1, Math.ceil((this.dueCount() * SECONDS_PER_CARD) / 60)),
  );
  /** Words got wrong in this lesson's practice that were put in the review queue (one per word). */
  private readonly missedSent = new Set<string>();
  protected readonly missCount = signal(0);
  protected readonly missNote = computed(() => {
    const n = this.missCount();
    return n > 0 ? `Đã đưa ${n} từ làm sai vào lịch ôn.` : null;
  });
  protected readonly stepError = signal<string | null>(null);

  protected readonly tab = signal<Tab>('lesson');
  /**
   * The lesson's overview (steps and their state, time, objective, Luyện thêm) instead of the
   * step: shown when a lesson being learnt opens, and from "i".
   */
  protected readonly overview = signal(false);
  /** The Luyện thêm practice on screen, over the step the learner was on. */
  protected readonly extra = signal<ExtraKey | null>(null);
  /** Practice gone through to its end on this page (the dialogue and speaking have no score). */
  protected readonly practiced = signal<ReadonlySet<StepKey>>(new Set());
  /** Position (from 1) in `steps`. */
  protected readonly step = signal(1);
  protected readonly translateIndex = signal(0);
  protected readonly fillScore = signal<Score | null>(null);
  protected readonly translateResults = signal<(boolean | null)[]>([]);
  protected readonly finished = signal(false);
  protected readonly rate = signal(1);
  private readonly seed = signal(newSeed());

  protected readonly status = computed(() => this.study()?.status ?? 'other');
  protected readonly streak = computed(() => this.study()?.streak ?? null);
  protected readonly level = computed(() => {
    const l = this.lesson();
    return l ? levelLabel(l.level) : '';
  });
  protected readonly words = computed(() => {
    const v = this.vocabulary();
    return v?.available ? v.items : [];
  });
  protected readonly hasDialogue = computed(() => hasDialogue(this.practice()));
  protected readonly hasFill = computed(() => hasFill(this.practice()));
  protected readonly hasTranslations = computed(() => hasTranslations(this.practice()));
  /**
   * Sentences to repeat in the Speaking step: the dialogue's turns (short, spoken English), else the
   * short sentences of the text.
   */
  protected readonly speakSentences = computed(() => {
    const turns = this.practice()?.dialogue?.turns.map((t) => t.text) ?? [];
    const source = turns.length > 0 ? turns : (this.lesson()?.sentences ?? []).map((s) => s.text);
    return source
      .map((t) => t.trim())
      .filter((t) => t && t.split(/\s+/).length <= SPEAK_MAX_WORDS)
      .slice(0, SPEAK_COUNT);
  });

  /** The lesson goes through its main steps (studied, or learnt again), the practice kept for later. */
  protected readonly learning = computed(() => this.studyFlow() || this.relearn());

  /**
   * Steps with content, in order. A lesson being studied goes words → Đọc → Nghe → Luyện nói →
   * Viết, its practice being Luyện thêm; any other lesson has the practice only, Luyện nói last.
   */
  protected readonly steps = computed<StepKey[]>(() => {
    const words: StepKey[] = this.words().length > 0 ? ['words'] : [];
    const practice: StepKey[] = [];
    if (this.hasDialogue()) {
      practice.push('dialogue');
    }
    if (this.hasFill()) {
      practice.push('fill');
    }
    if (this.hasTranslations()) {
      practice.push('translate');
    }
    const speak: StepKey[] = this.speakingOn && this.speakSentences().length > 0 ? ['speak'] : [];
    return this.learning()
      ? [...words, 'read', 'listen', ...speak, 'write']
      : [...words, ...practice, ...speak];
  });
  /** Luyện thêm of a lesson being learnt: its practice with content. */
  protected readonly extras = computed<ExtraKey[]>(() => {
    if (!this.learning()) {
      return [];
    }
    const keys: ExtraKey[] = [];
    if (this.hasDialogue()) {
      keys.push('dialogue');
    }
    if (this.hasFill()) {
      keys.push('fill');
    }
    if (this.hasTranslations()) {
      keys.push('translate');
    }
    return keys;
  });
  /**
   * Luyện thêm opens once Nghe is done: the practice is made of the lesson's text and dialogue, so
   * before Đọc and Nghe it is guesswork and gives their answers away. A lesson learnt again, or
   * finished, has it open at once, and so has the summary at the end of the steps.
   */
  protected readonly extrasOpen = computed(
    () =>
      !this.studyFlow() ||
      this.relearn() ||
      this.lessonDone() ||
      this.finished() ||
      this.study()?.steps.listen === 'done',
  );
  /** On Viết still to do, once Luyện thêm is open: a quiet offer to practise before writing. */
  protected readonly offerExtras = computed(
    () =>
      this.extras().length > 0 &&
      this.extrasOpen() &&
      !this.extra() &&
      !this.overview() &&
      this.current() === 'write' &&
      !this.studyStepDone(),
  );
  /** "5 bước · khoảng 17 phút" in the lesson's details. */
  protected readonly estimate = computed(() => {
    const steps = this.steps();
    const minutes = steps.reduce((sum, k) => sum + STEP_MINUTES[k], 0);
    return steps.length ? `${steps.length} bước · khoảng ${minutes} phút` : '';
  });
  /** The step on screen: a Luyện thêm practice when one is open, else the current step. */
  protected readonly onScreen = computed<StepKey | null>(() => this.extra() ?? this.current());
  protected readonly hasSteps = computed(() => this.steps().length > 0);
  protected readonly current = computed<StepKey | null>(
    () => this.steps()[this.step() - 1] ?? null,
  );
  /** The lesson's step on screen, if it is one, and whether it is done or still to do. */
  protected readonly studyStep = computed<Step | null>(() => {
    const key = this.current();
    return isStudyStep(key) ? key : null;
  });
  protected readonly studyStepDone = computed(() => {
    const s = this.studyStep();
    if (this.relearn()) {
      // Learning a finished lesson again: Đọc and Nghe are done again, the writing is only shown.
      return s === 'write';
    }
    return !!s && (this.lessonDone() || this.study()?.steps[s] === 'done');
  });
  /** "Học lại bài" is offered on a completed lesson that is not being learnt again already. */
  protected readonly canRelearn = computed(() => this.status() === 'completed' && !this.relearn());
  /** The tab on screen: always Bài đọc when there is nothing to practise. */
  protected readonly shownTab = computed<Tab>(() => (this.hasSteps() ? this.tab() : 'reading'));

  /** Word banks are shuffled from the seed; Làm lại picks a new seed. */
  protected readonly fillBank = computed(() =>
    shuffle(this.practice()?.fill?.wordBank ?? [], this.seed()),
  );
  protected readonly translationTiles = computed(() =>
    (this.practice()?.translations ?? []).map((t, i) => shuffle(t.tiles, this.seed() + i + 1)),
  );
  protected readonly turnTexts = computed(
    () => this.practice()?.dialogue?.turns.map((t) => t.text) ?? [],
  );

  protected readonly progress = computed(() =>
    this.finished() || this.lessonDone() ? this.steps().length : this.step(),
  );
  protected readonly stepAnnouncement = computed(() => {
    if (this.lessonDone()) {
      return 'Hoàn thành bài';
    }
    if (this.finished()) {
      return 'Tổng kết';
    }
    const key = this.current();
    return key ? `Bước ${this.step()}/${this.steps().length}: ${STEP_NAMES[key]}` : '';
  });
  protected readonly nextLabel = computed(() => (this.isLast() ? 'Hoàn thành' : 'Tiếp theo'));
  /** The name of the step after the current one, for "Sang bước Nghe". */
  protected readonly nextName = computed(() => {
    const key = this.steps()[this.step()];
    return key ? STEP_NAMES[key] : '';
  });
  /**
   * The finishing button of a step: back to the lesson from Luyện thêm, else on to the next step by
   * name, or Hoàn thành on the last one.
   */
  protected readonly finishLabel = computed(() => {
    if (this.extra()) {
      return 'Xong, về bài học';
    }
    return this.isLast() ? 'Hoàn thành' : `Sang bước ${this.nextName()}`;
  });
  /** True on a lesson step still to do: moving on leaves it for later. */
  protected readonly skipping = computed(() => !!this.studyStep() && !this.studyStepDone());
  /** Some of the lesson is done (learning it again starts over): "Tiếp tục", else "Bắt đầu". */
  private readonly started = computed(
    () => this.step() > 1 || (this.studyFlow() && !this.relearn() && STEPS.some((s) => this.study()?.steps[s] === 'done')),
  );
  /**
   * ← in the bar: back from Luyện thêm, to the step before, or from the first step to the
   * overview of a lesson being learnt.
   */
  protected readonly canBack = computed(
    () => !this.overview() && (!!this.extra() || this.step() > 1 || this.learning()),
  );
  /** Đọc and the practice steps put their own buttons in the bar (← and Kiểm tra, Câu tiếp, …). */
  protected readonly ownBar = computed(() => {
    const key = this.onScreen();
    return (
      !this.overview() &&
      (key === 'read' || key === 'dialogue' || key === 'fill' || key === 'translate' || key === 'speak')
    );
  });
  /**
   * The bar's main button where the step has none of its own: start or go on from the overview,
   * leave a lesson step still to do for later (not the last one), or go on from a step done.
   * Null on the words (their card has its own) and on Viết still to do (Nộp, Bỏ qua).
   */
  protected readonly barMain = computed<{ label: string; leave: boolean } | null>(() => {
    if (this.overview()) {
      const key = this.current();
      return key ? { label: `${this.started() ? 'Tiếp tục' : 'Bắt đầu'}: ${STEP_NAMES[key]}`, leave: false } : null;
    }
    if (this.extra() || !this.studyStep()) {
      return null;
    }
    if (this.studyStepDone()) {
      return { label: this.isLast() ? 'Hoàn thành' : `Sang bước ${this.nextName()}`, leave: false };
    }
    return this.isLast() ? null : { label: `Để sau, sang bước ${this.nextName()}`, leave: true };
  });
  /**
   * The bar at the bottom of the lesson: ← and the main button (design, 2026-10-14). The practice
   * steps render theirs with their own buttons; the done card and the summary have their own.
   */
  protected readonly showBar = computed(
    () =>
      this.shownTab() === 'lesson' &&
      this.hasSteps() &&
      !this.ownBar() &&
      (!!this.extra() || (!this.finished() && !this.lessonDone())) &&
      (this.canBack() || this.barMain() !== null),
  );
  private readonly isLast = computed(() => this.step() >= this.steps().length);
  /**
   * Lesson steps left for later, once the learner is on the last step: the lesson completes only
   * when every one is done, in any order.
   */
  protected readonly leftBehind = computed<Step[]>(() => {
    const study = this.study();
    if (!this.studyFlow() || this.lessonDone() || this.extra() || !study || !this.isLast()) {
      return [];
    }
    const current = this.studyStep();
    return STEPS.filter((s) => s !== current && study.steps[s] !== 'done');
  });
  protected readonly stepNames = STEP_NAMES;

  /**
   * The practice in the summary and in Luyện thêm: a score or what was done for those gone through,
   * null for those not done yet (never "0/5" for a step that was skipped).
   */
  protected readonly practiceItems = computed<PracticeItem[]>(() => {
    const keys: StepKey[] = this.learning() ? this.extras() : this.steps().filter((k) => k !== 'words');
    return keys.map((key) => ({ key, name: STEP_NAMES[key], result: this.practiceResult(key) }));
  });

  private positionTimer?: ReturnType<typeof setTimeout>;
  private loading?: Subscription;

  /** True once the saved step has been applied: before that, the step on screen must not overwrite it. */
  private readonly restored = signal(false);

  constructor() {
    // With the natural voice, generate everything the lesson may read aloud, in the order of the steps.
    effect(() => {
      const words = this.words();
      const lesson = this.lesson();
      const practice = this.practice();
      if (!this.speech.natural()) {
        return;
      }
      this.speech.prefetch([
        ...words.map((w) => w.lemma),
        ...(practice?.examples ?? []).map((e) => e.sentence),
        ...(lesson?.sentences ?? []).map((s) => s.text),
        ...(practice?.dialogue?.turns ?? []).map((t) => t.text),
        ...(practice?.translations ?? []).map((t) => t.answer.join(' ')),
      ]);
    });
    // Remember the step on screen so that coming back lands on it, not on the first step or on Viết.
    effect(() => {
      const key = this.steps()[this.step() - 1];
      if (!this.restored() || !this.studyFlow() || !key) {
        return;
      }
      writeSavedStep(this.id(), this.lessonDone() ? null : key);
    });
    inject(DestroyRef).onDestroy(() => {
      this.speech.stop();
      clearTimeout(this.positionTimer);
    });
    // "Sang bài tiếp theo" stays on this page: the lesson changes with the route.
    inject(ActivatedRoute)
      .paramMap.pipe(takeUntilDestroyed())
      .subscribe((params) => this.open(params.get('id') ?? ''));
  }

  private open(id: string): void {
    this.speech.stop();
    clearTimeout(this.positionTimer);
    this.loading?.unsubscribe();
    this.id.set(id);
    this.lesson.set(null);
    this.loadError.set(null);
    this.study.set(null);
    this.studyFlow.set(false);
    this.restored.set(false);
    this.lessonDone.set(false);
    this.relearn.set(false);
    this.relearnAsk.set(false);
    this.relearnError.set(false);
    this.dueCount.set(0);
    this.doneWriting.set(null);
    this.missedSent.clear();
    this.missCount.set(0);
    this.stepError.set(null);
    this.tab.set('lesson');
    this.overview.set(false);
    this.extra.set(null);
    this.practiced.set(new Set());
    this.step.set(1);
    this.translateIndex.set(0);
    this.fillScore.set(null);
    this.finished.set(false);
    this.seed.set(newSeed());

    this.loading = forkJoin({
      lesson: this.reading.getLesson(id),
      vocabulary: this.reading.vocabulary(id).pipe(catchError(() => of(null))),
      practice: this.reading.practice(id).pipe(catchError(() => of(null))),
      study: this.studyApi.lessonStudy(id).pipe(catchError(() => of(null))),
    }).subscribe({
      next: (r) => {
        this.vocabulary.set(r.vocabulary);
        this.practice.set(r.practice);
        this.study.set(r.study);
        this.translateResults.set((r.practice?.translations ?? []).map(() => null));
        this.studyFlow.set(r.study?.status === 'studying');
        this.resumeStep(r.study);
        this.restoreSavedStep();
        this.restored.set(true);
        // A lesson being learnt opens on its overview, the bar going on from where the learner is.
        this.overview.set(this.learning() && this.hasSteps());
        this.lesson.set(r.lesson);
      },
      error: (err: unknown) => this.loadError.set(loadErrorMessage(err)),
    });
  }

  /** The furthest of the server state and the step saved on this device (the practice is only known here). */
  private restoreSavedStep(): void {
    if (!this.studyFlow()) {
      return;
    }
    const saved = readSavedStep(this.id());
    const steps = this.steps();
    const index = saved ? steps.indexOf(saved as StepKey) : -1;
    // The saved step may be behind where the server puts the learner (the practice sits before
    // Viết), but never behind a step the server knows is done: the floor is the one after the last.
    const doneAt = STEPS.filter((s) => this.study()?.steps[s] === 'done').map((s) => steps.indexOf(s));
    const floor = Math.max(-1, ...doneAt) + 1;
    if (index >= floor) {
      this.step.set(index + 1);
    }
  }

  /** Back to the lesson's step in progress, when one was done or a position saved. */
  private resumeStep(study: LessonStudy | null): void {
    if (study?.status !== 'studying' || !isStudyStep(study.currentStep as StepKey)) {
      return;
    }
    const started = STEPS.some((s) => study.steps[s] === 'done') || study.sentenceIndex > 0;
    if (started) {
      // The server does not track the practice, so a learner who left after Nghe resumes on Viết.
      this.step.set(this.steps().indexOf(study.currentStep as Step) + 1);
    }
  }

  private practiceResult(key: StepKey): string | null {
    switch (key) {
      case 'fill': {
        const f = this.fillScore();
        return f ? `Đúng ${f.correct}/${f.total} ô` : null;
      }
      case 'translate': {
        const results = this.translateResults();
        const checked = results.filter((r) => r !== null).length;
        if (checked === 0) {
          return null;
        }
        const correct = results.filter((r) => r === true).length;
        return checked === results.length
          ? `Đúng ${correct}/${results.length} câu`
          : `Đã làm ${checked}/${results.length} câu`;
      }
      case 'dialogue':
        return this.practiced().has('dialogue') ? 'Đã nghe hết' : null;
      case 'speak':
        return this.practiced().has('speak') ? `Đã luyện ${this.speakSentences().length} câu` : null;
      default:
        return null;
    }
  }

  /** Reads a word or sentence of the practice with the browser's voice. */
  protected play(text: string): void {
    this.speech.speak(text, this.rate());
  }

  /** Reading goes on across tabs; the Bài học tab stays as it was. */
  /** "Bài đọc" in the title line: the text and its grammar note, then back to the lesson. */
  protected toggleReading(): void {
    this.tab.set(this.tab() === 'lesson' ? 'reading' : 'lesson');
  }

  /** "i": the overview of the lesson, or back to the step from it. */
  protected toggleOverview(): void {
    this.speech.stop();
    this.tab.set('lesson');
    this.overview.set(!this.overview());
    this.toTop();
  }

  /** ← in the bar: back from Luyện thêm, else to the step before, from the first to the overview. */
  protected goBack(): void {
    if (this.extra()) {
      this.closeExtra();
    } else if (this.step() > 1) {
      this.back();
    } else if (this.learning()) {
      this.speech.stop();
      this.overview.set(true);
      this.toTop();
    }
  }

  /** The bar's main button: start from the overview, else on to the next step. */
  protected barAction(): void {
    if (this.overview()) {
      this.overview.set(false);
      this.toTop();
    } else {
      this.next();
    }
  }

  protected onFillChecked(score: Score): void {
    this.fillScore.set(score);
  }

  protected onTranslationChecked(index: number, ok: boolean): void {
    this.translateResults.update((r) => r.map((v, i) => (i === index ? ok : v)));
    const answer = this.practice()?.translations[index]?.answer.join(' ') ?? '';
    if (!ok && answer) {
      this.sendMisses(wordsIn(answer, this.words()));
    }
  }

  /**
   * Words got wrong in the practice go to the review queue, due now. Only the lesson being studied
   * does this, once per word; a failure is ignored, the practice does not depend on it.
   */
  protected sendMisses(words: string[]): void {
    const fresh = words.filter((w) => !this.missedSent.has(w.toLowerCase()));
    if (!this.studyFlow() || fresh.length === 0) {
      return;
    }
    fresh.forEach((w) => this.missedSent.add(w.toLowerCase()));
    this.vocabApi.practiceMisses(this.id(), fresh).subscribe({
      next: (r) => this.missCount.update((n) => n + r.added + r.rescheduled),
      error: () => undefined,
    });
  }

  /** The end of a practice step: noted for the summary, then on, or back from Luyện thêm. */
  protected practiceDone(key: StepKey): void {
    this.practiced.update((set) => new Set(set).add(key));
    if (this.extra()) {
      this.closeExtra();
    } else {
      this.next();
    }
  }

  /** Opens a Luyện thêm practice over the step the learner is on. */
  protected openExtra(key: ExtraKey): void {
    if (!this.extrasOpen()) {
      return;
    }
    this.speech.stop();
    this.overview.set(false);
    this.tab.set('lesson');
    this.extra.set(key);
    this.toTop();
  }

  protected closeExtra(): void {
    this.speech.stop();
    this.extra.set(null);
    this.toTop();
  }

  /** A step picked in the summary or the details: Luyện thêm for the practice kept for later. */
  protected openStep(key: StepKey): void {
    if ((this.extras() as StepKey[]).includes(key)) {
      this.openExtra(key as ExtraKey);
      return;
    }
    this.overview.set(false);
    this.tab.set('lesson');
    this.extra.set(null);
    this.finished.set(false);
    this.goToStep(key);
  }

  protected next(): void {
    this.speech.stop();
    if (this.step() < this.steps().length) {
      this.step.update((s) => s + 1);
    } else {
      this.finished.set(true);
    }
    this.toTop();
  }

  /** ← Bước trước: an earlier step again; the lesson's progress does not change. */
  protected back(): void {
    this.speech.stop();
    if (this.step() > 1) {
      this.step.update((s) => s - 1);
    }
    this.toTop();
  }

  /** Back to a lesson step left for later. */
  protected goToStep(key: StepKey): void {
    const index = this.steps().indexOf(key);
    if (index >= 0) {
      this.speech.stop();
      this.step.set(index + 1);
      this.toTop();
    }
  }

  /** ← → and Câu tiếp theo of the translation card. */
  protected goTranslation(index: number): void {
    const count = this.practice()?.translations.length ?? 0;
    if (index >= 0 && index < count) {
      this.speech.stop();
      this.translateIndex.set(index);
    }
  }


  /** Làm lại: back to step 1 with every answer cleared and the banks shuffled again. */
  protected retry(): void {
    this.speech.stop();
    this.finished.set(false);
    this.extra.set(null);
    this.practiced.set(new Set());
    this.step.set(1);
    this.translateIndex.set(0);
    this.fillScore.set(null);
    this.translateResults.set((this.practice()?.translations ?? []).map(() => null));
    this.seed.set(newSeed());
    this.toTop();
  }

  protected complete(step: Step): Promise<void> {
    if (this.relearn()) {
      // Nothing to record: the lesson is completed already. On to the next step.
      if (this.step() < this.steps().length) {
        this.step.update((s) => s + 1);
        this.toTop();
      }
      return Promise.resolve();
    }
    const done = this.record(this.studyApi.completeStep(this.id(), step));
    // Viết completes once the writing is submitted: the done card links to its grading.
    if (step !== 'write') {
      return done;
    }
    return done.then(() => {
      if (this.lessonDone()) {
        this.loadDoneWriting();
      }
    });
  }

  /**
   * Học lại bài: forgets this learner's comprehension answers and dictation results of the lesson
   * (the new ones count), then goes through every step again from the words.
   */
  protected async relearnLesson(): Promise<void> {
    this.relearnAsk.set(false);
    this.relearnError.set(false);
    try {
      await firstValueFrom(
        forkJoin([this.reading.resetAnswers(this.id()), this.listeningApi.reset(this.id())]),
      );
    } catch {
      this.relearnError.set(true);
      return;
    }
    this.lessonDone.set(false);
    this.relearn.set(true);
    this.tab.set('lesson');
    this.retry();
    this.overview.set(true);
  }

  /** Làm lại câu này in the translation: its result is forgotten until it is checked again. */
  protected onTranslationRedone(index: number): void {
    this.translateResults.update((r) => r.map((v, i) => (i === index ? null : v)));
  }

  /** Làm lại bước này in the translation: every result forgotten, back to the first sentence. */
  protected restartTranslations(): void {
    this.translateResults.set((this.practice()?.translations ?? []).map(() => null));
    this.translateIndex.set(0);
  }

  /** Bỏ qua in the Write step: the lesson is done without a writing. */
  protected skipWrite(): Promise<void> {
    return this.record(this.studyApi.skipWrite(this.id()));
  }

  private async record(request: Observable<LessonStudy>): Promise<void> {
    clearTimeout(this.positionTimer);
    this.saving.set(true);
    this.stepError.set(null);
    try {
      const study = await firstValueFrom(request);
      this.study.set(study);
      if (study.status === 'completed') {
        if (study.goalCompleted) {
          await this.router.navigateByUrl('/goal?completed=1');
          return;
        }
        this.lessonDone.set(true);
        this.loadDue();
      } else if (this.step() < this.steps().length) {
        this.step.update((s) => s + 1);
      }
      this.toTop();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.stepError.set(body?.message ?? 'Chưa lưu được bước này, vui lòng thử lại.');
    } finally {
      this.saving.set(false);
    }
  }

  /** How many cards are due now: the done card suggests reviewing them before the next lesson. */
  private loadDue(): void {
    this.vocabApi.due(1).subscribe({
      next: (list) => this.dueCount.set(list.total),
      error: () => undefined,
    });
  }

  /** The writing submitted in this lesson: the done card links to its grading. */
  private loadDoneWriting(): void {
    const lessonId = this.id();
    this.writingApi.lessonWriting(lessonId).subscribe({
      next: (v) => {
        if (this.id() === lessonId && v.writing?.status === 'submitted') {
          this.doneWriting.set(v.writing);
        }
      },
      error: () => undefined,
    });
  }

  /** Saves where the learner is, once they stop moving for a second. */
  protected savePosition(step: Step, sentenceIndex: number): void {
    if (this.relearn()) {
      return; // the lesson is completed: there is no position to keep
    }
    clearTimeout(this.positionTimer);
    const id = this.id();
    this.positionTimer = setTimeout(() => {
      this.studyApi.savePosition(id, step, sentenceIndex).subscribe({ error: () => undefined });
    }, POSITION_DELAY);
  }

  private toTop(): void {
    this.host.nativeElement.scrollIntoView?.({ block: 'start' });
  }
}
