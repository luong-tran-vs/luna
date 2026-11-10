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
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgOptimizedImage } from '@angular/common';

import { VerbProgress } from '../../../core/models/verbs';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { VerbCard } from '../verb-card/verb-card';
import { VERB_LESSONS } from '../verb-lessons.data';
import { VerbPattern } from '../verb-pattern/verb-pattern';
import { VerbPictures } from '../verb-pictures.service';
import { VerbSentenceRound } from '../verb-sentence-round/verb-sentence-round';
import { isLearned, lessonVerbs } from '../verb-study.logic';
import { VerbSummary } from '../verb-summary/verb-summary';
import { TypedResult, VerbTypeRound } from '../verb-type-round/verb-type-round';

/** Làm quen is three screens since 2026-11-10: the rule (pattern), the cards (learn), the table (summary). */
type Stage = 'pattern' | 'learn' | 'summary' | 'choose' | 'type' | 'confirm' | 'done';

const STEPS: readonly { key: Stage; label: string; stages: readonly Stage[] }[] = [
  { key: 'learn', label: 'Làm quen', stages: ['pattern', 'learn', 'summary'] },
  { key: 'choose', label: 'Chọn', stages: ['choose'] },
  { key: 'type', label: 'Gõ', stages: ['type'] },
];

/**
 * One lesson of verbs (design/de-xuat-dong-tu.png, reworked 2026-11-10). Làm quen: the rule of the
 * lesson on all its verbs and a guess of one (VerbPattern), a card per verb with its picture, beats
 * and uses (VerbCard), then the table to test oneself (VerbSummary). Chọn: the forms in sentences
 * (VerbSentenceRound). Gõ: typing V2 and V3. Then the learner confirms which verbs are learned; only
 * those join the reviews.
 */
@Component({
  selector: 'lu-verb-lesson',
  imports: [
    NgOptimizedImage,
    RouterLink,
    Icon,
    VerbTypeRound,
    VerbPattern,
    VerbCard,
    VerbSummary,
    VerbSentenceRound,
  ],
  templateUrl: './verb-lesson.html',
  styleUrl: './verb-lesson.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbLessonPage {
  private readonly api = inject(VerbsApiService);
  private readonly injector = inject(Injector);
  private readonly pictureList = inject(VerbPictures);

  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly steps = STEPS;
  protected readonly lessonCount = VERB_LESSONS.length;
  /** Picture URL of each verb that has one. */
  protected readonly pictures = this.pictureList.urls;

  protected readonly number = signal(0);
  protected readonly lesson = computed(
    () => VERB_LESSONS.find((l) => l.number === this.number()) ?? null,
  );
  protected readonly verbs = computed(() => {
    const l = this.lesson();
    return l ? lessonVerbs(l, IRREGULAR_VERBS) : [];
  });
  protected readonly nextNumber = computed(() =>
    this.number() < VERB_LESSONS.length ? this.number() + 1 : null,
  );

  protected readonly progress = signal<ReadonlyMap<string, VerbProgress>>(new Map());
  protected readonly stage = signal<Stage>('pattern');
  protected readonly stepIndex = computed(() =>
    STEPS.findIndex((s) => s.stages.includes(this.stage())),
  );
  /** The step of a later stage is done. */
  protected readonly confirming = computed(
    () => this.stage() === 'confirm' || this.stage() === 'done',
  );

  /** The card on screen in Làm quen. */
  protected readonly card = signal(0);
  protected readonly typed = signal<TypedResult[]>([]);
  /** Bases ticked on the confirm screen. */
  protected readonly ticked = signal<ReadonlySet<string>>(new Set());
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);

  protected readonly learnedTotal = computed(
    () => [...this.progress().values()].filter((p) => isLearned(p)).length,
  );

  constructor() {
    inject(ActivatedRoute)
      .paramMap.pipe(takeUntilDestroyed())
      .subscribe((p) => {
        this.number.set(Number(p.get('number')));
        this.restart();
      });
    this.api.progress().subscribe({
      next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
      // Without it nothing is ticked as learned already; confirming still works.
      error: () => this.progress.set(new Map()),
    });
    this.pictureList.load();
  }

  protected restart(): void {
    this.stage.set('pattern');
    this.card.set(0);
    this.typed.set([]);
    this.saveError.set(false);
  }

  protected isLearned(base: string): boolean {
    return isLearned(this.progress().get(base));
  }

  // --- Làm quen ---

  protected toCards(): void {
    this.card.set(0);
    this.go('learn');
  }

  protected nextCard(): void {
    if (this.card() + 1 < this.verbs().length) {
      this.card.update((c) => c + 1);
      this.focusHeading();
      return;
    }
    this.go('summary');
  }

  protected prevCard(): void {
    this.card.update((c) => Math.max(0, c - 1));
    this.focusHeading();
  }

  protected toChoose(): void {
    this.go('choose');
  }

  protected toType(): void {
    this.go('type');
  }

  // --- Gõ, then confirm ---

  protected onTyped(results: TypedResult[]): void {
    this.typed.set(results);
    // Ticked: the verbs typed right, and those already learned.
    this.ticked.set(
      new Set(
        this.verbs()
          .map((v) => v.base)
          .filter((b) => this.isLearned(b) || this.typedRight(b)),
      ),
    );
    this.go('confirm');
  }

  protected typedRight(base: string): boolean {
    return this.typed().find((r) => r.verb.base === base)?.right ?? false;
  }

  protected typedText(base: string): string {
    const r = this.typed().find((t) => t.verb.base === base);
    return r ? `${r.past || '(trống)'} · ${r.participle || '(trống)'}` : '';
  }

  protected toggle(base: string): void {
    this.ticked.update((s) => {
      const out = new Set(s);
      if (out.has(base)) {
        out.delete(base);
      } else {
        out.add(base);
      }
      return out;
    });
  }

  /** Sends the ticks: ticked verbs become learned, unticked ones learned before are no longer. */
  protected confirm(): void {
    const bases = this.verbs().map((v) => v.base);
    const learn = bases.filter((b) => this.ticked().has(b) && !this.isLearned(b));
    const unlearn = bases.filter((b) => !this.ticked().has(b) && this.isLearned(b));
    if (learn.length === 0 && unlearn.length === 0) {
      this.go('done');
      return;
    }
    this.saving.set(true);
    this.saveError.set(false);
    const calls = [
      ...(learn.length ? [{ bases: learn, learned: true }] : []),
      ...(unlearn.length ? [{ bases: unlearn, learned: false }] : []),
    ];
    let left = calls.length;
    for (const c of calls) {
      this.api.setLearned(c.bases, c.learned).subscribe({
        next: (r) => {
          this.progress.update((m) => new Map([...m, ...r.verbs.map((v) => [v.base, v] as const)]));
          if (--left === 0) {
            this.saving.set(false);
            this.go('done');
          }
        },
        error: () => {
          this.saving.set(false);
          this.saveError.set(true);
        },
      });
    }
  }

  private go(stage: Stage): void {
    this.stage.set(stage);
    this.focusHeading();
  }

  private focusHeading(): void {
    afterNextRender(
      () => {
        document.scrollingElement?.scrollTo?.({ top: 0 });
        this.heading()?.nativeElement.focus({ preventScroll: true });
      },
      { injector: this.injector },
    );
  }
}
