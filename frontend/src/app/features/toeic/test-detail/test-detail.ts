import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { TestDetail as Detail, ToeicPart } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { errorBody } from '../../admin/json-import/json-import';
import {
  durationLabel,
  kindLabel,
  PART_NAMES,
  shortDate,
  suggestedMinutes,
} from '../shared/score.logic';

type Tab = 'practice' | 'full' | 'results';

/** The other time limits offered besides the suggested one (minutes; 0 = no limit). */
const LIMITS = [10, 20, 30, 45, 60, 75, 0];

/**
 * The detail of a test (FR-004, FR-005): "Luyện tập" ticks parts (each with its questions and
 * question types) and a time limit (a suggestion from the questions ticked, others, or none);
 * "Làm full test" comes with batch 3; "Kết quả" lists every sitting, retakes labelled.
 */
@Component({
  selector: 'lu-toeic-test-detail',
  imports: [RouterLink, ConfirmDialog],
  templateUrl: './test-detail.html',
  styleUrls: ['../shared/toeic-page.css', './test-detail.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TestDetail implements OnInit {
  protected readonly key = signal(inject(ActivatedRoute).snapshot.paramMap.get('key') ?? '');

  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);

  protected readonly data = signal<Detail | null>(null);
  protected readonly error = signal('');
  protected readonly tab = signal<Tab>('practice');
  protected readonly picked = signal<ToeicPart[]>([]);
  /** null = the suggested time. */
  protected readonly limit = signal<number | null>(null);
  protected readonly starting = signal(false);
  protected readonly startError = signal('');
  protected readonly busyId = signal<string | null>(null);

  protected readonly allParts: ToeicPart[] = [1, 2, 3, 4, 5, 6, 7];
  protected readonly names = PART_NAMES;
  protected readonly limits = LIMITS;

  protected readonly partsById = computed(() => {
    const out = new Map<ToeicPart, { questions: number; tags: string[] }>();
    for (const p of this.data()?.parts ?? []) {
      out.set(p.part, p);
    }
    return out;
  });
  protected readonly pickedQuestions = computed(() =>
    this.picked().reduce((n, p) => n + (this.partsById().get(p)?.questions ?? 0), 0),
  );
  protected readonly suggested = computed(() => suggestedMinutes(this.pickedQuestions()));
  protected readonly minutes = computed(() => this.limit() ?? this.suggested());

  ngOnInit(): void {
    this.api.test(this.key()).subscribe({
      next: (d) => {
        this.data.set(d);
        this.picked.set(d.parts.filter((p) => p.part >= 5).map((p) => p.part));
      },
      error: (err: unknown) =>
        this.error.set(
          err instanceof ApiError && err.status === 404
            ? 'Không tìm thấy đề này.'
            : 'Không tải được đề. Kiểm tra mạng rồi thử lại.',
        ),
    });
  }

  protected kind(): string {
    const d = this.data();
    return d ? kindLabel(d.kind) : '';
  }

  protected toggle(p: ToeicPart): void {
    this.picked.update((list) =>
      list.includes(p) ? list.filter((x) => x !== p) : [...list, p].sort((a, b) => a - b),
    );
  }

  protected setLimit(event: Event): void {
    const v = (event.target as HTMLSelectElement).value;
    this.limit.set(v === 'suggested' ? null : Number(v));
  }

  protected start(): void {
    if (this.picked().length === 0 || this.starting()) {
      return;
    }
    this.starting.set(true);
    this.startError.set('');
    this.api
      .start({
        source: { kind: 'test', testKey: this.key() },
        mode: 'practice',
        parts: this.picked(),
        timeLimitMin: this.minutes(),
      })
      .subscribe({
        next: (v) => void this.router.navigate(['/toeic/attempts', v.id]),
        error: (err: unknown) => {
          this.starting.set(false);
          const body = errorBody(err).body;
          if (body?.['error'] === 'attempt_in_progress' && typeof body['attemptId'] === 'string') {
            this.busyId.set(body['attemptId']);
            return;
          }
          this.startError.set(errorBody(err).message ?? 'Chưa bắt đầu được, thử lại.');
        },
      });
  }

  protected resume(): void {
    const id = this.busyId() ?? this.data()?.inProgress;
    if (id) {
      void this.router.navigate(['/toeic/attempts', id]);
    }
  }

  protected duration(ms: number): string {
    return durationLabel(ms);
  }

  protected date(at: string): string {
    return shortDate(at);
  }

  protected partsLabel(parts: ToeicPart[]): string {
    return parts.length === 7 ? 'Đủ 7 Part' : parts.map((p) => `Part ${p}`).join(', ');
  }
}
