import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  inject,
  input,
  output,
} from '@angular/core';

let nextId = 0;

/**
 * The Vietnamese translation of one sentence of the reading text, shown after a long press on the
 * sentence (or "T" on one of its words). Positioning is done by the reading page.
 */
@Component({
  selector: 'lu-sentence-translation',
  template: `
    <div class="card" role="dialog" [attr.aria-labelledby]="titleId" tabindex="-1">
      <div class="head">
        <span class="title" [id]="titleId">Bản dịch câu</span>
        <button type="button" class="icon-btn" aria-label="Đóng" (click)="closed.emit()">✕</button>
      </div>
      <p class="en" lang="en">{{ sentence() }}</p>
      @if (translation(); as vi) {
        <p class="vi">{{ vi }}</p>
      } @else {
        <p class="muted">Bài này chưa có bản dịch câu.</p>
      }
    </div>
  `,
  styleUrl: './sentence-translation.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SentenceTranslation {
  /** The English sentence. */
  readonly sentence = input.required<string>();
  /** Its Vietnamese translation; null when the lesson has none yet. */
  readonly translation = input<string | null>(null);
  readonly closed = output<void>();

  protected readonly titleId = `sentence-translation-${nextId++}`;

  constructor() {
    // Move focus into the card when it opens so keyboard and screen-reader users land on it.
    const host = inject<ElementRef<HTMLElement>>(ElementRef);
    afterNextRender(() => host.nativeElement.querySelector<HTMLElement>('.card')?.focus());
  }
}
