import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, EMPTY, interval, startWith, Subject, switchMap, merge, tap } from 'rxjs';

import { AiCall, AiTotals, AiUsage as Usage } from '../../../core/models/ai-usage';
import { AdminApiService } from '../admin-api.service';
import { FEATURE_COSTS, opLabel } from './ai-usage-catalog';

/** How often the page asks for fresh numbers while open. */
const REFRESH_MS = 30_000;

const numberFormat = new Intl.NumberFormat('vi-VN');
const timeFormat = new Intl.DateTimeFormat('vi-VN', { hour: '2-digit', minute: '2-digit' });
const dateTimeFormat = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
});

const OUTCOME_LABELS: Record<AiCall['outcome'], string> = {
  ok: 'Thành công',
  quota: 'Hết lượt',
  error: 'Lỗi',
};

/** One bar of the requests-per-minute chart. */
interface Bar {
  label: string;
  requests: number;
  tokens: number;
  /** Height in percent of the busiest minute. */
  height: number;
}

/**
 * The AI usage page: requests and tokens sent to Gemini, recorded by the server from each answer.
 * The free tier limits requests and tokens per minute and requests per day, per model, so the
 * page shows the last minute, the last hour minute by minute and today (Google's day, from
 * midnight Pacific time), then the last 7 days per feature, the last requests and what each
 * feature costs.
 */
@Component({
  selector: 'lu-ai-usage',
  templateUrl: './ai-usage.html',
  styleUrl: './ai-usage.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AiUsage {
  private readonly api = inject(AdminApiService);
  private readonly refreshes = new Subject<void>();

  protected readonly usage = signal<Usage | null>(null);
  protected readonly loadError = signal(false);
  protected readonly loading = signal(false);
  protected readonly featureCosts = FEATURE_COSTS;
  protected readonly opLabel = opLabel;
  /** The bar under the pointer or keyboard focus, for the tooltip. */
  protected readonly active = signal<number | null>(null);

  protected readonly updatedAt = computed(() => {
    const u = this.usage();
    return u ? dateTimeFormat.format(new Date(u.now)) : '';
  });

  /** When Google's day began, in the viewer's time. */
  protected readonly dayStart = computed(() => {
    const u = this.usage();
    return u ? timeFormat.format(new Date(u.dayStart)) : '';
  });

  protected readonly lastMinute = computed(() => sum(this.usage()?.lastMinute ?? []));
  protected readonly today = computed(() => sum(this.usage()?.today ?? []));
  protected readonly hourTotal = computed(() => sum(this.usage()?.minutes ?? []));

  protected readonly bars = computed<Bar[]>(() => {
    const minutes = this.usage()?.minutes ?? [];
    const top = Math.max(1, ...minutes.map((m) => m.requests));
    return minutes.map((m) => ({
      label: timeFormat.format(new Date(m.start)),
      requests: m.requests,
      tokens: m.totalTokens,
      height: (m.requests / top) * 100,
    }));
  });
  protected readonly barTop = computed(() => Math.max(1, ...this.bars().map((b) => b.requests)));
  protected readonly activeBar = computed(() => {
    const i = this.active();
    return i === null ? null : (this.bars()[i] ?? null);
  });

  constructor() {
    merge(interval(REFRESH_MS), this.refreshes)
      .pipe(
        startWith(null),
        tap(() => this.loading.set(true)),
        switchMap(() =>
          this.api.aiUsage().pipe(
            catchError(() => {
              this.loadError.set(true);
              this.loading.set(false);
              return EMPTY;
            }),
          ),
        ),
        takeUntilDestroyed(),
      )
      .subscribe((u) => {
        this.usage.set(u);
        this.loadError.set(false);
        this.loading.set(false);
      });
  }

  protected refresh(): void {
    this.refreshes.next();
  }

  protected n(value: number): string {
    return numberFormat.format(value);
  }

  protected time(at: string): string {
    return dateTimeFormat.format(new Date(at));
  }

  protected outcome(c: AiCall): string {
    return c.outcome === 'error' && c.status ? `${OUTCOME_LABELS.error} ${c.status}` : OUTCOME_LABELS[c.outcome];
  }

  /** Average tokens per answered request. */
  protected perRequest(t: AiTotals): string {
    const answered = t.requests - t.errors - t.quota;
    return answered > 0 ? this.n(Math.round(t.totalTokens / answered)) : '–';
  }

  protected seconds(ms: number): string {
    return (ms / 1000).toLocaleString('vi-VN', { maximumFractionDigits: 1 }) + ' s';
  }
}

function sum(list: readonly AiTotals[]): AiTotals {
  const out: AiTotals = { requests: 0, errors: 0, quota: 0, promptTokens: 0, outputTokens: 0, totalTokens: 0 };
  for (const t of list) {
    out.requests += t.requests;
    out.errors += t.errors;
    out.quota += t.quota;
    out.promptTokens += t.promptTokens;
    out.outputTokens += t.outputTokens;
    out.totalTokens += t.totalTokens;
  }
  return out;
}
