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
import { RouterLink } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { SpeechService } from '../../../core/services/speech.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from '../irregular-verbs.data';
import { PATTERNS, patternOf, spokenForms } from '../irregular-verbs.logic';
import { gradeForm, inSet, pickRound, ROUND_SIZE, summarize, VerbSet } from './verb-practice.logic';

/** Reading speed of the forms, as in the list. */
const RATE = 0.85;

/** What was typed for a verb answered wrong the first time, for "Cần ôn thêm". */
interface Miss {
  verb: IrregularVerb;
  past: string;
  participle: string;
}

/**
 * The irregular verbs practice: choose a set, then a round of ROUND_SIZE verbs where the learner
 * types V2 and V3 of each. A verb got wrong is asked again at the end of the round; its first
 * answer is what the server records (POST /api/verbs/results), moving it up a box or back to 0.
 */
@Component({
  selector: 'lu-verb-practice',
  imports: [NgOptimizedImage, RouterLink, Icon, Loading],
  templateUrl: './verb-practice.html',
  styleUrl: './verb-practice.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbPractice {
  private readonly api = inject(VerbsApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);

  private readonly pastInput = viewChild<ElementRef<HTMLInputElement>>('pastInput');
  private readonly nextButton = viewChild<ElementRef<HTMLButtonElement>>('nextButton');
  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly canSpeak = this.speech.supported;
  protected readonly patterns = PATTERNS;

  protected readonly stage = signal<'setup' | 'quiz' | 'done'>('setup');
  protected readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);
  protected readonly loadError = signal(false);
  protected readonly set = signal<VerbSet>('common');

  protected readonly queue = signal<IrregularVerb[]>([]);
  protected readonly index = signal(0);
  /** Verbs of the round, in the order first asked. */
  private readonly roundSize = signal(0);
  protected readonly past = signal('');
  protected readonly participle = signal('');
  /** null until the answer on screen is checked. */
  protected readonly checked = signal<{ past: boolean; participle: boolean } | null>(null);
  /** The first answer of each verb of the round. */
  private readonly first = signal<ReadonlyMap<string, boolean>>(new Map());
  protected readonly misses = signal<Miss[]>([]);

  protected readonly saving = signal(false);
  protected readonly saveError = signal(false);
  /** Known verbs of the set before the round, to tell how many it added. */
  private readonly knownBefore = signal(0);
  protected readonly saved = signal(false);

  /** The choices of the setup screen, each with how much of it is known. */
  protected readonly options = computed(() => {
    const progress = this.progress() ?? new Map<string, VerbProgress>();
    const now = new Date();
    const sets: { key: VerbSet; title: string; hint: string }[] = [
      { key: 'common', title: 'Thông dụng', hint: 'Những động từ gặp nhiều nhất' },
      ...PATTERNS.map((p) => ({ key: p.key, title: p.label, hint: `${p.example} · ${p.hintVi}` })),
      { key: 'all', title: 'Tất cả', hint: `${IRREGULAR_VERBS.length} động từ` },
    ];
    return sets.map((s) => ({ ...s, ...summarize(inSet(IRREGULAR_VERBS, s.key), progress, now) }));
  });
  protected readonly chosen = computed(() => this.options().find((o) => o.key === this.set())!);
  protected readonly startSize = computed(() => {
    const o = this.chosen();
    const left = o.due + (o.total - o.known);
    return Math.min(ROUND_SIZE, left > 0 ? left : o.total);
  });

  protected readonly current = computed(() => this.queue()[this.index()] ?? null);
  protected readonly isLast = computed(() => this.index() + 1 >= this.queue().length);
  protected readonly canCheck = computed(() => this.past().trim() !== '' && this.participle().trim() !== '');
  protected readonly right = computed(() => {
    const c = this.checked();
    return !!c && c.past && c.participle;
  });
  /** The verb on screen is asked a second time (it was wrong the first time). */
  protected readonly isRetry = computed(() => this.index() >= this.roundSize());
  protected readonly firstRight = computed(() => [...this.first().values()].filter(Boolean).length);
  protected readonly total = computed(() => this.first().size);
  protected readonly knownNow = computed(() => this.chosen().known);
  protected readonly gained = computed(() => Math.max(0, this.knownNow() - this.knownBefore()));

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api.progress().subscribe({
      next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
      error: () => this.loadError.set(true),
    });
  }

  protected patternLabel(v: IrregularVerb): string {
    const key = patternOf(v);
    return PATTERNS.find((p) => p.key === key)?.label ?? '';
  }

  protected patternHint(v: IrregularVerb): string {
    const key = patternOf(v);
    return PATTERNS.find((p) => p.key === key)?.hintVi ?? '';
  }

  protected forms(v: IrregularVerb): string {
    return `${v.base} – ${v.past} – ${v.participle}`;
  }

  protected start(): void {
    const verbs = inSet(IRREGULAR_VERBS, this.set());
    const round = pickRound(verbs, this.progress() ?? new Map(), new Date(), newSeed());
    this.knownBefore.set(this.chosen().known);
    this.queue.set(round);
    this.roundSize.set(round.length);
    this.index.set(0);
    this.first.set(new Map());
    this.misses.set([]);
    this.saved.set(false);
    this.saveError.set(false);
    this.stage.set('quiz');
    this.resetAnswer();
  }

  protected onPast(event: Event): void {
    this.past.set((event.target as HTMLInputElement).value);
  }

  protected onParticiple(event: Event): void {
    this.participle.set((event.target as HTMLInputElement).value);
  }

  /** Enter in V2 goes to V3; Enter in V3 checks. */
  protected onPastEnter(next: HTMLInputElement): void {
    next.focus();
  }

  protected check(): void {
    const v = this.current();
    if (!v || this.checked() || !this.canCheck()) {
      return;
    }
    const result = {
      past: gradeForm(this.past(), v.past),
      participle: gradeForm(this.participle(), v.participle),
    };
    this.checked.set(result);
    const right = result.past && result.participle;
    if (!this.first().has(v.base)) {
      this.first.update((m) => new Map(m).set(v.base, right));
      if (!right) {
        this.misses.update((m) => [...m, { verb: v, past: this.past().trim(), participle: this.participle().trim() }]);
        // Asked again at the end of the round, once.
        this.queue.update((q) => [...q, v]);
      }
    }
    afterNextRender(() => this.nextButton()?.nativeElement.focus(), { injector: this.injector });
  }

  protected next(): void {
    if (!this.checked()) {
      return;
    }
    if (this.isLast()) {
      this.stage.set('done');
      this.send();
      afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
      return;
    }
    this.index.update((i) => i + 1);
    this.resetAnswer();
  }

  protected send(): void {
    const results = [...this.first()].map(([base, right]) => ({ base, right }));
    if (results.length === 0) {
      return;
    }
    this.saving.set(true);
    this.saveError.set(false);
    this.api.record(results).subscribe({
      next: (r) => {
        this.progress.update((m) => {
          const out = new Map(m ?? []);
          for (const v of r.verbs) {
            out.set(v.base, v);
          }
          return out;
        });
        this.saving.set(false);
        this.saved.set(true);
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set(true);
      },
    });
  }

  protected listen(text: string): void {
    this.speech.speak(text, RATE);
  }

  protected listenForms(v: IrregularVerb): void {
    this.speech.speak(spokenForms(v), RATE);
  }

  protected backToSetup(): void {
    this.stage.set('setup');
  }

  private resetAnswer(): void {
    this.past.set('');
    this.participle.set('');
    this.checked.set(null);
    afterNextRender(
      () => {
        this.pastInput()?.nativeElement.focus();
      },
      { injector: this.injector },
    );
  }
}
