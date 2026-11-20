import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { TestAdmin, ToeicStatus } from '../../../../core/models/toeic';
import { ToeicApiService } from '../../../../core/services/toeic-api.service';
import { kindLabel, PART_NAMES } from '../../../toeic/shared/score.logic';
import { errorBody, fieldLines, parseJson } from '../../json-import/json-import';
import { JsonSource } from '../../json-import/json-source';
import { GroupPreview } from '../group-preview';
import { PhotoPanel } from './photo-panel';
import { reasonsOf, STATUS_LABELS } from '../toeic-admin.logic';

/**
 * A test (FR-043): what it has and still needs in each part, the draft groups, the Part 1 photos
 * missing, graphics, texts and listening minutes; "Đăng đề" only when it is ready (the reason shown
 * otherwise); "Ẩn đề"; its groups by part; and, while not published, editing its JSON.
 */
@Component({
  selector: 'lu-toeic-test-page',
  imports: [RouterLink, JsonSource, GroupPreview, PhotoPanel],
  templateUrl: './test-page.html',
  styleUrls: ['../toeic-admin.css', './test-page.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TestPage {
  private readonly api = inject(ToeicApiService);
  private readonly key = inject(ActivatedRoute).snapshot.paramMap.get('key') ?? '';

  protected readonly test = signal<TestAdmin | null>(null);
  protected readonly error = signal('');
  protected readonly busy = signal(false);
  protected readonly preview = signal<string | null>(null);
  protected readonly editing = signal(false);
  protected readonly text = signal('');
  protected readonly editError = signal('');
  protected readonly editFields = signal<{ key: string; text: string }[]>([]);
  protected readonly names = PART_NAMES;

  protected readonly reasons = computed(() => {
    const t = this.test();
    return t ? reasonsOf(t.check) : [];
  });

  /** The Part 1 groups of the test, in order. */
  protected readonly part1Keys = computed(() =>
    (this.test()?.groupRows ?? []).filter((g) => g.part === 1).map((g) => g.key),
  );

  constructor() {
    this.load();
  }

  /** After a photo was saved: the check counts it. */
  protected reload(): void {
    this.load();
  }

  private load(): void {
    this.api.testAdmin(this.key).subscribe({
      next: (t) => this.test.set(t),
      error: () => this.error.set('Không tải được đề này.'),
    });
  }

  protected kind(): string {
    const t = this.test();
    return t ? kindLabel(t.kind) : '';
  }

  protected statusLabel(s: ToeicStatus): string {
    return STATUS_LABELS[s];
  }

  protected setStatus(status: ToeicStatus): void {
    this.busy.set(true);
    this.error.set('');
    this.api.setTestStatus(this.key, status).subscribe({
      next: (t) => {
        this.busy.set(false);
        this.test.set(t);
      },
      error: (err: unknown) => {
        this.busy.set(false);
        this.error.set(errorBody(err).message ?? 'Chưa đổi được trạng thái đề.');
      },
    });
  }

  protected startEdit(): void {
    const t = this.test();
    if (!t) {
      return;
    }
    this.text.set(
      JSON.stringify(
        { key: t.key, title: t.title, kind: t.kind, order: t.order, groups: t.groups },
        null,
        2,
      ),
    );
    this.editing.set(true);
  }

  protected saveEdit(): void {
    this.editError.set('');
    this.editFields.set([]);
    const parsed = parseJson(this.text());
    if ('error' in parsed) {
      this.editError.set(parsed.error);
      return;
    }
    this.api.updateTest(this.key, parsed.value).subscribe({
      next: (t) => {
        this.test.set(t);
        this.editing.set(false);
      },
      error: (err: unknown) => {
        const e = errorBody(err);
        this.editError.set(e.message ?? 'Chưa lưu được.');
        if (e.fields) {
          this.editFields.set(fieldLines(e.fields));
        }
      },
    });
  }
}
