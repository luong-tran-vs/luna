import { ChangeDetectionStrategy, Component, computed, effect, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router } from '@angular/router';
import { map } from 'rxjs';

import { DailyChallenge } from './daily-challenge/daily-challenge';
import { GrammarSpot } from './grammar-spot/grammar-spot';
import { LetterDrop } from './letter-drop/letter-drop';
import { MemoryMatch } from './memory-match/memory-match';
import { PictureQuiz } from './picture-quiz/picture-quiz';
import { VerbDuel } from './verb-duel/verb-duel';

/** The games that can be played, in the order they were released (P1 → P6). */
export const PLAYABLE = ['picture', 'verbs', 'letters', 'memory', 'grammar', 'daily'] as const;
export type PlayableGame = (typeof PLAYABLE)[number];

/** Whether a route's game is one that can be played. */
export function isPlayable(id: string | null | undefined): id is PlayableGame {
  return (PLAYABLE as readonly string[]).includes(id ?? '');
}

/**
 * The page of /games/:game: the screen of that game, full screen (isFocusPage); a game not
 * released goes back to the list.
 */
@Component({
  selector: 'lu-game-host',
  imports: [PictureQuiz, VerbDuel, LetterDrop, MemoryMatch, GrammarSpot, DailyChallenge],
  template: `
    @switch (game()) {
      @case ('picture') {
        <lu-picture-quiz />
      }
      @case ('verbs') {
        <lu-verb-duel />
      }
      @case ('letters') {
        <lu-letter-drop />
      }
      @case ('memory') {
        <lu-memory-match />
      }
      @case ('grammar') {
        <lu-grammar-spot />
      }
      @case ('daily') {
        <lu-daily-challenge />
      }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GameHost {
  private readonly router = inject(Router);
  private readonly param = toSignal(
    inject(ActivatedRoute).paramMap.pipe(map((p) => p.get('game'))),
    { initialValue: null },
  );

  protected readonly game = computed(() => {
    const id = this.param();
    return isPlayable(id) ? id : null;
  });

  constructor() {
    effect(() => {
      const id = this.param();
      if (id !== null && !isPlayable(id)) {
        void this.router.navigateByUrl('/games');
      }
    });
  }
}
