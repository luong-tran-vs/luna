import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { FlagKind, Lesson, LessonFlag } from '../../../core/models/lesson';
import { FixHelper } from '../fix-helper/fix-helper';

export const FLAG_LABELS: Record<FlagKind, string> = {
  mismatch: 'Cần xem: AI giải ra đáp án khác',
  ambiguous: 'AI thấy câu mơ hồ',
  wrong: 'AI thấy có thể sai',
  unchecked: 'AI chưa kiểm tra được câu này',
};

/** The id of the note of a flag, so the check summary can jump to it. */
export function flagNoteId(flag: Pick<LessonFlag, 'area' | 'index'>): string {
  return `flag-${flag.area}-${flag.index}`;
}

/**
 * One flag the AI check raised: a text label (never color alone), the note, a keep-as-is button
 * and, with a lessonId, "AI gợi ý sửa" (FixHelper).
 */
@Component({
  selector: 'lu-flag-note',
  imports: [FixHelper],
  templateUrl: './flag-note.html',
  styleUrl: './flag-note.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Focusable by script only: the check summary moves focus here when it jumps to the flag.
  host: { '[id]': 'hostId()', tabindex: '-1' },
})
export class FlagNote {
  readonly flag = input.required<LessonFlag>();
  /** Name of the flagged spot for screen readers, e.g. "câu hỏi 2". */
  readonly subject = input('chỗ này');
  readonly disabled = input(false);
  /** The lesson of the flag; '' hides "AI gợi ý sửa". */
  readonly lessonId = input('');
  readonly confirm = output<LessonFlag>();
  /** The lesson after an AI-suggested correction was applied. */
  readonly applied = output<Lesson>();

  protected readonly hostId = computed(() => flagNoteId(this.flag()));
  protected readonly label = computed(() => (this.flag().confirmed ? 'Đã xem, giữ nguyên' : FLAG_LABELS[this.flag().kind]));
}
