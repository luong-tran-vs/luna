import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../../shared/components/icon/icon';
import { LessonItem } from './lesson-items';

/** The five marks of the score of a lesson's latest review round. */
const MASTERY_MARKS = [1, 2, 3, 4, 5];

/**
 * Lessons learned as rows: the title, when last seen, the latest round's score, the state of its
 * words and "Ôn lại" (design/tat-ca-de-xuat.png (mục on-tap-bo-cuc)). Used by the review tab and "Bài đã học".
 */
@Component({
  selector: 'lu-lesson-list',
  imports: [Icon, RouterLink],
  templateUrl: './lesson-list.html',
  styleUrl: './lesson-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonList {
  readonly items = input.required<readonly LessonItem[]>();
  protected readonly marks = MASTERY_MARKS;
}
