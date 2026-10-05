import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { FlagKind, LessonFlag } from '../../../core/models/lesson';

export const FLAG_LABELS: Record<FlagKind, string> = {
  mismatch: 'Cần xem: AI giải ra đáp án khác',
  ambiguous: 'AI thấy câu mơ hồ',
  wrong: 'AI thấy có thể sai',
  unchecked: 'AI chưa kiểm tra được câu này',
};

/** One flag the AI check raised: a text label (never color alone), the note and a keep-as-is button. */
@Component({
  selector: 'lu-flag-note',
  templateUrl: './flag-note.html',
  styleUrl: './flag-note.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FlagNote {
  readonly flag = input.required<LessonFlag>();
  /** Name of the flagged spot for screen readers, e.g. "câu hỏi 2". */
  readonly subject = input('chỗ này');
  readonly disabled = input(false);
  readonly confirm = output<LessonFlag>();

  protected readonly label = computed(() => (this.flag().confirmed ? 'Đã xem, giữ nguyên' : FLAG_LABELS[this.flag().kind]));
}
