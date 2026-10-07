import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { VocabItem } from '../../../../core/models/vocab';
import { SpeechService } from '../../../../core/services/speech.service';
import { VocabApiService } from '../../../../core/services/vocab-api.service';
import { ReadingApiService } from '../../reading-api.service';
import { Loading } from '../../../../shared/components/loading/loading';
import { SpeakButton } from '../../../../shared/directives/speak-button';

const normalize = (s: string) => s.trim().replace(/\s+/g, ' ').toLowerCase();

/**
 * The "Từ vựng" section of the Reading step (F5): the lesson's annotated words with Save and
 * Save all. Collapsed by default; loads when first opened. Never calls the AI.
 */
@Component({
  selector: 'lu-lesson-vocabulary',
  imports: [Loading, SpeakButton],
  templateUrl: './lesson-vocabulary.html',
  styleUrl: './lesson-vocabulary.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonVocabulary {
  private readonly readingApi = inject(ReadingApiService);
  private readonly vocab = inject(VocabApiService);
  private readonly speech = inject(SpeechService);

  readonly lessonId = input.required<string>();
  /** Base forms already in the notebook (normalized), shared with the reading page. */
  readonly saved = input.required<ReadonlySet<string>>();
  /** Base forms just saved from this section. */
  readonly added = output<string[]>();

  protected readonly open = signal(false);
  protected readonly items = signal<VocabItem[] | null>(null);
  protected readonly available = signal(false);
  protected readonly loadError = signal(false);
  protected readonly saving = signal(false);
  protected readonly status = signal<string | null>(null);
  protected readonly error = signal<string | null>(null);

  protected readonly unsaved = computed(() =>
    (this.items() ?? []).filter((i) => !this.saved().has(normalize(i.lemma))).map((i) => i.lemma),
  );


  protected isSaved(item: VocabItem): boolean {
    return this.saved().has(normalize(item.lemma));
  }

  protected toggle(): void {
    this.open.update((o) => !o);
    if (this.open() && this.items() === null) {
      this.load();
    }
  }

  protected load(): void {
    this.loadError.set(false);
    this.readingApi.vocabulary(this.lessonId()).subscribe({
      next: (v) => {
        this.available.set(v.available);
        this.items.set(v.items);
      },
      error: () => this.loadError.set(true),
    });
  }

  protected play(item: VocabItem): void {
    this.speech.speak(item.lemma, 1);
  }

  protected save(lemmas: string[]): void {
    if (lemmas.length === 0 || this.saving()) {
      return;
    }
    void this.saveLemmas(lemmas);
  }

  private async saveLemmas(lemmas: string[]): Promise<void> {
    this.saving.set(true);
    this.status.set(null);
    this.error.set(null);
    try {
      const res = await firstValueFrom(this.vocab.bulk(this.lessonId(), lemmas));
      this.added.emit(lemmas.map(normalize));
      this.status.set(`Đã lưu ${res.added} từ`);
    } catch {
      this.error.set('Không lưu được, vui lòng thử lại.');
    } finally {
      this.saving.set(false);
    }
  }
}
