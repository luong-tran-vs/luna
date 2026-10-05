import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { GrammarList as GrammarListData, GrammarListItem, STATUS_LABELS } from '../../../core/models/grammar-study';
import { Level, LEVELS } from '../../../core/models/lesson';
import { GrammarApiService } from '../../../core/services/grammar-api.service';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';

const STORAGE_KEY = 'luna.grammar.level';

/** The level chosen earlier in this browser session, if any. */
function savedLevel(): Level | null {
  try {
    const v = sessionStorage.getItem(STORAGE_KEY);
    return LEVELS.find((l) => l === v) ?? null;
  } catch {
    return null;
  }
}

function saveLevel(level: Level): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, level);
  } catch {
    // The choice is only a convenience.
  }
}

/** Grammar points of one level in curriculum order, with the learner's status on each (F20). */
@Component({
  selector: 'lu-grammar-list',
  imports: [RouterLink, Icon, Loading],
  templateUrl: './grammar-list.html',
  styleUrl: './grammar-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarList {
  private readonly api = inject(GrammarApiService);

  protected readonly levels = LEVELS;
  protected readonly statusLabels = STATUS_LABELS;
  protected readonly level = signal<Level | null>(null);
  protected readonly data = signal<GrammarListData | null>(null);
  protected readonly failed = signal(false);

  protected readonly points = computed(() => this.data()?.points ?? []);
  protected readonly nextPoint = computed(() => {
    const id = this.data()?.next;
    return id ? (this.points().find((p) => p.id === id) ?? null) : null;
  });

  constructor() {
    const saved = savedLevel();
    if (saved) {
      this.select(saved, false);
    } else {
      // Start at the level of the goal being studied; A1 when there is none.
      inject(StudyApiService)
        .goals()
        .subscribe({
          next: (g) => this.select(g.active?.level ?? 'A1', false),
          error: () => this.select('A1', false),
        });
    }
  }

  protected select(level: Level, remember = true): void {
    this.level.set(level);
    if (remember) {
      saveLevel(level);
    }
    this.load();
  }

  protected load(): void {
    const level = this.level();
    if (!level) {
      return;
    }
    this.data.set(null);
    this.failed.set(false);
    this.api.points(level).subscribe({
      // Ignore an answer for a level that is no longer the chosen one.
      next: (d) => this.level() === level && this.data.set(d),
      error: () => this.level() === level && this.failed.set(true),
    });
  }

  protected onLevelKeydown(event: KeyboardEvent, index: number): void {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const next = (index + step + LEVELS.length) % LEVELS.length;
    this.select(LEVELS[next]);
    const group = (event.currentTarget as HTMLElement).closest('[role="radiogroup"]');
    group?.querySelectorAll<HTMLElement>('[role="radio"]')[next]?.focus();
  }

  protected statusText(p: GrammarListItem): string {
    const label = STATUS_LABELS[p.status];
    return p.status === 'mastered' && p.bestMastery > 0 ? `${label} · ${p.bestMastery}%` : label;
  }
}
