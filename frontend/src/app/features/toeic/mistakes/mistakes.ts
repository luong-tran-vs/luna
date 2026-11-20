import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { CheckReply, MistakesView, RoundItem, ToeicPart } from '../../../core/models/toeic';
import { AuthService } from '../../../core/services/auth.service';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { DocView } from '../shared/doc-view';
import { QuestionView } from '../shared/question-view';
import { letter, PART_NAMES } from '../shared/score.logic';
import { ToeicTabs } from '../shared/toeic-tabs';

/** How many questions a round of the notebook has. */
const ROUND_SIZE = 10;

/**
 * The mistakes notebook (FR-031): the wrong and skipped questions, filtered by part, with how many
 * right answers in a row each has (●○). "Làm lại 10 câu" grades each question at once, showing its
 * whole group; two right in a row take it out. The end of a round counts as a study day.
 */
@Component({
  selector: 'lu-toeic-mistakes',
  imports: [RouterLink, DocView, QuestionView, ToeicTabs],
  templateUrl: './mistakes.html',
  styleUrls: ['../shared/toeic-page.css', './mistakes.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Mistakes {
  private readonly api = inject(ToeicApiService);
  protected readonly guest = inject(AuthService).isGuest;

  protected readonly view = signal<MistakesView | null>(null);
  protected readonly error = signal('');
  protected readonly part = signal<ToeicPart | null>(null);
  protected readonly names = PART_NAMES;
  protected readonly parts: ToeicPart[] = [1, 2, 3, 4, 5, 6, 7];

  protected readonly round = signal<RoundItem[] | null>(null);
  protected readonly index = signal(0);
  protected readonly choice = signal<number | null>(null);
  protected readonly reply = signal<CheckReply | null>(null);
  protected readonly checking = signal(false);
  protected readonly left = signal(0);

  protected readonly total = computed(() =>
    Object.values(this.view()?.counts ?? {}).reduce((a, b) => a + b, 0),
  );
  protected readonly item = computed(() => this.round()?.[this.index()] ?? null);
  protected readonly finished = computed(() => {
    const r = this.round();
    return r !== null && this.index() >= r.length;
  });

  constructor() {
    if (!this.guest()) {
      this.load();
    }
  }

  protected load(): void {
    this.error.set('');
    this.api.mistakes(this.part() ?? undefined).subscribe({
      next: (v) => this.view.set(v),
      error: () => this.error.set('Không tải được sổ câu sai. Kiểm tra mạng rồi thử lại.'),
    });
  }

  protected filter(p: ToeicPart | null): void {
    this.part.set(p);
    this.load();
  }

  protected count(p: ToeicPart): number {
    return this.view()?.counts[String(p)] ?? 0;
  }

  protected dots(streak: number): string {
    const need = this.view()?.need ?? 2;
    return '●'.repeat(streak) + '○'.repeat(Math.max(0, need - streak));
  }

  protected startRound(): void {
    this.error.set('');
    this.api.mistakeRound(this.part(), ROUND_SIZE).subscribe({
      next: (r) => {
        this.round.set(r.items);
        this.index.set(0);
        this.left.set(0);
        this.reset();
      },
      error: () => this.error.set('Chưa mở được lượt làm lại, thử lại.'),
    });
  }

  private reset(): void {
    this.choice.set(null);
    this.reply.set(null);
  }

  protected answer(i: number): void {
    const it = this.item();
    if (!it || this.reply() || this.checking()) {
      return;
    }
    this.choice.set(i);
    this.checking.set(true);
    this.api.checkMistake(it.questionId, i).subscribe({
      next: (r) => {
        this.checking.set(false);
        this.reply.set(r);
        if (r.removed) {
          this.left.update((n) => n + 1);
        }
      },
      error: () => {
        this.checking.set(false);
        this.choice.set(null);
        this.error.set('Chưa chấm được, thử lại.');
      },
    });
  }

  protected next(): void {
    this.index.update((i) => i + 1);
    this.reset();
    if (this.finished()) {
      this.api.finishMistakes().subscribe({ error: () => undefined });
    }
  }

  protected close(): void {
    this.round.set(null);
    this.load();
  }

  protected letterOf(i: number): string {
    return letter(i);
  }

  protected image(url: string | null): string | null {
    return url ? serverUrl(url) : null;
  }

  protected blanks(it: RoundItem): Record<number, number> {
    const q = it.group.questions[0];
    return q ? { [q.no]: q.number } : {};
  }
}
