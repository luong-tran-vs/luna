import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { VerbProgress } from '../../core/models/verbs';
import { VerbsApiService } from '../../core/services/verbs-api.service';
import { Icon } from '../../shared/components/icon/icon';
import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { byLetter, filterVerbs, patternOf, VerbPattern } from './irregular-verbs.logic';
import { VerbFilters } from './verb-filters/verb-filters';
import { VERB_LESSONS } from './verb-lessons.data';
import { VerbSheet } from './verb-sheet/verb-sheet';
import { isLearned, learnedIn, modelForms, nextLesson, reviewSummary } from './verb-study.logic';
import { VerbTable } from './verb-table/verb-table';

/** How long the "Đã đánh dấu" note stays, in ms. */
const NOTE_MS = 6000;

const VIEW_KEY = 'luna.verbs.view';

type View = 'learn' | 'lookup';

function savedView(): View {
  try {
    return sessionStorage.getItem(VIEW_KEY) === 'lookup' ? 'lookup' : 'learn';
  } catch {
    return 'learn';
  }
}

/**
 * The "Động từ" tab (design/tat-ca-de-xuat.png (mục dong-tu-gon-lai-v2)), in two parts. Học: one card with how many
 * common verbs are learned, the next lesson (its pattern on one verb) and the way to review, then
 * the path of lessons. Tra cứu: the irregular verbs in three aligned columns V1 | V2 | V3, with
 * search (English forms or Vietnamese meaning, marks optional) and filters; tapping a row opens its
 * details, where it is read aloud and marked as learned.
 */
@Component({
  selector: 'lu-irregular-verbs',
  imports: [RouterLink, Icon, VerbFilters, VerbTable, VerbSheet],
  templateUrl: './irregular-verbs.html',
  styleUrl: './irregular-verbs.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class IrregularVerbs {
  private readonly api = inject(VerbsApiService);

  protected readonly total = IRREGULAR_VERBS.length;
  protected readonly commonTotal = IRREGULAR_VERBS.filter((v) => v.common).length;
  protected readonly lessonCount = VERB_LESSONS.length;

  protected readonly view = signal<View>(savedView());
  protected readonly query = signal('');
  protected readonly commonOnly = signal(false);
  protected readonly learnedOnly = signal(false);
  protected readonly pattern = signal<VerbPattern | null>(null);
  /** The learner's records; null until loaded (or when they cannot be: the lists work without). */
  protected readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);
  /** The verb whose details are open. */
  protected readonly selected = signal<IrregularVerb | null>(null);
  /** The last verb marked or unmarked, to say so and offer to undo it. */
  protected readonly lastMark = signal<{ base: string; learned: boolean } | null>(null);
  protected readonly markError = signal(false);
  /** The row button that opened the details, focused again when they close. */
  private opener: HTMLElement | null = null;

  protected readonly learned = computed(() => {
    const p = this.progress();
    return new Set(p ? [...p.values()].filter((v) => isLearned(v)).map((v) => v.base) : []);
  });
  private readonly map = computed(() => this.progress() ?? new Map<string, VerbProgress>());
  protected readonly next = computed(() => nextLesson(VERB_LESSONS, this.map()));
  protected readonly review = computed(() => reviewSummary(this.map(), new Date()));
  /**
   * The ring: the common verbs learned while the next lesson is a common one, then every verb once
   * the learner has reached "Nâng cao".
   */
  protected readonly ring = computed(() => {
    const all = !this.next() || !!this.next()?.advanced;
    const lessons = VERB_LESSONS.filter((l) => all || !l.advanced);
    return {
      learned: lessons.reduce((n, l) => n + learnedIn(l, this.map()), 0),
      total: lessons.reduce((n, l) => n + l.bases.length, 0),
      label: all ? 'Động từ đã học' : 'Động từ thông dụng đã học',
    };
  });
  protected readonly lessons = computed(() => {
    const next = this.next();
    return VERB_LESSONS.map((l) => {
      const learned = learnedIn(l, this.map());
      return {
        ...l,
        model: modelForms(l),
        others: l.bases.slice(1).join(' · '),
        learned,
        done: learned === l.bases.length,
        current: l === next,
      };
    });
  });
  /** The path in its two parts: the common verbs, then "Nâng cao". */
  protected readonly parts = computed(() => [
    { title: 'Thông dụng', lessons: this.lessons().filter((l) => !l.advanced) },
    { title: 'Nâng cao', lessons: this.lessons().filter((l) => l.advanced) },
  ]);

  protected readonly shown = computed(() => {
    const verbs = filterVerbs(IRREGULAR_VERBS, {
      query: this.query(),
      commonOnly: this.commonOnly(),
      pattern: this.pattern(),
    });
    return this.learnedOnly() ? verbs.filter((v) => this.learned().has(v.base)) : verbs;
  });
  /** Grouped by letter, except while searching (the best match comes first then). */
  protected readonly groups = computed(() =>
    this.query().trim() ? [{ letter: '', verbs: this.shown() }] : byLetter(this.shown()),
  );
  /** What each pattern chip would show with the other filters as they are. */
  protected readonly counts = computed(() => {
    const verbs = filterVerbs(IRREGULAR_VERBS, { query: this.query(), commonOnly: this.commonOnly(), pattern: null });
    const out: Record<VerbPattern | 'all', number> = { all: verbs.length, AAA: 0, ABB: 0, ABA: 0, ABC: 0 };
    for (const v of verbs) {
      out[patternOf(v)]++;
    }
    return out;
  });

  constructor() {
    this.api.progress().subscribe({
      next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
      // The lists are still there; only the counts, the ticks and "Đã học" are missing.
      error: () => this.progress.set(null),
    });
  }

  protected setView(view: View): void {
    this.view.set(view);
    try {
      sessionStorage.setItem(VIEW_KEY, view);
    } catch {
      // Only a convenience.
    }
  }

  protected open(verb: IrregularVerb): void {
    this.opener = document.activeElement as HTMLElement | null;
    this.selected.set(verb);
  }

  protected closeSheet(): void {
    this.selected.set(null);
    this.opener?.focus();
    this.opener = null;
  }

  /** Marks a verb as learned, or takes it back out of the reviews. */
  protected setLearned(base: string, learned: boolean): void {
    this.markError.set(false);
    this.api.setLearned([base], learned).subscribe({
      next: (r) => {
        this.progress.update((m) => new Map([...(m ?? []), ...r.verbs.map((v) => [v.base, v] as const)]));
        const mark = { base, learned };
        this.lastMark.set(mark);
        // The note goes away by itself unless another verb was marked meanwhile.
        setTimeout(() => this.lastMark() === mark && this.lastMark.set(null), NOTE_MS);
      },
      error: () => this.markError.set(true),
    });
  }

  protected toggleLearned(base: string): void {
    this.setLearned(base, !this.learned().has(base));
  }

  protected undo(): void {
    const m = this.lastMark();
    if (m) {
      this.setLearned(m.base, !m.learned);
      this.lastMark.set(null);
    }
  }
}
