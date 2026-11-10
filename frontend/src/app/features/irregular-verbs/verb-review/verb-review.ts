import { NgOptimizedImage, NgTemplateOutlet } from '@angular/common';
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
import { ActivatedRoute, RouterLink } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { SpeechService } from '../../../core/services/speech.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IrregularVerb } from '../irregular-verbs.data';
import { spokenForms } from '../irregular-verbs.logic';
import { canFlip, dueLabel, learnedVerbs, reviewRound } from '../verb-study.logic';
import { QuickResult, VerbFlipRound } from '../verb-flip-round/verb-flip-round';
import { VerbPictures } from '../verb-pictures.service';
import { TypedResult, VerbTypeRound } from '../verb-type-round/verb-type-round';

/**
 * Reviewing the learned verbs (design/tat-ca-de-xuat.png (mục dong-tu-da-hoc-on-lai)): every learned verb, due ones
 * first, a round where the learner types V2 and V3 of the due ones, and the result with when each
 * verb comes back. A wrong answer keeps the verb learned; it is only asked again sooner. A verb
 * can be taken out of the reviews here ("Bỏ đánh dấu"). ?start=1 starts the round at once.
 */
@Component({
  selector: 'lu-verb-review',
  imports: [
    NgOptimizedImage,
    NgTemplateOutlet,
    RouterLink,
    Icon,
    Loading,
    VerbTypeRound,
    VerbFlipRound,
  ],
  templateUrl: './verb-review.html',
  styleUrl: './verb-review.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbReview {
  private readonly api = inject(VerbsApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);
  private readonly route = inject(ActivatedRoute);

  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly canSpeak = this.speech.supported;
  protected readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);
  protected readonly loadError = signal(false);
  /** flip: "Ôn nhanh" by flipping cards (added 2026-11-10); quiz: typing V2 and V3 ("Ôn kỹ"). */
  protected readonly stage = signal<'list' | 'flip' | 'quiz' | 'done'>('list');
  /** The verbs of the round flipped, and those typed (all of them in "Ôn kỹ"). */
  protected readonly flipRound = signal<IrregularVerb[]>([]);
  protected readonly quick = signal<QuickResult[]>([]);
  protected readonly pictures = inject(VerbPictures).urls;
  protected readonly round = signal<IrregularVerb[]>([]);
  protected readonly results = signal<TypedResult[]>([]);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  /** The base being taken out of the reviews, to disable its button. */
  protected readonly removing = signal<string | null>(null);

  private readonly now = signal(new Date());
  protected readonly learned = computed(() =>
    learnedVerbs(this.progress() ?? new Map(), this.now()),
  );
  protected readonly due = computed(() => this.learned().filter((l) => l.due));
  protected readonly upcoming = computed(() => this.learned().filter((l) => !l.due));
  protected readonly rightCount = computed(
    () =>
      this.results().filter((r) => r.right).length +
      this.quick().filter((q) => q.grade !== 'forgot').length,
  );
  protected readonly doneCount = computed(() => this.results().length + this.quick().length);

  constructor() {
    this.load();
    inject(VerbPictures).load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api.progress().subscribe({
      next: (r) => {
        this.progress.set(new Map(r.verbs.map((v) => [v.base, v])));
        if (this.route.snapshot?.queryParamMap.get('start') === '1' && this.learned().length > 0) {
          this.start();
        }
      },
      error: () => this.loadError.set(true),
    });
  }

  protected label(p: VerbProgress): string {
    return dueLabel(p.dueAt, this.now());
  }

  protected readonly emptyProgress: ReadonlyMap<string, VerbProgress> = new Map();

  protected gradeLabel(g: QuickResult['grade']): string {
    return g === 'forgot' ? 'Quên · vẫn là đã học' : g === 'hard' ? 'Hơi nhớ' : 'Nhớ';
  }

  protected forms(v: IrregularVerb): string {
    return `${v.base} · ${v.past} · ${v.participle}`;
  }

  /**
   * Starts a round. "Ôn nhanh" (quick) flips the cards of the verbs typed right before and has the
   * others typed after; "Ôn kỹ" types them all.
   */
  protected start(quick = false): void {
    this.now.set(new Date());
    const progress = this.progress() ?? new Map();
    const round = reviewRound(progress, this.now(), newSeed());
    const flip = quick ? round.filter((v) => canFlip(progress.get(v.base))) : [];
    this.flipRound.set(flip);
    this.round.set(round.filter((v) => !flip.includes(v)));
    this.results.set([]);
    this.quick.set([]);
    this.stage.set(flip.length > 0 ? 'flip' : 'quiz');
  }

  protected onFlipped(results: QuickResult[]): void {
    this.quick.set(results);
    if (this.round().length > 0) {
      this.stage.set('quiz');
      return;
    }
    this.finish();
  }

  protected onFinished(results: TypedResult[]): void {
    this.results.set(results);
    this.finish();
  }

  private finish(): void {
    this.stage.set('done');
    this.send();
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }

  protected send(): void {
    this.saving.set(true);
    this.saveError.set(false);
    const typed = this.results().map((r) => ({ base: r.verb.base, right: r.right }));
    const quick = this.quick().map((q) => ({
      base: q.verb.base,
      right: q.grade !== 'forgot',
      quick: true,
      hard: q.grade === 'hard',
    }));
    this.api.record([...typed, ...quick]).subscribe({
      next: (r) => {
        this.merge(r.verbs);
        this.now.set(new Date());
        this.saving.set(false);
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set(true);
      },
    });
  }

  /** When the verb of a result comes back, once the round is saved. */
  protected nextLabel(base: string): string {
    const p = this.progress()?.get(base);
    return p && !this.saving() && !this.saveError()
      ? `hỏi lại ${dueLabel(p.dueAt, this.now())}`
      : '';
  }

  protected unlearn(base: string): void {
    this.removing.set(base);
    this.api.setLearned([base], false).subscribe({
      next: (r) => {
        this.merge(r.verbs);
        this.removing.set(null);
      },
      error: () => this.removing.set(null),
    });
  }

  protected listenForms(v: IrregularVerb): void {
    this.speech.speak(spokenForms(v), 0.85);
  }

  protected backToList(): void {
    this.now.set(new Date());
    this.stage.set('list');
  }

  private merge(verbs: VerbProgress[]): void {
    this.progress.update((m) => new Map([...(m ?? []), ...verbs.map((v) => [v.base, v] as const)]));
  }
}
