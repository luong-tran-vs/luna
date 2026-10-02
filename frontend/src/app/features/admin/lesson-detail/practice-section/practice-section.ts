import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { JobStatus, Lesson } from '../../../../core/models/lesson';
import { AdminApiService } from '../../admin-api.service';
import { StatusChip } from '../../status-chip/status-chip';

/**
 * The lesson's practice (F17) as admins see it: status, failure reason, the generated content
 * (read only) and a button to generate it again with one AI request.
 */
@Component({
  selector: 'lu-practice-section',
  imports: [StatusChip],
  templateUrl: './practice-section.html',
  styleUrl: './practice-section.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeSection {
  private readonly api = inject(AdminApiService);

  readonly lesson = input.required<Lesson>();
  /** The lesson returned by the API once regeneration is queued. */
  readonly regenerated = output<Lesson>();

  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  /** Status for the chip; null when there is no practice yet. */
  protected readonly chipStatus = computed<JobStatus | null>(() => {
    const s = this.lesson().practiceStatus;
    return s === 'none' ? null : s;
  });

  /** Why regenerating is not possible now, or null. */
  protected readonly blockedReason = computed(() => {
    const l = this.lesson();
    if (l.practiceStatus === 'running') {
      return 'Đang sinh…';
    }
    return l.annotationStatus === 'done' ? null : 'Cần chú thích xong trước';
  });

  protected async regenerate(): Promise<void> {
    if (this.busy() || this.blockedReason()) {
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    try {
      this.regenerated.emit(await firstValueFrom(this.api.regeneratePractice(this.lesson().id)));
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Không tạo lại được, vui lòng thử lại.');
    } finally {
      this.busy.set(false);
    }
  }
}
