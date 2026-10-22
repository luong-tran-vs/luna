import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { PathLesson, PathRoadmap } from '../../../../core/models/study';

/**
 * The way on to the next topic once the main lessons of the goal's topic are done and an extra one
 * is being studied: extra lessons can be skipped.
 */
export interface MoveOn {
  roadmap: PathRoadmap;
  /** Number of the next topic. */
  number: number;
  /** How many main lessons the topic has. */
  mains: number;
  /** The extra lesson being studied, to go on with instead. */
  current: PathLesson;
}

/** Under the goal's road: go on with the extra lesson, or start the next topic. */
@Component({
  selector: 'lu-move-on',
  imports: [RouterLink],
  template: `
    <p class="move-on-text">
      Bài thêm giúp bạn gặp lại từ và ngữ pháp của chủ đề. Bạn có thể học tiếp hoặc sang chủ đề sau.
    </p>
    <div class="move-on-actions">
      <a class="btn" [routerLink]="['/lessons', moveOn().current.id]"
        >Tiếp tục bài thêm<span class="visually-hidden">: {{ moveOn().current.title }}</span></a
      >
      <button type="button" class="btn-primary" [disabled]="pending()" (click)="moveOnStart.emit()">
        Học xong {{ moveOn().mains }} bài chính — sang Chủ đề {{ moveOn().number }}:
        {{ moveOn().roadmap.topicName }}
      </button>
    </div>
  `,
  styleUrl: './move-on.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'soft-card' },
})
export class MoveOnCard {
  readonly moveOn = input.required<MoveOn>();
  readonly pending = input(false);
  readonly moveOnStart = output();
}
