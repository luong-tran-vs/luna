import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { TestCard, ToeicLibrary } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { kindLabel, shortDate } from '../shared/score.logic';
import { ToeicTabs } from '../shared/toeic-tabs';

/**
 * The TOEIC home (FR-002): the tabs, the test cards with the learner's state on each (new, in
 * progress at question N, done with the latest result) and a side column with the goal, the
 * latest estimated score and the recent sittings. A guest sees the members' tests locked.
 */
@Component({
  selector: 'lu-toeic-library',
  imports: [RouterLink, Icon, ToeicTabs],
  templateUrl: './library.html',
  styleUrls: ['../shared/toeic-page.css', './library.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Library {
  private readonly api = inject(ToeicApiService);

  protected readonly data = signal<ToeicLibrary | null>(null);
  protected readonly error = signal(false);
  protected readonly tests = computed(() => this.data()?.tests ?? []);

  constructor() {
    this.load();
  }

  protected load(): void {
    this.error.set(false);
    this.api.library().subscribe({
      next: (d) => this.data.set(d),
      error: () => this.error.set(true),
    });
  }

  protected kind(t: TestCard): string {
    return kindLabel(t.kind);
  }

  protected result(t: TestCard): string {
    const s = t.state;
    if (s.total !== null) {
      return `${s.total} điểm (ước tính)`;
    }
    if (s.correct !== null) {
      return `${s.correct}/${s.questions ?? t.questions} câu đúng`;
    }
    return '';
  }

  protected date(at: string | null): string {
    return at ? shortDate(at) : '';
  }
}
