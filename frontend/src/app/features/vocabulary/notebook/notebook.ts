import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal, viewChild } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Subscription } from 'rxjs';

import { Card, DayCard, LessonCounts } from '../../../core/models/vocab';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import { AudioPlayer } from '../../../shared/components/audio-player/audio-player';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { Icon } from '../../../shared/components/icon/icon';
import { CardForm } from '../card-form/card-form';

interface DayGroup {
  day: string;
  label: string;
  cards: DayCard[];
}

const SEARCH_DELAY = 300;

/** The notebook (F5): cards grouped by the day they were saved, search, lesson filter, edit. */
@Component({
  selector: 'lu-notebook',
  imports: [AudioPlayer, CardForm, ConfirmDialog, Icon, RouterLink],
  templateUrl: './notebook.html',
  styleUrl: './notebook.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Notebook {
  private readonly api = inject(VocabApiService);

  protected readonly cards = signal<DayCard[]>([]);
  protected readonly today = signal('');
  protected readonly yesterday = signal('');
  protected readonly hasMore = signal(false);
  protected readonly loading = signal(true);
  protected readonly loadError = signal<string | null>(null);
  protected readonly q = signal('');
  protected readonly lessonId = signal('');
  protected readonly lessons = signal<LessonCounts>({ lessons: [], manualCount: 0 });
  protected readonly dueCount = signal<number | null>(null);
  protected readonly adding = signal(false);
  protected readonly editingId = signal<string | null>(null);
  protected readonly deleting = signal<Card | null>(null);
  protected readonly actionError = signal<string | null>(null);

  protected readonly filtered = computed(() => this.q().trim() !== '' || this.lessonId() !== '');
  protected readonly groups = computed<DayGroup[]>(() => {
    const groups: DayGroup[] = [];
    for (const c of this.cards()) {
      const last = groups.at(-1);
      if (last?.day === c.day) {
        last.cards.push(c);
      } else {
        groups.push({ day: c.day, label: this.dayLabel(c.day), cards: [c] });
      }
    }
    return groups;
  });

  private readonly player = viewChild(AudioPlayer);
  private page = 1;
  private listSub?: Subscription;
  private searchTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    this.loadPage(1);
    this.loadLessons();
    this.api.due(1).subscribe({ next: (d) => this.dueCount.set(d.total), error: () => this.dueCount.set(null) });
    inject(DestroyRef).onDestroy(() => {
      this.listSub?.unsubscribe();
      clearTimeout(this.searchTimer);
    });
  }

  private dayLabel(day: string): string {
    if (day === this.today()) {
      return 'Hôm nay';
    }
    if (day === this.yesterday()) {
      return 'Hôm qua';
    }
    const [y, m, d] = day.split('-');
    return `${d}/${m}/${y}`;
  }

  private loadPage(page: number): void {
    this.listSub?.unsubscribe();
    this.loading.set(true);
    this.loadError.set(null);
    this.listSub = this.api.list({ q: this.q().trim(), lessonId: this.lessonId(), page }).subscribe({
      next: (res) => {
        this.page = page;
        this.today.set(res.today);
        this.yesterday.set(res.yesterday);
        this.cards.update((cards) => (page === 1 ? res.cards : [...cards, ...res.cards]));
        this.hasMore.set(res.hasMore);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.loadError.set('Không tải được sổ từ.');
      },
    });
  }

  private loadLessons(): void {
    this.api.lessons().subscribe({ next: (l) => this.lessons.set(l), error: () => undefined });
  }

  protected more(): void {
    this.loadPage(this.page + 1);
  }

  protected onSearch(event: Event): void {
    this.q.set((event.target as HTMLInputElement).value);
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => this.loadPage(1), SEARCH_DELAY);
  }

  protected onLessonChange(event: Event): void {
    this.lessonId.set((event.target as HTMLSelectElement).value);
    this.loadPage(1);
  }

  protected play(card: Card): void {
    this.player()?.replay(this.api.wordAudioUrl(card.text));
  }

  protected onAdded(): void {
    this.adding.set(false);
    this.loadPage(1);
    this.loadLessons();
  }

  protected onEdited(updated: Card): void {
    this.editingId.set(null);
    this.cards.update((cards) => cards.map((c) => (c.id === updated.id ? { ...c, ...updated } : c)));
  }

  protected confirmDelete(): void {
    const card = this.deleting();
    if (!card) {
      return;
    }
    this.deleting.set(null);
    this.actionError.set(null);
    this.api.remove(card.id).subscribe({
      next: () => {
        this.cards.update((cards) => cards.filter((c) => c.id !== card.id));
        this.loadLessons();
      },
      error: () => this.actionError.set(`Không xoá được “${card.text}”, vui lòng thử lại.`),
    });
  }
}
