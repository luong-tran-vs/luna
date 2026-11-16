import { DOCUMENT } from '@angular/common';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  inject,
  Injector,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { Icon } from '../../../shared/components/icon/icon';
import { pageHidden } from './visibility-pause';

type Overlay = 'pause' | 'quit' | null;

/**
 * The frame of a game screen (FR-001, FR-006, FR-007): the row of pills on top (projected with the
 * `pills` attribute), an optional time bar, the close button, and the game below. While a round
 * runs, hiding the page pauses it ("Tạm dừng": Tiếp tục / Thoát) and the close button asks
 * "Thoát lượt này?" first. The game reacts to paused / resumed / quit.
 */
@Component({
  selector: 'lu-game-shell',
  imports: [Icon],
  templateUrl: './game-shell.html',
  styleUrl: './game-shell.css',
  host: { '(document:visibilitychange)': 'onVisibility()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GameShell {
  /** The game's name, for the screen reader heading. */
  readonly title = input.required<string>();
  /** Whether a round is being played: only then can it pause, and closing asks first. */
  readonly running = input(false);
  /** The time bar, from 1 (full) to 0; null hides it. */
  readonly progress = input<number | null>(null);
  /** What leaving a round does, shown when asking "Thoát lượt này?". */
  readonly quitNote = input('Lượt này không tính điểm, nhưng từ đã sai vẫn được đưa về lịch ôn.');

  readonly paused = output<void>();
  readonly resumed = output<void>();
  readonly quit = output<void>();

  private readonly document = inject(DOCUMENT);
  private readonly injector = inject(Injector);
  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');
  protected readonly overlay = signal<Overlay>(null);

  protected onVisibility(): void {
    if (pageHidden(this.document) && this.running() && this.overlay() === null) {
      this.open('pause');
    }
  }

  protected askQuit(): void {
    if (!this.running()) {
      this.quit.emit();
      return;
    }
    if (this.overlay() === null) {
      this.open('quit');
    }
  }

  protected resume(): void {
    this.overlay.set(null);
    this.resumed.emit();
  }

  protected leave(): void {
    this.overlay.set(null);
    this.quit.emit();
  }

  private open(kind: Overlay): void {
    this.overlay.set(kind);
    this.paused.emit();
    afterNextRender(() => this.dialog()?.nativeElement.querySelector<HTMLElement>('button')?.focus(), {
      injector: this.injector,
    });
  }
}
