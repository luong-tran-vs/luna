import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import {
  CheckItem,
  CurriculumLesson,
  CurriculumSlot,
  IMPORT_SOURCE,
} from '../../../../core/models/curriculum';

export type RowState = 'empty' | 'draft' | 'published';
type CheckResult = 'ok' | 'warn' | 'bad';

const MARK: Record<CheckResult, string> = { ok: '✓', warn: '!', bad: '✕' };
const RESULT_TEXT: Record<CheckResult, string> = { ok: 'Đạt', warn: 'Cảnh báo', bad: 'Chưa đạt' };
const STATE_TEXT: Record<RowState, string> = {
  empty: 'Trống',
  draft: 'Nháp',
  published: 'Đã đăng',
};

/** The state of a lesson that fills a slot, or of a slot without one. */
export function lessonState(lesson: CurriculumLesson | null): RowState {
  if (!lesson) {
    return 'empty';
  }
  return lesson.draft ? 'draft' : 'published';
}

export function checkResult(item: CheckItem): CheckResult {
  if (item.ok) {
    return 'ok';
  }
  return item.warn ? 'warn' : 'bad';
}

/**
 * One line of a topic of the course: a slot (empty, a draft or published) or another lesson of the
 * topic. The state is a text pill, actions are small buttons on the right (under the title on
 * narrow screens); a draft's checks open under the line.
 */
@Component({
  selector: 'lu-lesson-row',
  imports: [RouterLink],
  templateUrl: './lesson-row.html',
  styleUrl: './lesson-row.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[attr.data-state]': 'state()',
    '[attr.aria-busy]': 'generating()',
  },
})
export class LessonRow {
  /** The slot shown; null for a lesson of the topic outside the course (other). */
  readonly slot = input<CurriculumSlot | null>(null);
  readonly other = input<CurriculumLesson | null>(null);
  /** The topic is in the catalogue, so its slots can be generated. */
  readonly applied = input(true);
  /** Some generation or removal runs on the page: no other can start. */
  readonly busy = input(false);
  readonly generating = input(false);
  readonly publishing = input(false);
  readonly rechecking = input(false);

  readonly generate = output<void>();
  /** Opens the import of a lesson written outside the app into the empty slot (FR-030). */
  readonly importLesson = output<void>();
  readonly publish = output<string>();
  readonly recheck = output<string>();
  readonly remove = output<void>();

  protected readonly checksOpen = signal(false);

  protected readonly lesson = computed(() => this.slot()?.lesson ?? this.other());
  protected readonly state = computed(() => lessonState(this.lesson()));
  protected readonly stateText = computed(() => STATE_TEXT[this.state()]);
  protected readonly id = computed(() => {
    const s = this.slot();
    return s ? `slot-${s.key}` : `other-${this.other()?.id ?? ''}`;
  });
  /** The lesson's own title once filled (as in the lesson list), else the planned one. */
  readonly title = computed(() => this.lesson()?.title || this.slot()?.title || '');
  protected readonly grammarText = computed(
    () =>
      this.slot()
        ?.grammar.map((g) => g.titleVi)
        .join(' + ') ?? '',
  );
  /** "Nhập" for an imported lesson, "AI sinh" for a generated one (FR-030); other sources show nothing. */
  protected readonly sourceText = computed(() => {
    const source = this.lesson()?.source;
    return source === IMPORT_SOURCE || source === 'AI sinh' ? source : '';
  });
  protected readonly idea = computed(() => this.slot()?.idea ?? '');
  protected readonly checks = computed(() => this.slot()?.checks ?? null);
  protected readonly counts = computed(() => {
    const c = { ok: 0, warn: 0, bad: 0 };
    for (const item of this.checks()?.items ?? []) {
      c[checkResult(item)]++;
    }
    return c;
  });
  protected readonly countsText = computed(() => {
    const c = this.counts();
    return `${c.ok} đạt, ${c.warn} cảnh báo, ${c.bad} chưa đạt`;
  });
  protected readonly generateHint = computed(() => {
    if (!this.applied()) {
      return 'Bấm “Áp dụng thứ tự” trước để tạo chủ đề.';
    }
    return this.busy() && !this.generating() ? 'Đang có bài khác được sinh, đợi xong đã.' : null;
  });
  /** Like the roadmap page: learners would get a lesson without words and questions before. */
  protected readonly canPublish = computed(() => this.lesson()?.annotationStatus === 'done');
  protected readonly publishHint = computed(() =>
    this.lesson()?.annotationStatus === 'failed'
      ? 'Chú thích lỗi, mở bài để chạy lại.'
      : 'Đang chú thích, đăng được khi xong.',
  );

  protected toggleChecks(): void {
    this.checksOpen.update((open) => !open);
  }

  protected result(item: CheckItem): CheckResult {
    return checkResult(item);
  }

  protected mark(item: CheckItem): string {
    return MARK[checkResult(item)];
  }

  protected resultText(item: CheckItem): string {
    return RESULT_TEXT[checkResult(item)];
  }
}
