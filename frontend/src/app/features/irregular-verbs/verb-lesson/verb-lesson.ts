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
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { SpeechService } from '../../../core/services/speech.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from '../irregular-verbs.data';
import { spellings, spokenForms } from '../irregular-verbs.logic';
import { VERB_EXAMPLES, VERB_IPA, VERB_LESSONS, VERB_SAY_TIPS } from '../verb-lessons.data';
import { changedPart, choiceQuestion, exampleParts, isLearned, lessonVerbs } from '../verb-study.logic';
import { TypedResult, VerbTypeRound } from '../verb-type-round/verb-type-round';

/** Reading speed of the forms and examples. */
const RATE = 0.85;

type Stage = 'learn' | 'choose' | 'type' | 'confirm' | 'done';

const STEPS: readonly { key: Stage; label: string }[] = [
  { key: 'learn', label: 'Làm quen' },
  { key: 'choose', label: 'Chọn' },
  { key: 'type', label: 'Gõ' },
];

/**
 * One lesson of verbs (design/tat-ca-de-xuat.png (mục dong-tu-hoc-theo-bai)): a card per verb to get to know it, a
 * multiple-choice question per verb, typing V2 and V3, then the learner confirms which verbs are
 * learned (design/tat-ca-de-xuat.png (mục dong-tu-da-hoc-on-lai)); only those join the reviews.
 */
@Component({
  selector: 'lu-verb-lesson',
  imports: [NgOptimizedImage, RouterLink, Icon, VerbTypeRound],
  templateUrl: './verb-lesson.html',
  styleUrl: './verb-lesson.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbLessonPage {
  private readonly api = inject(VerbsApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);

  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly canSpeak = this.speech.supported;
  protected readonly steps = STEPS;
  protected readonly lessonCount = VERB_LESSONS.length;

  protected readonly number = signal(0);
  protected readonly lesson = computed(() => VERB_LESSONS.find((l) => l.number === this.number()) ?? null);
  protected readonly verbs = computed(() => {
    const l = this.lesson();
    return l ? lessonVerbs(l, IRREGULAR_VERBS) : [];
  });
  protected readonly nextNumber = computed(() => (this.number() < VERB_LESSONS.length ? this.number() + 1 : null));

  protected readonly progress = signal<ReadonlyMap<string, VerbProgress>>(new Map());
  protected readonly stage = signal<Stage>('learn');
  protected readonly stepIndex = computed(() => STEPS.findIndex((s) => s.key === this.stage()));

  /** The card on screen in Làm quen. */
  protected readonly card = signal(0);
  protected readonly questions = signal<ReturnType<typeof choiceQuestion>[]>([]);
  protected readonly question = signal(0);
  protected readonly chosen = signal<string | null>(null);
  protected readonly typed = signal<TypedResult[]>([]);
  /** Bases ticked on the confirm screen. */
  protected readonly ticked = signal<ReadonlySet<string>>(new Set());
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);

  protected readonly learnedTotal = computed(() => [...this.progress().values()].filter((p) => isLearned(p)).length);

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
  }

  protected restart(): void {
    this.stage.set('learn');
    this.card.set(0);
    this.question.set(0);
    this.chosen.set(null);
    this.typed.set([]);
    this.saveError.set(false);
  }

  protected example(v: IrregularVerb): { en: string; vi: string } | null {
    return VERB_EXAMPLES[v.base] ?? null;
  }

  /** The form with the part that differs from the base marked: [same, changed]. */
  protected parts(v: IrregularVerb, form: string): [string, string] {
    return changedPart(v.base, spellings(form)[0]);
  }

  /** How form i (0: V1, 1: V2, 2: V3) is said, "/bɔːt/"; '' when not written. */
  protected ipa(v: IrregularVerb, i: number): string {
    return VERB_IPA[v.base]?.[i].map((x) => `/${x}/`).join(', ') ?? '';
  }

  /** A tip for forms often said wrong, else the verb's note. */
  protected tip(v: IrregularVerb): string {
    return VERB_SAY_TIPS[v.base] ?? v.noteVi ?? '';
  }

  protected exampleParts(sentence: string, v: IrregularVerb): { text: string; verb: boolean }[] {
    return exampleParts(sentence, v);
  }

  protected isLearned(base: string): boolean {
    return isLearned(this.progress().get(base));
  }

  protected listen(text: string): void {
    this.speech.speak(text, RATE);
  }

  protected listenForms(v: IrregularVerb): void {
    this.speech.speak(spokenForms(v), RATE);
  }

  // --- Làm quen ---

  protected nextCard(): void {
    if (this.card() + 1 < this.verbs().length) {
      this.card.update((c) => c + 1);
      this.focusHeading();
      return;
    }
    const verbs = this.verbs();
    const seed = newSeed();
    this.questions.set(verbs.map((v, i) => choiceQuestion(v, i % 2 === 0 ? 'past' : 'participle', verbs, seed + i)));
    this.go('choose');
  }

  protected prevCard(): void {
    this.card.update((c) => Math.max(0, c - 1));
    this.focusHeading();
  }

  // --- Chọn ---

  protected choose(option: string): void {
    if (this.chosen() === null) {
      this.chosen.set(option);
    }
  }

  protected nextQuestion(): void {
    if (this.question() + 1 < this.questions().length) {
      this.question.update((q) => q + 1);
      this.chosen.set(null);
      this.focusHeading();
      return;
    }
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
