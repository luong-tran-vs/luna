import { CdkConnectedOverlay, ConnectedPosition } from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  input,
  signal,
} from '@angular/core';

import { ReadingLesson, TextTurn } from '../../../../core/models/reading';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { SentenceTranslation } from '../../reading/sentence-translation/sentence-translation';

/** A line of a dialogue lesson with its speaker's number (0, 1, … in order of appearance). */
export type ShownTurn = TextTurn & { who: number };

/** A sentence of a reading text with its index in the lesson. */
interface ShownSentence {
  index: number;
  text: string;
}

/** How long a sentence or line is held to show its translation, and how far the pointer may move meanwhile. */
const LONG_PRESS_MS = 2000;
const PRESS_SLOP = 10;

/** The translation below the sentence, or above it when there is no room. */
const POSITIONS: ConnectedPosition[] = [
  { originX: 'center', originY: 'bottom', overlayX: 'center', overlayY: 'top', offsetY: 8 },
  { originX: 'center', originY: 'top', overlayX: 'center', overlayY: 'bottom', offsetY: -8 },
];

/**
 * The text of the Bài đọc tab. A reading text shows its paragraphs, each sentence read aloud when
 * tapped; a dialogue lesson shows one bubble per line (the second speaker on the right), each
 * speaker in their own browser voice, with Nghe cả đoạn to hear the whole conversation. Holding a
 * sentence or a line for 2 seconds (or "T" on it) shows its Vietnamese translation. The text
 * cannot be copied.
 */
@Component({
  selector: 'lu-reading-text',
  imports: [Icon, CdkConnectedOverlay, SentenceTranslation],
  templateUrl: './reading-text.html',
  styleUrls: ['../practice.css', './reading-text.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    // A long press on a sentence or a line shows its translation.
    '(pointerdown)': 'onPressStart($event)',
    '(pointermove)': 'onPressMove($event)',
    '(pointerup)': 'onPressEnd()',
    '(pointercancel)': 'onPressEnd()',
    '(keydown)': 'onKeydown($event)',
    // The text cannot be copied; the long-press menu would also cover the translation.
    '(contextmenu)': 'onBlocked($event)',
    '(copy)': 'onBlocked($event)',
  },
})
export class ReadingText {
  private readonly speech = inject(SpeechService);
  protected readonly hostElement = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;

  readonly lesson = input.required<ReadingLesson>();
  /** Reading speed. */
  readonly rate = input(1);

  /** False when the browser has no voice: the text is plain. */
  protected readonly canSpeak = this.speech.supported;
  /** The text being read aloud, to highlight its sentence or line. */
  protected readonly speaking = computed(() => this.speech.playing() ?? this.speech.preparing());

  /** The text as paragraphs of sentences. */
  protected readonly paragraphs = computed<ShownSentence[][]>(() => {
    const l = this.lesson();
    const text = new Map(l.sentences.map((s) => [s.index, s.text]));
    const groups = l.paragraphs.length ? l.paragraphs : [l.sentences.map((s) => s.index)];
    return groups.map((p) => p.map((index) => ({ index, text: text.get(index) ?? '' })).filter((s) => s.text !== ''));
  });

  /**
   * A dialogue lesson as its lines; who numbers the speakers in order of appearance and picks
   * their side of the conversation and their voice. Null for a reading text.
   */
  protected readonly turns = computed<ShownTurn[] | null>(() => {
    const turns = this.lesson().turns;
    if (!turns?.length) {
      return null;
    }
    const speakers: string[] = [];
    return turns.map((t) => {
      if (!speakers.includes(t.speaker)) {
        speakers.push(t.speaker);
      }
      return { ...t, who: speakers.indexOf(t.speaker) };
    });
  });

  /** Nghe cả đoạn was pressed and has not finished or been stopped. */
  private readonly playingAll = signal(false);
  /** Also false once something else stopped the reading (another button, leaving the tab). */
  protected readonly allPlaying = computed(() => this.playingAll() && this.speaking() !== null);

  protected readonly positions = POSITIONS;
  /** The sentence or line being pressed (its data-tr), highlighted while the press lasts. */
  protected readonly pressing = signal<string | null>(null);
  /** The translation shown, next to the sentence or line it belongs to. */
  protected readonly translated = signal<{ key: string; anchor: HTMLElement; en: string; vi: string | null } | null>(
    null,
  );
  private pressTimer?: ReturnType<typeof setTimeout>;
  private pressFrom = { x: 0, y: 0 };
  /** A long press just ended: the click it may still fire is not a tap. */
  private afterLongPress = false;

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.pressTimer));
  }

  /** A sentence was tapped: read it, or stop if it is the one being read. */
  protected readSentence(text: string): void {
    if (!this.afterLongPress) {
      this.toggle(text, 0);
    }
  }

  /** A line of a dialogue was tapped: read it in its speaker's voice, or stop if it is being read. */
  protected readTurn(turn: ShownTurn): void {
    if (!this.afterLongPress) {
      this.toggle(turn.text, turn.who);
    }
  }

  private toggle(text: string, voice: number): void {
    this.playingAll.set(false);
    if (this.speaking() === text) {
      this.speech.stop();
    } else {
      this.speech.speak(text, this.rate(), {}, voice);
    }
  }

  /** Nghe cả đoạn: every line in turn, each in its speaker's voice; pressed again, stops. */
  protected toggleAll(): void {
    if (this.allPlaying()) {
      this.playingAll.set(false);
      this.speech.stop();
      return;
    }
    this.playingAll.set(true);
    this.playTurn(0);
  }

  private playTurn(i: number): void {
    const turn = this.turns()?.[i];
    if (!turn || !this.playingAll()) {
      this.playingAll.set(false);
      return;
    }
    const handlers = { ended: () => this.playTurn(i + 1), failed: () => this.playingAll.set(false) };
    this.speech.speak(turn.text, this.rate(), handlers, turn.who);
  }

  // --- a long press on a sentence or a line: its translation ---

  protected onPressStart(event: PointerEvent): void {
    this.cancelPress();
    const item = this.itemOf(event.target);
    if (!item || event.button !== 0) {
      return;
    }
    const key = item.dataset['tr']!;
    this.pressFrom = { x: event.clientX, y: event.clientY };
    this.pressing.set(key);
    this.pressTimer = setTimeout(() => {
      this.pressing.set(null);
      this.afterLongPress = true;
      this.showTranslation(item);
    }, LONG_PRESS_MS);
  }

  /** Moving away (scrolling the page) is not a press. */
  protected onPressMove(event: PointerEvent): void {
    if (
      this.pressing() !== null &&
      Math.hypot(event.clientX - this.pressFrom.x, event.clientY - this.pressFrom.y) > PRESS_SLOP
    ) {
      this.cancelPress();
    }
  }

  protected onPressEnd(): void {
    this.cancelPress();
    if (this.afterLongPress) {
      // The click of this release comes right after it; later clicks are taps again.
      setTimeout(() => (this.afterLongPress = false));
    }
  }

  /** "T" on a sentence or a line: the keyboard's long press. */
  protected onKeydown(event: KeyboardEvent): void {
    const item = this.itemOf(event.target);
    if (item && (event.key === 't' || event.key === 'T')) {
      event.preventDefault();
      this.showTranslation(item);
    }
  }

  protected onBlocked(event: Event): void {
    if (event.target instanceof Node && this.hostElement.querySelector('.text-body')?.contains(event.target)) {
      event.preventDefault();
    }
  }

  protected closeTranslation(): void {
    const t = this.translated();
    this.translated.set(null);
    t?.anchor.focus();
  }

  protected onTranslationOutsideClick(): void {
    if (!this.afterLongPress) {
      this.closeTranslation();
    }
  }

  protected onTranslationKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      this.closeTranslation();
    }
  }

  /** The sentence or line (an element with data-tr: its sentence indexes) at target, if any. */
  private itemOf(target: EventTarget | null): HTMLElement | null {
    const item = target instanceof Element ? target.closest<HTMLElement>('[data-tr]') : null;
    return item && this.hostElement.contains(item) ? item : null;
  }

  private cancelPress(): void {
    clearTimeout(this.pressTimer);
    this.pressing.set(null);
  }

  /** The translation of a sentence, or of a line made of several sentences; null when one is missing. */
  private showTranslation(item: HTMLElement): void {
    const key = item.dataset['tr']!;
    const all = this.lesson().sentenceTranslations ?? [];
    const parts = key.split(',').map((i) => all[Number(i)]);
    const vi = parts.length && parts.every((p) => !!p) ? parts.join(' ') : null;
    this.translated.set({ key, anchor: item, en: item.textContent?.trim() ?? '', vi });
  }
}
