import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  input,
  output,
  viewChild,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../../shared/components/icon/icon';
import { formatScore } from './game-score.logic';

/** A missed item of the round: what it was and its answer or meaning. */
export interface MissedItem {
  text: string;
  detail: string;
}

/**
 * The result screen every game ends on (FR-002): the score, right and wrong answers, the best ("Kỷ
 * lục mới" when beaten), what was missed with its answer, "Chơi lại" and "Về danh sách trò".
 */
@Component({
  selector: 'lu-game-result',
  imports: [RouterLink, Icon],
  templateUrl: './game-result.html',
  styleUrl: './game-result.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GameResult {
  /** e.g. "Hết giờ!", "Hết mạng!", "Xong!". */
  readonly heading = input('Xong lượt!');
  readonly score = input.required<number>();
  /** How the score is named ("điểm", "động từ liền"). */
  readonly unit = input('điểm');
  readonly right = input<number | null>(null);
  readonly wrong = input<number | null>(null);
  /** Extra lines (e.g. "18 lượt lật · 0:42"). */
  readonly extra = input('');
  /** The best after the round; null while it is being sent. */
  readonly best = input<number | null>(null);
  readonly newBest = input(false);
  readonly missed = input<readonly MissedItem[]>([]);
  /** The result could not be sent: it is kept on this device. */
  readonly offline = input(false);

  readonly again = output<void>();

  private readonly title = viewChild<ElementRef<HTMLElement>>('title');
  protected readonly format = formatScore;

  constructor() {
    afterNextRender(() => this.title()?.nativeElement.focus());
  }
}
