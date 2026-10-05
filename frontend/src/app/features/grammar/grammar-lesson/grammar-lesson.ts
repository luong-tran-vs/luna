import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AttemptResult, GrammarDetail, STATUS_LABELS } from '../../../core/models/grammar-study';
import { GrammarApiService } from '../../../core/services/grammar-api.service';
import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { GrammarQuiz } from '../grammar-quiz/grammar-quiz';

type TabKey = 'learn' | 'practice' | 'test';

const TABS: readonly { key: TabKey; label: string }[] = [
  { key: 'learn', label: 'Học' },
  { key: 'practice', label: 'Luyện tập' },
  { key: 'test', label: 'Kiểm tra' },
];

/** One grammar point (F20): learn it, practise it, then pass the mastery test. */
@Component({
  selector: 'lu-grammar-lesson',
  imports: [RouterLink, Icon, Loading, GrammarQuiz],
  templateUrl: './grammar-lesson.html',
  styleUrl: './grammar-lesson.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarLesson {
  private readonly api = inject(GrammarApiService);
  private readonly speech = inject(SpeechService);
  private readonly route = inject(ActivatedRoute);

  protected readonly tabs = TABS;
  protected readonly statusLabels = STATUS_LABELS;
  protected readonly canSpeak = this.speech.supported;
  protected readonly detail = signal<GrammarDetail | null>(null);
  /** 'notFound' when the point has no published lesson; 'failed' for any other error. */
  protected readonly error = signal<'notFound' | 'failed' | null>(null);
  protected readonly tab = signal<TabKey>('learn');
  protected readonly pointId = signal('');
  protected readonly content = computed(() => this.detail()?.content ?? null);

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((p) => {
      this.pointId.set(p.get('pointId') ?? '');
      this.load();
    });
  }

  protected load(): void {
    this.detail.set(null);
    this.error.set(null);
    this.api.get(this.pointId()).subscribe({
      next: (d) => this.detail.set(d),
      error: (err: unknown) => this.error.set(err instanceof ApiError && err.status === 404 ? 'notFound' : 'failed'),
    });
  }

  protected select(key: TabKey): void {
    this.tab.set(key);
  }

  protected onTabKeydown(event: KeyboardEvent, index: number): void {
    const last = TABS.length - 1;
    const target = {
      ArrowRight: index === last ? 0 : index + 1,
      ArrowLeft: index === 0 ? last : index - 1,
      Home: 0,
      End: last,
    }[event.key];
    if (target === undefined) {
      return;
    }
    event.preventDefault();
    this.select(TABS[target].key);
    (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLElement>('[role="tab"]')[target]?.focus();
  }

  protected listen(text: string): void {
    this.speech.speak(text, 0.9);
  }

  protected onRecorded(result: AttemptResult): void {
    this.detail.update((d) => (d ? { ...d, progress: result.progress } : d));
  }
}
