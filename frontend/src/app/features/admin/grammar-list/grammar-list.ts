import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { GrammarPoint } from '../../../core/models/grammar';
import {
  GrammarLessonAdmin,
  GrammarReportGroup,
  reasonsText,
  GrammarLessonSummary,
  grammarStatusText,
} from '../../../core/models/grammar-admin';
import { Level, LEVELS } from '../../../core/models/lesson';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { Loading } from '../../../shared/components/loading/loading';
import { AdminApiService } from '../admin-api.service';
import { grammarFailure } from '../grammar-errors';

const GENERATE_FAILED = 'Sinh bài thất bại, vui lòng thử lại.';
const ACTION_FAILED = 'Không thực hiện được, vui lòng thử lại.';

/** F20: the curriculum by level with the state of each point's grammar lesson and its actions. */
@Component({
  selector: 'lu-grammar-list',
  imports: [Loading, ConfirmDialog, RouterLink],
  templateUrl: './grammar-list.html',
  styleUrl: './grammar-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarList {
  private readonly api = inject(AdminApiService);

  protected readonly levels = LEVELS;
  protected readonly statusText = grammarStatusText;
  protected readonly level = signal<Level>('A1');
  protected readonly points = signal<GrammarPoint[] | null>(null);
  protected readonly lessons = signal<Record<string, GrammarLessonSummary>>({});
  protected readonly loadError = signal<string | null>(null);
  /** Point whose request is running. */
  protected readonly busy = signal<string | null>(null);
  protected readonly busyAction = signal<'generate' | 'publish' | 'unpublish' | null>(null);
  protected readonly rowError = signal<{ pointId: string; message: string } | null>(null);
  protected readonly note = signal('');
  /** Point waiting for the admin to confirm replacing its lesson. */
  protected readonly confirming = signal<GrammarPoint | null>(null);

  protected readonly published = computed(() => {
    const lessons = this.lessons();
    return (this.points() ?? []).filter((p) => lessons[p.id]?.status === 'published').length;
  });

  /** F21: open learner reports; a failed load only hides the section. */
  protected readonly reports = signal<GrammarReportGroup[]>([]);
  protected readonly reasons = reasonsText;
  private readonly reportCounts = computed(() => {
    const counts: Record<string, number> = {};
    for (const r of this.reports()) {
      counts[r.pointId] = (counts[r.pointId] ?? 0) + r.count;
    }
    return counts;
  });

  private loadSeq = 0;

  constructor() {
    this.load();
    this.loadReports();
  }

  private loadReports(): void {
    this.api.grammarReports().subscribe({
      next: (list) => this.reports.set(list),
      error: () => this.reports.set([]),
    });
  }

  protected reportCount(p: GrammarPoint): number {
    return this.reportCounts()[p.id] ?? 0;
  }

  protected reportTitle(pointId: string): string {
    return this.points()?.find((p) => p.id === pointId)?.titleVi ?? pointId;
  }

  private load(): void {
    const seq = ++this.loadSeq;
    this.points.set(null);
    this.loadError.set(null);
    this.rowError.set(null);
    this.note.set('');
    const level = this.level();
    let pointsDone = false;
    let lessonsDone = false;
    let points: GrammarPoint[] = [];
    let lessons: GrammarLessonSummary[] = [];
    const finish = () => {
      if (seq !== this.loadSeq || !pointsDone || !lessonsDone) {
        return;
      }
      this.lessons.set(Object.fromEntries(lessons.map((l) => [l.pointId, l])));
      this.points.set(points);
    };
    const fail = () => {
      if (seq === this.loadSeq) {
        this.loadError.set('Không tải được danh sách điểm ngữ pháp.');
      }
    };
    this.api.grammarPoints(level).subscribe({
      next: (list) => {
        points = list;
        pointsDone = true;
        finish();
      },
      error: fail,
    });
    this.api.grammarLessons().subscribe({
      next: (list) => {
        lessons = list;
        lessonsDone = true;
        finish();
      },
      error: fail,
    });
  }

  protected retry(): void {
    this.load();
  }

  protected selectLevel(level: Level): void {
    if (level === this.level()) {
      return;
    }
    this.level.set(level);
    this.load();
  }

  protected lessonOf(p: GrammarPoint): GrammarLessonSummary | null {
    return this.lessons()[p.id] ?? null;
  }

  protected errorOf(p: GrammarPoint): string | null {
    const e = this.rowError();
    return e?.pointId === p.id ? e.message : null;
  }

  protected askGenerate(p: GrammarPoint): void {
    const lesson = this.lessonOf(p);
    if (this.busy()) {
      return;
    }
    if (lesson && (lesson.edited || lesson.status === 'published')) {
      this.confirming.set(p);
      return;
    }
    void this.generate(p, false);
  }

  protected confirmGenerate(): void {
    const p = this.confirming();
    this.confirming.set(null);
    if (p) {
      void this.generate(p, true);
    }
  }

  protected confirmMessage(): string {
    const p = this.confirming();
    if (!p) {
      return '';
    }
    const lesson = this.lessonOf(p);
    const what = lesson?.status === 'published' ? 'đã đăng' : 'đã sửa tay';
    return `Bài ngữ pháp “${p.titleVi}” ${what}. Sinh lại sẽ thay toàn bộ nội dung bằng một bản nháp mới.`;
  }

  private async generate(p: GrammarPoint, force: boolean): Promise<void> {
    await this.run(p, 'generate', () => this.api.generateGrammarLesson(p.id, force), `Đã sinh bản nháp cho “${p.titleVi}”.`);
  }

  protected publish(p: GrammarPoint): Promise<void> {
    return this.run(p, 'publish', () => this.api.publishGrammarLesson(p.id), `Đã đăng “${p.titleVi}”.`);
  }

  protected unpublish(p: GrammarPoint): Promise<void> {
    return this.run(p, 'unpublish', () => this.api.unpublishGrammarLesson(p.id), `Đã gỡ “${p.titleVi}”.`);
  }

  private async run(
    p: GrammarPoint,
    action: 'generate' | 'publish' | 'unpublish',
    call: () => ReturnType<AdminApiService['generateGrammarLesson']>,
    done: string,
  ): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.busy.set(p.id);
    this.busyAction.set(action);
    this.rowError.set(null);
    this.note.set('');
    try {
      const lesson: GrammarLessonAdmin = await firstValueFrom(call());
      // The row only needs the summary fields; the content stays on the detail page.
      const summary: GrammarLessonSummary = {
        pointId: lesson.pointId,
        status: lesson.status,
        edited: lesson.edited,
        updatedAt: lesson.updatedAt,
        publishedAt: lesson.publishedAt,
        flags: lesson.checks.filter((c) => !c.confirmed).length,
        checked: lesson.checkedAt !== null,
        verified: lesson.verifiedAt !== null,
      };
      this.lessons.update((all) => ({ ...all, [p.id]: summary }));
      this.note.set(done);
    } catch (err) {
      const failure = grammarFailure(err, action === 'generate' ? GENERATE_FAILED : ACTION_FAILED, action === 'generate');
      const detail = failure.fields.length ? ` ${failure.fields.join('; ')}` : '';
      this.rowError.set({ pointId: p.id, message: failure.message + detail });
    } finally {
      this.busy.set(null);
      this.busyAction.set(null);
    }
  }
}
