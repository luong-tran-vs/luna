import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  inject,
  input,
  output,
  viewChild,
} from '@angular/core';

import { TalkSession } from '../../../../core/models/talk';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';

/**
 * The whole conversation in a dialog in the middle of the talk screen (added 2026-10-21): the
 * situation, every line with the Vietnamese of Leo's, a mic mark on the lines the learner said
 * aloud. Leo's lines can be heard again. Escape or the backdrop closes it.
 */
@Component({
  selector: 'lu-talk-history',
  imports: [Icon],
  template: `
    <div class="backdrop" (click)="closed.emit()" aria-hidden="true"></div>
    <div
      class="sheet"
      role="dialog"
      aria-modal="true"
      aria-labelledby="history-title"
      (keydown.escape)="closed.emit()"
    >
      <div class="head">
        <h2 id="history-title">Hội thoại</h2>
        <button
          #closeButton
          type="button"
          class="round"
          aria-label="Đóng hội thoại"
          (click)="closed.emit()"
        >
          <lu-icon name="x" />
        </button>
      </div>
      <p class="scenario"><b>Tình huống:</b> {{ session().scenarioVi }}</p>
      <ol class="lines">
        @for (t of session().turns; track $index) {
          <li class="line" [class.mine]="t.role === 'learner'">
            <p lang="en">
              <span class="who">{{ t.role === 'ai' ? 'Leo' : 'Bạn' }}:</span> {{ t.text }}
            </p>
            @if (t.role === 'ai') {
              @if (t.textVi) {
                <p class="vi">{{ t.textVi }}</p>
              }
              @if (canSpeak) {
                <button
                  type="button"
                  class="listen"
                  [attr.aria-label]="'Nghe lại câu ' + ($index + 1)"
                  (click)="listen(t.text)"
                >
                  <lu-icon name="volume" />
                </button>
              }
            } @else if (t.spoken) {
              <p class="vi"><lu-icon name="mic" /> nói</p>
            }
          </li>
        }
      </ol>
    </div>
  `,
  styleUrl: './talk-history.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkHistory {
  private readonly speech = inject(SpeechService);
  private readonly closeButton = viewChild<ElementRef<HTMLButtonElement>>('closeButton');

  readonly session = input.required<TalkSession>();
  readonly closed = output();

  protected readonly canSpeak = this.speech.supported;

  constructor() {
    afterNextRender(() => this.closeButton()?.nativeElement.focus());
  }

  protected listen(text: string): void {
    this.speech.speak(text, 1);
  }
}
