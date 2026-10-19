import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';

import { Icon } from '../../../shared/components/icon/icon';
import { PATTERNS, VerbPattern } from '../irregular-verbs.logic';

/**
 * Search and the filters in one row that scrolls sideways: all or common verbs, then the four
 * patterns of change (each with an example; tapping the chosen one again shows every pattern).
 */
@Component({
  selector: 'lu-verb-filters',
  imports: [Icon],
  templateUrl: './verb-filters.html',
  styleUrl: './verb-filters.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbFilters {
  readonly query = model('');
  readonly commonOnly = model(false);
  readonly pattern = model<VerbPattern | null>(null);
  /** Only the verbs the learner confirmed as learned. */
  readonly learnedOnly = model(false);

  readonly total = input(0);
  readonly commonTotal = input(0);
  /** How many verbs are learned; null hides the "Đã học" chip (the progress could not be loaded). */
  readonly learnedTotal = input<number | null>(null);
  /** How many verbs each pattern chip would show, and all of them. */
  readonly counts = input<Record<VerbPattern | 'all', number>>({
    all: 0,
    AAA: 0,
    ABB: 0,
    ABA: 0,
    ABC: 0,
  });

  protected readonly patterns = PATTERNS;

  protected onSearch(event: Event): void {
    this.query.set((event.target as HTMLInputElement).value);
  }

  protected clearSearch(input: HTMLInputElement): void {
    this.query.set('');
    input.value = '';
    input.focus();
  }

  protected togglePattern(key: VerbPattern): void {
    this.pattern.set(this.pattern() === key ? null : key);
  }
}
