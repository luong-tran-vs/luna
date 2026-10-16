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
import { patternParts, structureTone } from '../grammar-logic';
import { GrammarQuiz } from '../grammar-quiz/grammar-quiz';
import { savedStep, saveStep, stepIndex, STEPS } from '../grammar-steps';

/** One way back into a point already studied: a button of the "Học tiếp từ chỗ dở" card. */
interface ResumeAction {
  label: string;
  step: number;
  /** Starts the round over the practice exercises answered wrong last time. */
  weak?: boolean;
}

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
  imports: [NgOptimizedImage, RouterLink, Icon, Loading, GrammarQuiz, SpeakButton],
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
  private readonly practiceQuiz = viewChild<GrammarQuiz>('practiceQuiz');

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
  /** The bar's button on Lý thuyết and Cấu trúc (the quizzes have their own). */
  protected readonly barLabel = computed(() => {
    const next = STEPS[this.step() + 1];
    return this.step() < 2 && next ? `Sang ${next.label}` : '';
  });
  protected readonly testContinue = computed(() => {
    const n = this.nextPoint();
    return n ? `Học tiếp: ${n.titleVi}` : 'Về danh sách ngữ pháp';
  });
  /** The structures with their pattern cut into coloured pieces and the tone of their label. */
  protected readonly structures = computed(() => {
    const list = this.detail()?.content.structures ?? [];
    const parts = patternParts(list.map((s) => s.pattern));
    return list.map((s, i) => ({ ...s, parts: parts[i], tone: structureTone(s.label) }));
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

  protected next(): void {
    this.goTo(Math.min(this.step() + 1, STEPS.length - 1));
  }

  protected back(): void {
    this.goTo(Math.max(this.step() - 1, 0));
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
