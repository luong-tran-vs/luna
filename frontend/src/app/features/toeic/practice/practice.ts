import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

import { ToeicLibrary, ToeicPart } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { errorBody } from '../../admin/json-import/json-import';
import { PART_NAMES } from '../shared/score.logic';
import { ToeicTabs } from '../shared/toeic-tabs';
import { ToeicBack } from '../shared/toeic-back';

/** Parts a guest may practise (FR-045). */
const GUEST_PARTS: readonly ToeicPart[] = [1, 2, 5];

/** How many questions (Part 1, 2, 5) or groups (the others) a round may have. */
const COUNTS: Record<ToeicPart, number[]> = {
  1: [6, 10],
  2: [10, 20],
  3: [3, 5],
  4: [3, 5],
  5: [10, 20, 30],
  6: [2, 4],
  7: [3, 5, 8],
};

/** The time a round may take (minutes); a round always has a clock. */
const TIMES = [15, 20];

/**
 * "Luyện theo Part" (FR-029): pick a part and how many questions or texts; the questions come from
 * every published group, the ones not done yet first, in the exam room's practice mode.
 */
@Component({
  selector: 'lu-toeic-practice',
  imports: [ToeicTabs, ToeicBack],
  templateUrl: './practice.html',
  styleUrls: ['../shared/toeic-page.css', './practice.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Practice {
  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);

  protected readonly data = signal<ToeicLibrary | null>(null);
  protected readonly starting = signal<ToeicPart | null>(null);
  protected readonly error = signal('');
  protected readonly parts: ToeicPart[] = [1, 2, 3, 4, 5, 6, 7];
  protected readonly names = PART_NAMES;
  protected readonly counts = COUNTS;
  protected readonly chosen = signal<Partial<Record<ToeicPart, number>>>({});
  protected readonly times = TIMES;
  protected readonly chosenTime = signal<Partial<Record<ToeicPart, number>>>({});

  constructor() {
    this.api.library().subscribe({
      next: (d) => this.data.set(d),
      error: () => this.error.set('Không tải được. Kiểm tra mạng rồi thử lại.'),
    });
  }

  protected ready(p: ToeicPart): boolean {
    return (this.data()?.partCounts[p] ?? 0) > 0;
  }

  protected locked(p: ToeicPart): boolean {
    return (this.data()?.guest ?? false) && !GUEST_PARTS.includes(p);
  }

  protected unit(p: ToeicPart): string {
    return p === 1 || p === 2 || p === 5 ? 'câu' : p === 6 || p === 7 ? 'văn bản' : 'đoạn';
  }

  protected count(p: ToeicPart): number {
    return this.chosen()[p] ?? COUNTS[p][0];
  }

  protected pick(p: ToeicPart, event: Event): void {
    const n = Number((event.target as HTMLSelectElement).value);
    this.chosen.update((c) => ({ ...c, [p]: n }));
  }

  protected time(p: ToeicPart): number {
    return this.chosenTime()[p] ?? TIMES[0];
  }

  protected pickTime(p: ToeicPart, event: Event): void {
    const n = Number((event.target as HTMLSelectElement).value);
    this.chosenTime.update((c) => ({ ...c, [p]: n }));
  }

  protected start(p: ToeicPart): void {
    this.starting.set(p);
    this.error.set('');
    const source = { kind: 'part' as const, part: p, count: this.count(p) };
    this.api.start({ source, mode: 'practice', timeLimitMin: this.time(p) }).subscribe({
      next: (v) => void this.router.navigate(['/toeic/attempts', v.id]),
      error: (err: unknown) => {
        this.starting.set(null);
        this.error.set(errorBody(err).message ?? 'Chưa mở được lượt luyện, thử lại.');
      },
    });
  }
}
