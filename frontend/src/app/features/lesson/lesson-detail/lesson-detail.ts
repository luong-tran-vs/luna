import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, forkJoin, map, of } from 'rxjs';

import { PracticeView } from '../../../core/models/practice';
import { ReadingLesson } from '../../../core/models/reading';
import { MyLessons } from '../../../core/models/study';
import { LessonVocabulary } from '../../../core/models/vocab';
import { DashboardApiService } from '../../../core/services/dashboard-api.service';
import { StudyApiService } from '../../../core/services/study-api.service';
import { AudioPlayer } from '../../../shared/components/audio-player/audio-player';
import { Icon } from '../../../shared/components/icon/icon';
import { loadErrorMessage } from '../load-error';
import { ReadingApiService } from '../reading-api.service';
import { DialogueStep } from './dialogue-step/dialogue-step';
import { FillStep } from './fill-step/fill-step';
import {
  hasDialogue,
  hasFill,
  hasTranslations,
  levelLabel,
  newSeed,
  Score,
  shuffle,
  summary,
} from './practice-logic';
import { LessonStatus, PracticeSummary } from './practice-summary/practice-summary';
import { TranslateStep } from './translate-step/translate-step';
import { VocabStep } from './vocab-step/vocab-step';

type Tab = 'lesson' | 'reading';

const STEPS = 4;
const STEP_NAMES = [
  'Từ vựng quan trọng',
  'Hội thoại mẫu',
  'Điền vào ô trống',
  'Dịch câu sang tiếng Anh',
];

/**
 * A lesson's detail page (F17): header, then the Bài học tab with 4 practice steps (words,
 * dialogue, fill-in, translation) and a summary, or the Bài đọc tab with the text and grammar note.
 * Practice results stay on this page only: nothing is sent to the server.
 */
@Component({
  selector: 'lu-lesson-detail',
  imports: [
    AudioPlayer,
    DialogueStep,
    FillStep,
    Icon,
    PracticeSummary,
    RouterLink,
    TranslateStep,
    VocabStep,
  ],
  templateUrl: './lesson-detail.html',
  styleUrl: './lesson-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonDetail {
  private readonly reading = inject(ReadingApiService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly player = viewChild.required(AudioPlayer);

  protected readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';
  protected readonly lesson = signal<ReadingLesson | null>(null);
  protected readonly vocabulary = signal<LessonVocabulary | null>(null);
  protected readonly practice = signal<PracticeView | null>(null);
  protected readonly mine = signal<MyLessons | null>(null);
  protected readonly streak = signal<number | null>(null);
  protected readonly loadError = signal<string | null>(null);

  protected readonly steps = Array.from({ length: STEPS }, (_, i) => i + 1);
  protected readonly tab = signal<Tab>('lesson');
  protected readonly step = signal(1);
  protected readonly translateIndex = signal(0);
  protected readonly fillScore = signal<Score | null>(null);
  protected readonly translateResults = signal<(boolean | null)[]>([]);
  protected readonly finished = signal(false);
  protected readonly rate = signal(1);
  private readonly seed = signal(newSeed());

  protected readonly status = computed<LessonStatus>(() => {
    const m = this.mine();
    if (m?.today?.id === this.id) {
      return 'today';
    }
    return m?.completed.some((c) => c.id === this.id) ? 'completed' : 'other';
  });
  protected readonly level = computed(() => {
    const l = this.lesson();
    return l ? levelLabel(l.level) : '';
  });
  protected readonly words = computed(() => {
    const v = this.vocabulary();
    return v?.available ? v.items : [];
  });
  protected readonly hasDialogue = computed(() => hasDialogue(this.practice()));
  protected readonly hasFill = computed(() => hasFill(this.practice()));
  protected readonly hasTranslations = computed(() => hasTranslations(this.practice()));

  /** Word banks are shuffled from the seed; Làm lại picks a new seed. */
  protected readonly fillBank = computed(() =>
    shuffle(this.practice()?.fill?.wordBank ?? [], this.seed()),
  );
  protected readonly translationTiles = computed(() =>
    (this.practice()?.translations ?? []).map((t, i) => shuffle(t.tiles, this.seed() + i + 1)),
  );
  protected readonly turnAudio = computed(
    () => this.practice()?.dialogue?.turns.map((t) => t.audioUrl) ?? [],
  );

  protected readonly progress = computed(() => (this.finished() ? STEPS : this.step()));
  protected readonly stepAnnouncement = computed(() =>
    this.finished() ? 'Tổng kết' : `Bước ${this.step()}/${STEPS}: ${STEP_NAMES[this.step() - 1]}`,
  );
  protected readonly lastSentence = computed(
    () => this.translateIndex() >= (this.practice()?.translations.length ?? 0) - 1,
  );
  protected readonly nextLabel = computed(() =>
    this.step() === STEPS && this.lastSentence() ? 'Hoàn thành' : 'Tiếp theo',
  );
  protected readonly summary = computed(() =>
    summary(this.practice()?.fill?.blanks.length ?? 0, this.fillScore(), this.translateResults()),
  );

  /** The text as paragraphs of sentences, for the Bài đọc tab. */
  protected readonly paragraphs = computed(() => {
    const l = this.lesson();
    if (!l) {
      return [];
    }
    const text = new Map(l.sentences.map((s) => [s.index, s.text]));
    const groups = l.paragraphs.length ? l.paragraphs : [l.sentences.map((s) => s.index)];
    return groups.map((p) => p.map((i) => text.get(i) ?? '').join(' '));
  });

  constructor() {
    forkJoin({
      lesson: this.reading.getLesson(this.id),
      vocabulary: this.reading.vocabulary(this.id).pipe(catchError(() => of(null))),
      practice: this.reading.practice(this.id).pipe(catchError(() => of(null))),
      mine: inject(StudyApiService)
        .myLessons()
        .pipe(catchError(() => of(null))),
      // Not GET /api/today: it has side effects.
      streak: inject(DashboardApiService)
        .dashboard()
        .pipe(
          map((d) => d.streak),
          catchError(() => of(null)),
        ),
    }).subscribe({
      next: (r) => {
        this.vocabulary.set(r.vocabulary);
        this.practice.set(r.practice);
        this.mine.set(r.mine);
        this.streak.set(r.streak);
        this.translateResults.set((r.practice?.translations ?? []).map(() => null));
        this.lesson.set(r.lesson);
      },
      error: (err: unknown) => this.loadError.set(loadErrorMessage(err)),
    });
  }

  protected play(url: string): void {
    this.player().replay(url);
  }

  /** Audio keeps playing across tabs; the Bài học tab stays as it was. */
  protected selectTab(tab: Tab): void {
    this.tab.set(tab);
  }

  protected onTabKeydown(event: KeyboardEvent): void {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') {
      return;
    }
    event.preventDefault();
    const tab: Tab = this.tab() === 'lesson' ? 'reading' : 'lesson';
    this.selectTab(tab);
    const list = (event.currentTarget as HTMLElement).closest('[role="tablist"]');
    list?.querySelector<HTMLElement>(`#tab-${tab}`)?.focus();
  }

  protected onFillChecked(score: Score): void {
    this.fillScore.set(score);
  }

  protected onTranslationChecked(index: number, ok: boolean): void {
    this.translateResults.update((r) => r.map((v, i) => (i === index ? ok : v)));
  }

  protected next(): void {
    this.player().stop();
    if (this.step() < STEPS) {
      this.step.update((s) => s + 1);
    } else if (!this.lastSentence()) {
      this.translateIndex.update((i) => i + 1);
    } else {
      this.finished.set(true);
    }
    this.toTop();
  }

  /** Làm lại: back to step 1 with every answer cleared and the banks shuffled again. */
  protected retry(): void {
    this.player().stop();
    this.finished.set(false);
    this.step.set(1);
    this.translateIndex.set(0);
    this.fillScore.set(null);
    this.translateResults.set((this.practice()?.translations ?? []).map(() => null));
    this.seed.set(newSeed());
    this.toTop();
  }

  private toTop(): void {
    this.host.nativeElement.scrollIntoView?.({ block: 'start' });
  }
}
