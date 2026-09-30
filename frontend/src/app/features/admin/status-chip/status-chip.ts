import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { JobStatus } from '../../../core/models/lesson';

const TEXT: Record<JobStatus, string> = { running: 'Đang chạy', done: 'Xong', failed: 'Lỗi' };

/** Shows one background-work status with text and an icon, never color alone. */
@Component({
  selector: 'lu-status-chip',
  templateUrl: './status-chip.html',
  styleUrl: './status-chip.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatusChip {
  readonly status = input.required<JobStatus>();
  readonly label = input.required<string>();

  protected readonly text = computed(() => `${this.label()}: ${TEXT[this.status()]}`);
}
