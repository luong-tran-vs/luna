import { ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal } from '@angular/core';

import { SpeechService } from '../../core/services/speech.service';
import { Icon } from '../../shared/components/icon/icon';
import { IRREGULAR_VERBS } from './irregular-verbs.data';
import { byLetter, filterVerbs, patternOf, VerbPattern } from './irregular-verbs.logic';
import { VerbFilters } from './verb-filters/verb-filters';
import { VerbTable } from './verb-table/verb-table';

/**
 * The "Động từ" tab: the irregular verbs with search (English forms or Vietnamese meaning, marks
 * optional), the common verbs only, the four patterns of change, an A–Z index, each verb read
 * aloud, and a self-test mode that hides V2 and V3 until tapped.
 */
@Component({
  selector: 'lu-irregular-verbs',
  imports: [Icon, VerbFilters, VerbTable],
  templateUrl: './irregular-verbs.html',
  styleUrl: './irregular-verbs.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class IrregularVerbs {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  protected readonly total = IRREGULAR_VERBS.length;
  protected readonly commonTotal = IRREGULAR_VERBS.filter((v) => v.common).length;
  protected readonly canSpeak = inject(SpeechService).supported;

  protected readonly query = signal('');
  protected readonly commonOnly = signal(false);
  protected readonly pattern = signal<VerbPattern | null>(null);
  protected readonly testing = signal(false);

  protected readonly shown = computed(() =>
    filterVerbs(IRREGULAR_VERBS, { query: this.query(), commonOnly: this.commonOnly(), pattern: this.pattern() }),
  );
  /** Grouped by letter, except while searching (the best match comes first then). */
  protected readonly groups = computed(() =>
    this.query().trim() ? [{ letter: '', verbs: this.shown() }] : byLetter(this.shown()),
  );
  protected readonly letters = computed(() => this.groups().map((g) => g.letter).filter(Boolean));
  /** What each pattern chip would show with the other filters as they are. */
  protected readonly counts = computed(() => {
    const verbs = filterVerbs(IRREGULAR_VERBS, { query: this.query(), commonOnly: this.commonOnly(), pattern: null });
    const out: Record<VerbPattern | 'all', number> = { all: verbs.length, AAA: 0, ABB: 0, ABA: 0, ABC: 0 };
    for (const v of verbs) {
      out[patternOf(v)]++;
    }
    return out;
  });

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
