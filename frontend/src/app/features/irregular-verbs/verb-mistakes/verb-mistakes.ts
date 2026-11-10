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
import { RouterLink } from '@angular/router';

import { VerbMistake, VerbProgress } from '../../../core/models/verbs';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { IrregularVerb } from '../irregular-verbs.data';
import {
  CLEAN_RUN,
  typoDiff,
  PRACTICE_SIZE,
  rightForm,
  strugglingVerbs,
  typedResult,
} from '../verb-mistakes.logic';
import { pairOf } from '../verb-pairs.data';
import { VerbPictures } from '../verb-pictures.service';
import { TypedResult, VerbTypeRound } from '../verb-type-round/verb-type-round';

/**
 * "Từ hay sai" (design/de-xuat-dong-tu.png part 6, added 2026-11-10): the verbs the learner keeps
 * typing wrong, each with its last mistake shown letter by letter and, for a verb often mixed up
 * with another, the two side by side. "Luyện" types a few of them again; typed right CLEAN_RUN
 * times in a row, a verb leaves the list. Practising does not move the reviews.
 */
@Component({
  selector: 'lu-verb-mistakes',
  imports: [RouterLink, Icon, Loading, VerbTypeRound],
  templateUrl: './verb-mistakes.html',
  styleUrl: './verb-mistakes.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbMistakes {
  private readonly api = inject(VerbsApiService);
  private readonly injector = inject(Injector);
  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly pictures = inject(VerbPictures).urls;
  protected readonly cleanRun = CLEAN_RUN;
  protected readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);
  protected readonly loadError = signal(false);
  protected readonly stage = signal<'list' | 'practice' | 'done'>('list');
  protected readonly round = signal<IrregularVerb[]>([]);
  protected readonly results = signal<TypedResult[]>([]);
  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);

  private readonly now = signal(new Date());
  protected readonly verbs = computed(() =>
    strugglingVerbs(this.progress() ?? new Map(), this.now()),
  );
  protected readonly practiceCount = computed(() => Math.min(PRACTICE_SIZE, this.verbs().length));
  protected readonly rightCount = computed(() => this.results().filter((r) => r.right).length);

  constructor() {
    this.load();
    inject(VerbPictures).load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api.progress().subscribe({
      next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
      error: () => this.loadError.set(true),
    });
  }

  protected forms(v: IrregularVerb): string {
    return `${v.base} · ${v.past} · ${v.participle}`;
  }

  protected formLabel(m: VerbMistake): string {
    return m.form === 'past' ? 'V2' : 'V3';
  }

  protected right(v: IrregularVerb, m: VerbMistake): string {
    return rightForm(v, m);
  }

  protected diff(v: IrregularVerb, m: VerbMistake) {
    return typoDiff(m.typed, rightForm(v, m));
  }

  protected pair(base: string) {
    return pairOf(base);
  }

  protected start(): void {
    this.round.set(
      this.verbs()
        .slice(0, PRACTICE_SIZE)
        .map((s) => s.verb),
    );
    this.results.set([]);
    this.stage.set('practice');
  }

  protected onFinished(results: TypedResult[]): void {
    this.results.set(results);
    this.stage.set('done');
    this.send();
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }

  protected send(): void {
    this.saving.set(true);
    this.saveError.set(false);
    this.api.record(this.results().map((r) => typedResult(r, true))).subscribe({
      next: (r) => {
        this.progress.update(
          (m) => new Map([...(m ?? []), ...r.verbs.map((v) => [v.base, v] as const)]),
        );
        this.now.set(new Date());
        this.saving.set(false);
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set(true);
      },
    });
  }

  /** Whether a verb of the round is still "hay sai" once saved. */
  protected stillStruggling(base: string): boolean {
    return this.verbs().some((s) => s.verb.base === base);
  }

  /** Why a verb stays: "gõ đúng thêm 1 lần nữa" while a clean run is going, else "vẫn hay sai". */
  protected stayNote(base: string): string {
    const run = this.progress()?.get(base)?.cleanRun ?? 0;
    return run > 0 ? `gõ đúng thêm ${CLEAN_RUN - run} lần nữa` : 'vẫn hay sai';
  }

  protected backToList(): void {
    this.now.set(new Date());
    this.stage.set('list');
  }
}
