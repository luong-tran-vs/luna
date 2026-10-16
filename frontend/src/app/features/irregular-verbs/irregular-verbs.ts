import { ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { VerbProgress } from '../../core/models/verbs';
import { VerbsApiService } from '../../core/services/verbs-api.service';
import { Icon } from '../../shared/components/icon/icon';
import { IRREGULAR_VERBS } from './irregular-verbs.data';
import { byLetter, filterVerbs, patternOf, VerbPattern } from './irregular-verbs.logic';
import { VerbFilters } from './verb-filters/verb-filters';
import { inSet, isKnown, ROUND_SIZE, summarize } from './verb-practice/verb-practice.logic';
import { VerbTable } from './verb-table/verb-table';

/**
 * The "Động từ" tab: a card leading to the practice (with how many common verbs are known), then
 * the irregular verbs to look up, with search (English forms or Vietnamese meaning, marks
 * optional), the common verbs only, the four patterns of change, an A–Z index and each verb read
 * aloud. A verb the learner knows has a tick.
 */
@Component({
  selector: 'lu-irregular-verbs',
  imports: [RouterLink, Icon, VerbFilters, VerbTable],
  templateUrl: './irregular-verbs.html',
  styleUrl: './irregular-verbs.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class IrregularVerbs {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  protected readonly total = IRREGULAR_VERBS.length;
  protected readonly commonTotal = IRREGULAR_VERBS.filter((v) => v.common).length;
  protected readonly roundSize = ROUND_SIZE;

  protected readonly query = signal('');
  protected readonly commonOnly = signal(false);
  protected readonly pattern = signal<VerbPattern | null>(null);
  /** The learner's records; null until loaded (or when they cannot be: the list works without). */
  private readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);

  protected readonly known = computed(() => {
    const p = this.progress();
    return new Set(p ? [...p.values()].filter((v) => isKnown(v)).map((v) => v.base) : []);
  });
  /** The common verbs known and due, for the practice card. */
  protected readonly summary = computed(() => {
    const p = this.progress();
    return p ? summarize(inSet(IRREGULAR_VERBS, 'common'), p, new Date()) : null;
  });

  protected readonly shown = computed(() =>
    filterVerbs(IRREGULAR_VERBS, {
      query: this.query(),
      commonOnly: this.commonOnly(),
      pattern: this.pattern(),
    }),
  );
  /** Grouped by letter, except while searching (the best match comes first then). */
  protected readonly groups = computed(() =>
    this.query().trim() ? [{ letter: '', verbs: this.shown() }] : byLetter(this.shown()),
  );
  protected readonly letters = computed(() =>
    this.groups()
      .map((g) => g.letter)
      .filter(Boolean),
  );
  /** What each pattern chip would show with the other filters as they are. */
  protected readonly counts = computed(() => {
    const verbs = filterVerbs(IRREGULAR_VERBS, {
      query: this.query(),
      commonOnly: this.commonOnly(),
      pattern: null,
    });
    const out: Record<VerbPattern | 'all', number> = {
      all: verbs.length,
      AAA: 0,
      ABB: 0,
      ABA: 0,
      ABC: 0,
    };
    for (const v of verbs) {
      out[patternOf(v)]++;
    }
    return out;
  });

  constructor() {
    inject(VerbsApiService)
      .progress()
      .subscribe({
        next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
        // The list is still there to look verbs up; only the counts and ticks are missing.
        error: () => this.progress.set(null),
      });
  }

  /** Scrolls to the verbs of a letter and moves focus there. */
  protected jump(letter: string): void {
    const el = this.host.nativeElement.querySelector<HTMLElement>(`#verbs-${letter}`);
    if (!el) {
      return;
    }
    const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    el.scrollIntoView?.({ behavior: reduce ? 'auto' : 'smooth', block: 'start' });
    el.focus({ preventScroll: true });
  }
}
