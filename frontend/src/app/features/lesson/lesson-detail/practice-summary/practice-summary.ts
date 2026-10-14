import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { LessonStatus } from '../../../../core/models/study';
import { Icon } from '../../../../shared/components/icon/icon';
import type { PracticeItem } from '../lesson-detail';
import { PracticeList } from '../practice-list/practice-list';

/**
 * End of the practice of a lesson not being studied: what was done in each step (a step skipped
 * says "Chưa làm" with a way into it), Làm lại, and for a finished lesson the ways back into its steps.
 */
@Component({
  selector: 'lu-practice-summary',
  imports: [Icon, PracticeList, RouterLink],
  templateUrl: './practice-summary.html',
  styleUrls: ['../practice.css', './practice-summary.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeSummary {
  readonly items = input.required<PracticeItem[]>();
  readonly status = input<LessonStatus>('other');
  readonly lessonId = input.required<string>();

  readonly go = output<PracticeItem['key']>();
  readonly retry = output<void>();
}
