import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Outcome, ResultView, ToeicPart } from '../../../core/models/toeic';
import { AuthService } from '../../../core/services/auth.service';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { errorBody } from '../../admin/json-import/json-import';
import {
  accuracyOf,
  durationLabel,
  outcomeLabel,
  outcomeMark,
  PART_NAMES,
  percentLabel,
} from '../shared/score.logic';

/**
 * The result page (FR-024…FR-027): the counts, accuracy and time; the estimated scores when the
 * sitting has whole sections; the "question types" table of each part with its numbers coloured
 * and marked ✓ ✗ – (each opens the answer detail); and "right answer : your answer".
 */
@Component({
  selector: 'lu-toeic-result',
  imports: [RouterLink],
  templateUrl: './result.html',
  styleUrls: ['../shared/toeic-page.css', './result.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Result implements OnInit {
  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);
  private readonly isGuest = inject(AuthService).isGuest;
  protected readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';

  protected readonly data = signal<ResultView | null>(null);
  protected readonly error = signal('');
  protected readonly partTab = signal<ToeicPart | null>(null);
  protected readonly retrying = signal(false);
  protected readonly retryError = signal('');
  protected readonly names = PART_NAMES;

  protected readonly tagPart = computed(() => {
    const d = this.data();
    const p = this.partTab();
    return d?.byPart.find((x) => x.part === p) ?? d?.byPart[0] ?? null;
  });
  protected readonly toRetry = computed(() => {
    const d = this.data();
    return d ? d.wrong + d.skipped : 0;
  });
  protected readonly canRetry = computed(() => !this.isGuest() && this.toRetry() > 0);

  ngOnInit(): void {
    this.api.result(this.id).subscribe({
      next: (d) => this.data.set(d),
      error: (err: unknown) => {
        if (err instanceof ApiError && err.status === 409) {
          void this.router.navigate(['/toeic/attempts', this.id], { replaceUrl: true });
          return;
        }
        this.error.set(
          err instanceof ApiError && err.status === 404
            ? 'Không tìm thấy bài làm này.'
            : 'Không tải được kết quả. Kiểm tra mạng rồi thử lại.',
        );
      },
    });
  }

  protected duration(ms: number): string {
    return durationLabel(ms);
  }

  protected percent(v: number): string {
    return percentLabel(v);
  }

  protected rowAccuracy(right: number, wrong: number, skipped: number): string {
    return percentLabel(accuracyOf(right, right + wrong + skipped));
  }

  protected mark(o: Outcome): string {
    return outcomeMark(o);
  }

  protected label(o: Outcome): string {
    return outcomeLabel(o);
  }

  protected back(): string[] {
    const d = this.data();
    return d?.source.kind === 'test' && d.source.testKey ? ['/toeic/tests', d.source.testKey] : ['/toeic'];
  }

  /** Opens a sitting of only the wrong and skipped questions of this one (FR-032). */
  protected retry(): void {
    this.retrying.set(true);
    this.retryError.set('');
    this.api.start({ source: { kind: 'retry', fromAttempt: this.id } }).subscribe({
      next: (v) => void this.router.navigate(['/toeic/attempts', v.id]),
      error: (err: unknown) => {
        this.retrying.set(false);
        this.retryError.set(errorBody(err).message ?? 'Chưa mở được lượt làm lại, thử lại.');
      },
    });
  }
}
