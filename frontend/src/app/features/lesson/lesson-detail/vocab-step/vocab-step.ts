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
import { serverUrl } from '../../../../core/api-url';
import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';
import { posLabel } from '../../../../shared/utils/part-of-speech';
import { newSeed } from '../practice-logic';
import { buildRecall, RecallItem } from './vocab-recall';

/**
 * The end of the words: the lemmas marked "Đã biết" (left out of the review queue) and those got wrong
 * in the quick recall.
 */
export interface VocabResult {
  known: string[];
  missed: string[];
}

/** An example sentence cut around the word being learnt; `hit` is empty when it is not found. */
export interface Example {
  sentence: string;
  before: string;
  hit: string;
  after: string;
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

/**
 * Step 1: the lesson's words one card at a time, with a picture, IPA, meaning and an example, then a
 * quick recall (six-step lesson, 2026-10-22): each word asked once, by its sound or its meaning, among
 * four. Every word goes to the review queue (the page saves them) unless marked "Đã biết".
 */
@Component({
  selector: 'lu-vocab-step',
  imports: [Icon, SpeakButton],
  templateUrl: './vocab-step.html',
  styleUrls: ['../practice.css', './vocab-step.css'],
  host: { '[class.has-picture]': 'showImage()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VocabStep {
  /** Position of this step on the page, shown in its title (steps without content are left out). */
  readonly number = input(1);
  /** Name of the lesson's topic, shown next to the title; empty leaves it out. */
  readonly topic = input('');
  /** False when the browser has no voice: the listen buttons are hidden. */
  protected readonly canSpeak = inject(SpeechService).supported;
  protected readonly posLabel = posLabel;

  /** The lesson's words; empty when the lesson has none. */
  readonly words = input.required<VocabItem[]>();
  /** Example sentences by lemma; a word without one falls back to its sentence in the text. */
  readonly examples = input<PracticeExample[]>([]);
  /** Label of the main button on the last word, where it leaves the step. */
  readonly finishLabel = input('Tiếp theo');
  /** The lesson the words come from. */
  readonly lessonId = input.required<string>();
  /** Whether ← on the first word goes to the step before (or the overview). */
  readonly canBack = input(false);

  /** Words marked "Đã biết": kept out of the review queue and of the recall. */
  protected readonly known = linkedSignal<VocabItem[], ReadonlySet<string>>({
    source: this.words,
    computation: () => new Set(),
  });

  /** Text to read aloud with the browser's voice. */
  readonly readAloud = output<string>();
  /** ← on the first word. */
  readonly back = output<void>();
  /** The end of the recall: go to the next step. */
  readonly done = output<VocabResult>();

  /** 'cards' while going through the words, then 'recall'. */
  protected readonly phase = linkedSignal<VocabItem[], 'cards' | 'recall'>({
    source: this.words,
    computation: () => 'cards',
  });
  protected readonly recall = signal<RecallItem[]>([]);
  protected readonly recallIndex = signal(0);
  /** The lemma picked for the question on screen, null until one is picked. */
  protected readonly picked = signal<string | null>(null);
  /** Lemmas got wrong at the first try; their question comes again at the end. */
  private readonly missed = new Set<string>();
  /** Whether the words end with the recall: it needs two words to choose between. */
  protected readonly canRecall = computed(() => this.words().filter((w) => !this.known().has(w.lemma)).length >= 2);
  protected readonly question = computed(() => this.recall()[this.recallIndex()] ?? null);
  protected readonly recallPercent = computed(() => {
    const total = this.recall().length;
    return total === 0 ? 0 : ((this.recallIndex() + 1) / total) * 100;
  });
  protected readonly pickedRight = computed(() => {
    const q = this.question();
    return q !== null && this.picked() === q.word.lemma;
  });

  /** Index of the word on the card; back to the first one when the words change. */
  protected readonly index = linkedSignal({ source: this.words, computation: () => 0 });
  /** Words whose picture failed to load: their card is shown without a picture. */
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
        // Only the word's own picture (it needs the session cookie). A word without one gets no
        // picture area, rather than a stock photo that may not match or not load at all.
        image: w.imageUrl ? serverUrl(w.imageUrl) : null,
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
    return r !== null && r.image !== null && !this.brokenImages().has(r.word.lemma);
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
        if (r?.image && !this.brokenImages().has(r.word.lemma)) {
          const img = new Image();
          img.referrerPolicy = 'no-referrer';
          img.crossOrigin = 'use-credentials';
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

  /** ←: the word before, or from the first word the step before. */
  protected previous(): void {
    if (!this.isFirst()) {
      this.index.update((i) => i - 1);
    } else if (this.canBack()) {
      this.back.emit();
    }
  }

  /** "Đã biết": the word stays out of the review queue (pressed again, it goes back in). */
  protected toggleKnown(lemma: string): void {
    this.known.update((s) => {
      const next = new Set(s);
      if (!next.delete(lemma)) {
        next.add(lemma);
      }
      return next;
    });
  }

  /** Tiếp theo: the next word, or the recall from the last word. */
  protected nextOrDone(): void {
    if (!this.isLast()) {
      this.index.update((i) => i + 1);
      return;
    }
    const words = this.words().filter((w) => !this.known().has(w.lemma));
    this.missed.clear();
    this.recall.set(buildRecall(words, newSeed()));
    this.recallIndex.set(0);
    this.picked.set(null);
    if (this.recall().length === 0) {
      this.finish();
      return;
    }
    this.phase.set('recall');
    this.playQuestion();
  }

  /** Back to the words from the recall. */
  protected backToCards(): void {
    this.phase.set('cards');
  }

  /** Reads the word of a listening question aloud. */
  protected playQuestion(): void {
    const q = this.question();
    if (q?.kind === 'listen') {
      this.readAloud.emit(q.word.lemma);
    }
  }

  /** Picks an answer; a wrong one at the first try puts the question again at the end. */
  protected pick(lemma: string): void {
    const q = this.question();
    if (!q || this.picked() !== null) {
      return;
    }
    this.picked.set(lemma);
    if (lemma !== q.word.lemma && !this.missed.has(q.word.lemma)) {
      this.missed.add(q.word.lemma);
      this.recall.update((items) => [...items, { ...q, options: [...q.options].reverse() }]);
    }
  }

  /** Câu tiếp, or the end of the step after the last question. */
  protected nextQuestion(): void {
    if (this.recallIndex() + 1 >= this.recall().length) {
      this.finish();
      return;
    }
    this.recallIndex.update((i) => i + 1);
    this.picked.set(null);
    this.playQuestion();
  }

  private finish(): void {
    this.done.emit({ known: [...this.known()], missed: [...this.missed] });
  }

  protected imageFailed(lemma: string): void {
    this.brokenImages.update((s) => new Set(s).add(lemma));
  }
}
