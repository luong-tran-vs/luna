import {
  inject,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { SpeechService } from '../../../../core/services/speech.service';
import { VocabApiService } from '../../../../core/services/vocab-api.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';

/** An example sentence cut around the word being learnt; `hit` is empty when it is not found. */
export interface Example {
  sentence: string;
  before: string;
  hit: string;
  after: string;
}

/**
 * Stand-in picture for a word: a photo matched on the word, from a placeholder service. Words have
 * no picture of their own yet; `lock` keeps the same photo for the same word.
 */
export function placeholderImage(lemma: string): string {
  const tags = lemma.trim().toLowerCase().replace(/\s+/g, ',');
  let lock = 0;
  for (const ch of tags) {
    lock = (lock * 31 + ch.charCodeAt(0)) % 100000;
  }
  return `https://loremflickr.com/480/360/${encodeURIComponent(tags)}?lock=${lock}`;
}

/** Cuts `sentence` around the first whole-word match of one of `forms` (case ignored). */
export function highlight(sentence: string, forms: string[]): Example {
  const words = forms
    .map((f) => f.trim())
    .filter(Boolean)
    .sort((a, b) => b.length - a.length)
    .map((f) => f.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
  const match =
    words.length > 0 ? new RegExp(`\\b(${words.join('|')})\\b`, 'i').exec(sentence) : null;
  if (!match) {
    return { sentence, before: sentence, hit: '', after: '' };
  }
  const end = match.index + match[0].length;
  return {
    sentence,
    before: sentence.slice(0, match.index),
    hit: match[0],
    after: sentence.slice(end),
  };
}

/** Step 1: the lesson's words one card at a time, with a picture, IPA, meaning and an example. */
@Component({
  selector: 'lu-vocab-step',
  imports: [Icon, SpeakButton],
  templateUrl: './vocab-step.html',
  styleUrls: ['../practice.css', './vocab-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VocabStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  /** Name of the lesson's topic, shown next to the title; empty leaves it out. */
  readonly topic = input('');
  /** False when the browser has no voice: the listen buttons are hidden. */
  protected readonly canSpeak = inject(SpeechService).supported;

  /** The lesson's words; empty when the lesson has none. */
  readonly words = input.required<VocabItem[]>();
  /** Example sentences by lemma; a word without one falls back to its sentence in the text. */
  readonly examples = input<PracticeExample[]>([]);
  /** Label of the main button on the last word, where it leaves the step. */
  readonly finishLabel = input('Tiếp theo');
  /** The lesson the words come from: a card saved with Đã học belongs to it. */
  readonly lessonId = input.required<string>();

  private readonly vocab = inject(VocabApiService);
  /** Words confirmed with Đã học in this visit: their cards are in the review queue. */
  protected readonly learned = signal<ReadonlySet<string>>(new Set());
  /** The word being saved, or null. */
  protected readonly saving = signal<string | null>(null);
  /** Result of the last Đã học, read out by screen readers. */
  protected readonly learnStatus = signal<string | null>(null);
  protected readonly learnError = signal<string | null>(null);

  /** Text to read aloud with the browser's voice. */
  readonly readAloud = output<string>();
  /** Tiếp theo on the last word: go to the next step. */
  readonly done = output<void>();

  /** Index of the word on the card; back to the first one when the words change. */
  protected readonly index = linkedSignal({ source: this.words, computation: () => 0 });
  /** Words whose picture failed to load: they show the drawn placeholder instead. */
  private readonly brokenImages = signal<ReadonlySet<string>>(new Set());
  /**
   * The last picture that finished loading. The <img> is reused from word to word, and a browser
   * keeps showing the old picture until the new one has loaded: it stays hidden until then.
   */
  private readonly loadedImage = signal<string | null>(null);

  protected readonly rows = computed(() => {
    const byLemma = new Map(this.examples().map((e) => [e.lemma.toLowerCase(), e]));
    return this.words().map((w) => {
      const example = byLemma.get(w.lemma.toLowerCase());
      const sentence = example?.sentence ?? w.sentence ?? '';
      return {
        word: w,
        image: w.imageUrl || placeholderImage(w.lemma),
        example: sentence ? highlight(sentence, [w.text, w.lemma]) : null,
        // Only the generated example has a translation; the sentence of the text has none.
        exampleVi: example?.meaningVi ?? '',
      };
    });
  });

  protected readonly row = computed(() => this.rows()[this.index()] ?? null);
  protected readonly isFirst = computed(() => this.index() === 0);
  protected readonly isLast = computed(() => this.index() >= this.rows().length - 1);
  protected readonly percent = computed(() => {
    const total = this.rows().length;
    return total === 0 ? 0 : ((this.index() + 1) / total) * 100;
  });
  protected readonly showImage = computed(() => {
    const r = this.row();
    return r !== null && !this.brokenImages().has(r.word.lemma);
  });

  protected readonly imageLoaded = computed(() => this.row()?.image === this.loadedImage());

  constructor() {
    // Load the pictures of the words before and after, so moving to them shows theirs at once.
    effect(() => {
      const rows = this.rows();
      const i = this.index();
      if (typeof Image === 'undefined') {
        return;
      }
      for (const r of [rows[i + 1], rows[i - 1]]) {
        if (r && !this.brokenImages().has(r.word.lemma)) {
          const img = new Image();
          img.referrerPolicy = 'no-referrer';
          img.src = r.image;
        }
      }
    });
  }

  protected onImageLoad(src: string): void {
    this.loadedImage.set(src);
  }

  protected playWord(text: string): void {
    this.readAloud.emit(text);
  }

  protected previous(): void {
    if (!this.isFirst()) {
      this.index.update((i) => i - 1);
    }
  }

  protected forward(): void {
    if (!this.isLast()) {
      this.index.update((i) => i + 1);
    }
  }

  /** Tiếp theo: the next word, or the next step from the last word. */
  protected nextOrDone(): void {
    if (this.isLast()) {
      this.done.emit();
    } else {
      this.forward();
    }
  }

  /** Đã học: saves the word as a review card (a word already in the notebook stays as it is). */
  protected async learn(lemma: string): Promise<void> {
    if (this.saving() !== null || this.learned().has(lemma)) {
      return;
    }
    this.saving.set(lemma);
    this.learnStatus.set(null);
    this.learnError.set(null);
    try {
      const res = await firstValueFrom(this.vocab.bulk(this.lessonId(), [lemma]));
      this.learned.update((s) => new Set(s).add(lemma));
      this.learnStatus.set(
        res.added > 0 ? `Đã lưu "${lemma}" vào thẻ ôn tập.` : `"${lemma}" đã có trong thẻ ôn tập.`,
      );
    } catch {
      this.learnError.set('Không lưu được, vui lòng thử lại.');
    } finally {
      this.saving.set(null);
    }
  }

  protected imageFailed(lemma: string): void {
    this.brokenImages.update((s) => new Set(s).add(lemma));
  }
}
