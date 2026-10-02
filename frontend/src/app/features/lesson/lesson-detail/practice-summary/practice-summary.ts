import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { LessonStatus } from '../../../../core/models/study';
import { Icon } from '../../../../shared/components/icon/icon';
import { Summary } from '../practice-logic';

/**
 * End of the practice of a lesson not being studied: scores (kept only on this page), Làm lại, and
 * for a finished lesson the ways back into its steps.
 */
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
