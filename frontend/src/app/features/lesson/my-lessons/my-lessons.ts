import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { MyLessons as MyLessonsData } from '../../../core/models/study';
import { StudyApiService } from '../../../core/services/study-api.service';

/** The learner's lessons (L): today's, the ones studied (reopen freely), the locked upcoming ones. */
@Component({
  selector: 'lu-my-lessons',
  imports: [DatePipe, RouterLink],
  templateUrl: './my-lessons.html',
  styleUrl: './my-lessons.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MyLessons {
  protected readonly data = signal<MyLessonsData | null>(null);
  protected readonly error = signal(false);

  constructor() {
    inject(StudyApiService)
      .myLessons()
      .subscribe({ next: (d) => this.data.set(d), error: () => this.error.set(true) });
  }
}
