import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
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
import { SpeechService } from '../../../core/services/speech.service';
import { StudyApiService } from '../../../core/services/study-api.service';
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
  summary,
} from './practice-logic';
import { PracticeSummary } from './practice-summary/practice-summary';
import { TranslateStep } from './translate-step/translate-step';
import { VocabStep } from './vocab-step/vocab-step';

type Tab = 'lesson' | 'reading';

/** The practice steps (shown when the lesson has their content), then the lesson's own steps. */
type PracticeKey = 'words' | 'dialogue' | 'fill' | 'translate';
type StepKey = PracticeKey | Step;

const STEP_NAMES: Record<StepKey, string> = {
  words: 'Từ vựng quan trọng',
  dialogue: 'Hội thoại mẫu',
  fill: 'Điền vào ô trống',
  translate: 'Dịch câu sang tiếng Anh',
  read: 'Đọc',
  listen: 'Nghe',
  write: 'Viết',
};

/** Delay before saving the reading or listening position (debounce 1 s). */
const POSITION_DELAY = 1000;

function isStudyStep(key: StepKey | null): key is Step {
  return key !== null && (STEPS as readonly string[]).includes(key);
}

/**
 * A lesson's page (F17, L): header, then the Bài học tab with the practice steps that have content
 * (words, dialogue, fill-in, translation) or the Bài đọc tab with the text and grammar note.
 *
 * The lesson being studied is learnt here (updated 2026-10-02, no "today" page): after the
 * practice come Đọc → Nghe → Viết (optional), recorded on the server; "← Bước trước" shows an
 * earlier step again. Once done, "Sang bài tiếp theo" opens the next lesson at once. Other lessons
 * keep the practice only, ending on its summary. Practice results stay on this page.
 */
@Component({
  selector: 'lu-lesson-detail',
  imports: [
    DialogueStep,
    FillStep,
    Icon,
    Listening,
    PracticeSummary,
    Reading,
    RouterLink,
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

  protected readonly id = signal('');
  protected readonly lesson = signal<ReadingLesson | null>(null);
  protected readonly vocabulary = signal<LessonVocabulary | null>(null);
  protected readonly practice = signal<PracticeView | null>(null);
  protected readonly study = signal<LessonStudy | null>(null);
  protected readonly loadError = signal<string | null>(null);
  /** Whether the page leads through Đọc → Nghe → Viết: set once, for the lesson being studied. */
  protected readonly studyFlow = signal(false);
  /** The lesson was completed on this page. */
  protected readonly lessonDone = signal(false);
  protected readonly saving = signal(false);
  protected readonly stepError = signal<string | null>(null);

  protected readonly tab = signal<Tab>('lesson');
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

  /** Steps with content, in order, then the lesson's steps when it is being studied. */
  protected readonly steps = computed<StepKey[]>(() => {
    const out: StepKey[] = [];
    if (this.words().length > 0) {
      out.push('words');
    }
    if (this.hasDialogue()) {
      out.push('dialogue');
    }
    if (this.hasFill()) {
      out.push('fill');
    }
    if (this.hasTranslations()) {
      out.push('translate');
    }
    return this.studyFlow() ? [...out, ...STEPS] : out;
  });
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
    return !!s && (this.lessonDone() || this.study()?.steps[s] === 'done');
  });
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
  protected readonly lastSentence = computed(
    () => this.translateIndex() >= (this.practice()?.translations.length ?? 0) - 1,
  );
  protected readonly nextLabel = computed(() => (this.isLast() ? 'Hoàn thành' : 'Tiếp theo'));
  /** Tiếp theo is there, except on a lesson step still to do: that step has its own buttons. */
  protected readonly canGoOn = computed(() => !this.studyStep() || this.studyStepDone());
  protected readonly canGoBack = computed(() => this.step() > 1 || this.translateIndex() > 0);
  /** The bar at the bottom, with Bước trước and Tiếp theo when they apply. */
  protected readonly showBar = computed(
    () =>
      this.shownTab() === 'lesson' &&
      !this.finished() &&
      !this.lessonDone() &&
      (this.canGoBack() || this.canGoOn()),
  );
  /** On the last step, and on its last sentence when it is the translation. */
  private readonly isLast = computed(
    () =>
      this.step() >= this.steps().length && (this.current() !== 'translate' || this.lastSentence()),
  );
  protected readonly summary = computed(() =>
    summary(this.practice()?.fill?.blanks.length ?? 0, this.fillScore(), this.translateResults()),
  );

  /** The text as paragraphs of sentences, for the Bài đọc tab. */
  protected readonly paragraphs = computed(() => {
    const l = this.lesson();
    if (!l) {
      return [];
    }
    const text = new Map(l.sentences.map((s) => [s.index, s.text]));
    const groups = l.paragraphs.length ? l.paragraphs : [l.sentences.map((s) => s.index)];
    return groups.map((p) => p.map((i) => text.get(i) ?? '').join(' '));
  });

  private positionTimer?: ReturnType<typeof setTimeout>;
  private loading?: Subscription;

  constructor() {
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
    this.lessonDone.set(false);
    this.stepError.set(null);
    this.tab.set('lesson');
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
        this.lesson.set(r.lesson);
      },
      error: (err: unknown) => this.loadError.set(loadErrorMessage(err)),
    });
  }

  /** Back to the lesson's step in progress, when one was done or a position saved. */
  private resumeStep(study: LessonStudy | null): void {
    if (study?.status !== 'studying' || !isStudyStep(study.currentStep as StepKey)) {
      return;
    }
    const started = STEPS.some((s) => study.steps[s] === 'done') || study.sentenceIndex > 0;
    if (started) {
      this.step.set(this.steps().indexOf(study.currentStep as Step) + 1);
    }
  }

  /** Reads a word or sentence of the practice with the browser's voice. */
  protected play(text: string): void {
    this.speech.speak(text, this.rate());
  }

  /** Reading goes on across tabs; the Bài học tab stays as it was. */
  protected selectTab(tab: Tab): void {
    this.tab.set(tab);
  }

  protected onTabKeydown(event: KeyboardEvent): void {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') {
      return;
    }
    event.preventDefault();
    const tab: Tab = this.tab() === 'lesson' ? 'reading' : 'lesson';
    this.selectTab(tab);
    const list = (event.currentTarget as HTMLElement).closest('[role="tablist"]');
    list?.querySelector<HTMLElement>(`#tab-${tab}`)?.focus();
  }

  protected onFillChecked(score: Score): void {
    this.fillScore.set(score);
  }

  protected onTranslationChecked(index: number, ok: boolean): void {
    this.translateResults.update((r) => r.map((v, i) => (i === index ? ok : v)));
  }

  protected next(): void {
    this.speech.stop();
    if (this.current() === 'translate' && !this.lastSentence()) {
      this.translateIndex.update((i) => i + 1);
    } else if (this.step() < this.steps().length) {
      this.step.update((s) => s + 1);
    } else {
      this.finished.set(true);
    }
    this.toTop();
  }

  /** ← Bước trước: an earlier step again; the lesson's progress does not change. */
  protected back(): void {
    this.speech.stop();
    if (this.current() === 'translate' && this.translateIndex() > 0) {
      this.translateIndex.update((i) => i - 1);
    } else if (this.step() > 1) {
      this.step.update((s) => s - 1);
    }
    this.toTop();
  }

  /** Làm lại: back to step 1 with every answer cleared and the banks shuffled again. */
  protected retry(): void {
    this.speech.stop();
    this.finished.set(false);
    this.step.set(1);
    this.translateIndex.set(0);
    this.fillScore.set(null);
    this.translateResults.set((this.practice()?.translations ?? []).map(() => null));
    this.seed.set(newSeed());
    this.toTop();
  }

  protected complete(step: Step): Promise<void> {
    return this.record(this.studyApi.completeStep(this.id(), step));
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

  /** Saves where the learner is, once they stop moving for a second. */
  protected savePosition(step: Step, sentenceIndex: number): void {
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
