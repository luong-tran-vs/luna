import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { GrammarList as GrammarListData, GrammarListItem, STATUS_LABELS } from '../../../core/models/grammar-study';
import { Level, LEVELS } from '../../../core/models/lesson';
import { GrammarApiService } from '../../../core/services/grammar-api.service';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { savedStep, STEPS } from '../grammar-steps';

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

/**
 * Grammar points of one level with the learner's status on each (F20): a card with the point to
 * study next and how much of the level is mastered, the open points, then the
 * ones without content yet gathered in one collapsed "Sắp có" row; curriculum order within each.
 */
@Component({
  selector: 'lu-grammar-list',
  imports: [NgOptimizedImage, RouterLink, Icon, Loading],
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
  protected readonly openPoints = computed(() => this.points().filter((p) => p.available));
  protected readonly soonPoints = computed(() => this.points().filter((p) => !p.available));
  /** Whether the "Sắp có" row is expanded; closed again for every level. */
  protected readonly showSoon = signal(false);
  protected readonly nextPoint = computed(() => {
    const id = this.data()?.next;
    return id ? (this.points().find((p) => p.id === id) ?? null) : null;
  });
  /** Place of the next point in the level's curriculum, from 1. */
  protected readonly nextNumber = computed(() => this.points().findIndex((p) => p.id === this.data()?.next) + 1);
  /**
   * The card's button goes straight to what is left on the next point: redo the wrong answers,
   * take the test, or the step where the learner stopped in this browser; else the point's start.
   */
  protected readonly nextAction = computed((): { label: string; query: Record<string, string> | null } => {
    const n = this.nextPoint();
    if (n && (n.weakCount ?? 0) > 0) {
      return { label: `Luyện lại ${n.weakCount} câu sai`, query: { step: 'practice', redo: 'weak' } };
    }
    if (n?.status === 'learning') {
      return { label: 'Làm bài kiểm tra', query: { step: 'test' } };
    }
    const at = n ? savedStep(n.id) : 0;
    return at > 0
      ? { label: `Tiếp tục: Bước ${at + 1}`, query: { step: STEPS[at].key } }
      : { label: 'Bắt đầu học', query: null };
  });
  protected readonly masteredCount = computed(() => this.points().filter((p) => p.status === 'mastered').length);

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
    this.showSoon.set(false);
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
    if (this.stateTone(p) === 'weak') {
      return `${p.weakCount} câu sai cần luyện`;
    }
    return p.status === 'mastered' && p.bestMastery > 0 ? `${label} · ${p.bestMastery}%` : label;
  }

  /** Colour of the status: a point still being learnt with wrong answers to redo stands out. */
  protected stateTone(p: GrammarListItem): 'mastered' | 'weak' | 'learning' | 'new' {
    if (p.status === 'learning' && (p.weakCount ?? 0) > 0) {
      return 'weak';
    }
    return p.status;
  }
}
