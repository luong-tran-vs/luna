import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';

import { ReadingLesson, TextTurn } from '../../../../core/models/reading';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';

/** A line of a dialogue lesson with its speaker's number (0, 1, … in order of appearance). */
export type ShownTurn = TextTurn & { who: number };

/**
 * The text of the Bài đọc tab. A reading text shows its paragraphs, each sentence read aloud when
 * tapped; a dialogue lesson shows one bubble per line (the second speaker on the right), each
 * speaker in their own browser voice, with Nghe cả đoạn to hear the whole conversation.
 */
@Component({
  selector: 'lu-reading-text',
  imports: [Icon],
  templateUrl: './reading-text.html',
  styleUrls: ['../practice.css', './reading-text.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ReadingText {
  private readonly speech = inject(SpeechService);

  readonly lesson = input.required<ReadingLesson>();
  /** Reading speed. */
  readonly rate = input(1);

  /** False when the browser has no voice: the text is plain. */
  protected readonly canSpeak = this.speech.supported;
  /** The text being read aloud, to highlight its sentence or line. */
  protected readonly speaking = computed(() => this.speech.playing() ?? this.speech.preparing());

  /** The text as paragraphs of sentences. */
  protected readonly paragraphs = computed(() => {
    const l = this.lesson();
    const text = new Map(l.sentences.map((s) => [s.index, s.text]));
    const groups = l.paragraphs.length ? l.paragraphs : [l.sentences.map((s) => s.index)];
    return groups.map((p) => p.map((i) => text.get(i) ?? '').filter((s) => s !== ''));
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

  /** A sentence was tapped: read it, or stop if it is the one being read. */
  protected readSentence(text: string): void {
    this.toggle(text, 0);
  }

  /** A line of a dialogue was tapped: read it in its speaker's voice, or stop if it is being read. */
  protected readTurn(turn: ShownTurn): void {
    this.toggle(turn.text, turn.who);
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
}
