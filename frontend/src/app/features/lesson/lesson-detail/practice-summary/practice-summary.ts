import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../../../shared/components/icon/icon';
import { Summary } from '../practice-logic';

/** Whether the lesson is today's lesson, a finished one or neither (from GET /api/lessons/mine). */
export type LessonStatus = 'today' | 'completed' | 'other';

/** End of the practice: scores (kept only on this page), Làm lại and the way into the lesson. */
@Component({
  selector: 'lu-practice-summary',
  imports: [Icon, RouterLink],
  templateUrl: './practice-summary.html',
  styleUrls: ['../practice.css', './practice-summary.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeSummary {
  readonly summary = input.required<Summary>();
  readonly status = input<LessonStatus>('other');
  readonly lessonId = input.required<string>();

  readonly retry = output<void>();
}
