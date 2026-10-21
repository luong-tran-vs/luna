import { ChangeDetectionStrategy, Component, computed, inject, input, output } from '@angular/core';
import { NgOptimizedImage } from '@angular/common';
import { RouterLink } from '@angular/router';

import { TalkSession } from '../../../../core/models/talk';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';

/**
 * The review of an ended conversation (added 2026-10-21, from design/de-xuat-tro-chuyen-voi-ai.png):
 * Leo's overall comment, three counts from the session, the lines worth saying again (the better
 * line read aloud on demand), the well said lines and the topic's words as chips.
 */
@Component({
  selector: 'lu-talk-review',
  imports: [Icon, NgOptimizedImage, RouterLink],
  templateUrl: './talk-review.html',
  styleUrl: './talk-review.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkReview {
  private readonly speech = inject(SpeechService);

  readonly session = input.required<TalkSession>();
  /** While a new conversation about the topic is being prepared. */
  readonly restarting = input(false);
  readonly again = output();

  protected readonly canSpeak = this.speech.supported;
  protected readonly said = computed(
    () => this.session().turns.filter((t) => t.role === 'learner').length,
  );

  protected listen(text: string): void {
    this.speech.speak(text, 1);
  }
}
