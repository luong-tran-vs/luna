import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { Icon } from '../../../shared/components/icon/icon';
import { groupLearned, todayTasks, verbGroups } from '../verb-groups.logic';
import { strugglingVerbs } from '../verb-mistakes.logic';
import { nextLesson } from '../verb-study.logic';
import { VERB_LESSONS } from '../verb-lessons.data';

/**
 * The top of the "Động từ" tab's Học part (design/de-xuat-dong-tu.png parts 6–7, added 2026-11-10):
 * today's tasks ticked off as they are done, the verbs often typed wrong, and the map of the groups
 * with how many of each are learned. Each opens its page.
 */
@Component({
  selector: 'lu-verb-today',
  imports: [RouterLink, Icon],
  templateUrl: './verb-today.html',
  styleUrl: './verb-today.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbToday {
  readonly progress = input.required<ReadonlyMap<string, VerbProgress>>();
  /** The picture URL of each verb that has one. */
  readonly pictures = input<ReadonlyMap<string, string>>(new Map());

  private readonly now = new Date();
  protected readonly tasks = computed(() => todayTasks(this.progress(), this.now));
  protected readonly done = computed(() => this.tasks().filter((t) => t.done).length);
  protected readonly struggling = computed(() => strugglingVerbs(this.progress(), this.now));
  protected readonly next = computed(() => nextLesson(VERB_LESSONS, this.progress()));
  protected readonly groups = computed(() =>
    verbGroups().map((g, i) => {
      const learned = groupLearned(g, this.progress());
      return {
        ...g,
        learned,
        tone: i % 4,
        percent: Math.round((learned / g.bases.length) * 100),
      };
    }),
  );
  protected readonly learnedTotal = computed(() =>
    this.groups().reduce((n, g) => n + g.learned, 0),
  );
  protected readonly verbTotal = computed(() =>
    this.groups().reduce((n, g) => n + g.bases.length, 0),
  );

  /** Where a task leads. */
  protected link(key: string): string[] {
    const next = this.next();
    switch (key) {
      case 'review':
        return ['/irregular-verbs/review'];
      case 'mistakes':
        return ['/irregular-verbs/mistakes'];
      default:
        return next ? ['/irregular-verbs/lessons', String(next.number)] : ['/irregular-verbs'];
    }
  }

  protected strugglingBases(): string {
    return this.struggling()
      .slice(0, 5)
      .map((s) => s.verb.base)
      .join(', ');
  }
}
