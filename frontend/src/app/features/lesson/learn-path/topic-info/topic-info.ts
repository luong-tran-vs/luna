import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import { PathCourse } from '../../../../core/models/study';

/**
 * Over the road of a course topic (design/de-xuat-giao-trinh-a1.png, mục 4): its number and name
 * for the first topic of the part (the next ones get theirs on the line between two roadmaps), the
 * grammar it teaches as chips and what the learner can do after it.
 */
@Component({
  selector: 'lu-topic-info',
  template: `
    @if (showTitle()) {
      <h3 class="topic-title">Chủ đề {{ course().number }} • {{ topicName() }}</h3>
    }
    @if (course().grammarVi.length) {
      <p class="grammar">
        <span class="grammar-label">Ngữ pháp:</span>
        @for (g of course().grammarVi; track $index) {
          <span class="grammar-chip">{{ g }}</span>
        }
      </p>
    }
    @if (showCanDo() && course().canDoVi) {
      <p class="can-do">Sau chủ đề này bạn có thể {{ course().canDoVi }}</p>
    }
  `,
  styleUrl: './topic-info.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TopicInfo {
  readonly course = input.required<PathCourse>();
  readonly topicName = input.required<string>();
  readonly showTitle = input(false);
  /** Off for the topic named in the banner, which already says what it leads to. */
  readonly showCanDo = input(true);
}
