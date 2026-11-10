import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  input,
  output,
  viewChild,
} from '@angular/core';

/**
 * Asked when the learner comes back to a topic whose shown lessons are all done and lessons are
 * still held back (added 2026-11-09): study again from lesson 1, or open one new lesson. Escape and
 * "Để sau" just close it. Focus moves to "Học lại" when it opens; Tab stays inside.
 */
@Component({
  selector: 'lu-more-lessons',
  template: `
    @if (open()) {
      <div class="backdrop">
        <div
          class="dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="more-title"
          aria-describedby="more-message"
          tabindex="-1"
          (keydown)="trapTab($event)"
        >
          <h2 class="title" id="more-title">Học lại hay học bài mới?</h2>
          <p class="message" id="more-message">
            Bạn đã học xong {{ done() }} bài của chủ đề {{ topicName() }}. Chủ đề còn
            {{ remaining() }} bài mới — mỗi lần chọn "Học bài mới" sẽ mở thêm 1 bài.
          </p>
          <div class="actions">
            <button
              #first
              type="button"
              class="btn"
              [disabled]="pending()"
              (click)="relearn.emit()"
            >
              Học lại từ bài 1
            </button>
            <button type="button" class="btn-primary" [disabled]="pending()" (click)="more.emit()">
              Học bài mới
            </button>
          </div>
          <button #last type="button" class="later" (click)="closed.emit()">Để sau</button>
        </div>
      </div>
    }
  `,
  styleUrl: './more-lessons.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class MoreLessonsDialog {
  readonly open = input(false);
  readonly topicName = input.required<string>();
  /** Lessons of the topic done (all those shown). */
  readonly done = input(0);
  /** Lessons still held back. */
  readonly remaining = input(0);
  readonly pending = input(false);

  readonly relearn = output();
  readonly more = output();
  readonly closed = output();

  private readonly firstButton = viewChild<ElementRef<HTMLButtonElement>>('first');
  private readonly lastButton = viewChild<ElementRef<HTMLButtonElement>>('last');

  constructor() {
    afterRenderEffect(() => {
      if (this.open()) {
        this.firstButton()?.nativeElement.focus();
      }
    });
  }

  protected onEscape(): void {
    if (this.open()) {
      this.closed.emit();
    }
  }

  protected trapTab(event: KeyboardEvent): void {
    if (event.key !== 'Tab') {
      return;
    }
    const first = this.firstButton()?.nativeElement;
    const last = this.lastButton()?.nativeElement;
    const active = document.activeElement;
    if (event.shiftKey && active === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && active === last) {
      event.preventDefault();
      first?.focus();
    }
  }
}
