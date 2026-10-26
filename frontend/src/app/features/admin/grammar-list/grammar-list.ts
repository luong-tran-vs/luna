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
import { GrammarImportDialog } from './import-dialog/import-dialog';

const GENERATE_FAILED = 'Sinh bài thất bại, vui lòng thử lại.';
const ACTION_FAILED = 'Không thực hiện được, vui lòng thử lại.';

/** F20: the curriculum by level with the state of each point's grammar lesson and its actions. */
@Component({
  selector: 'lu-grammar-list',
  imports: [Loading, ConfirmDialog, RouterLink, GrammarImportDialog],
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
  protected readonly busyAction = signal<'generate' | 'publish' | 'unpublish' | 'delete' | null>(
    null,
  );
  /** Point waiting for the admin to confirm deleting its draft (added 2026-10-26). */
  protected readonly deleting = signal<GrammarPoint | null>(null);
  protected readonly rowError = signal<{ pointId: string; message: string } | null>(null);
  protected readonly note = signal('');
  /** Point waiting for the admin to confirm replacing its lesson. */
  protected readonly confirming = signal<GrammarPoint | null>(null);
  /** Point a lesson is being imported into (FR-030). */
  protected readonly importing = signal<GrammarPoint | null>(null);
  /** The control focused before the import dialog opened, focused again when it closes. */
  private opener: HTMLElement | null = null;

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

  protected askImport(p: GrammarPoint): void {
    if (this.busy()) {
      return;
    }
    this.opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    this.importing.set(p);
  }

  protected cancelImport(): void {
    this.importing.set(null);
    this.opener?.focus();
    this.opener = null;
  }

  /** The imported lesson is the point's draft now. */
  protected imported(lesson: GrammarLessonAdmin): void {
    const p = this.importing();
    this.lessons.update((all) => ({ ...all, [lesson.pointId]: summaryOf(lesson) }));
    this.note.set(`Đã nhập bài “${p?.titleVi ?? lesson.pointId}” thành nháp.`);
    this.cancelImport();
  }

  /** Only a draft can be deleted: a published lesson is unpublished first. */
  protected askDelete(p: GrammarPoint): void {
    if (this.busy()) {
      return;
    }
    this.opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    this.deleting.set(p);
  }

  protected cancelDelete(): void {
    this.deleting.set(null);
    this.opener?.focus();
    this.opener = null;
  }

  protected deleteMessage(): string {
    const p = this.deleting();
    return p
      ? `Xoá bài nháp của “${p.titleVi}”? Điểm này sẽ về “Chưa có bài” để sinh hoặc nhập lại. Không lấy lại được.`
      : '';
  }

  protected async confirmDelete(): Promise<void> {
    const p = this.deleting();
    this.deleting.set(null);
    this.opener = null;
    if (!p || this.busy()) {
      return;
    }
    this.busy.set(p.id);
    this.busyAction.set('delete');
    this.rowError.set(null);
    this.note.set('');
    try {
      await firstValueFrom(this.api.deleteGrammarLesson(p.id));
      this.lessons.update((all) => {
        const next = { ...all };
        delete next[p.id];
        return next;
      });
      this.note.set(`Đã xoá bài nháp của “${p.titleVi}”.`);
    } catch (err) {
      this.rowError.set({ pointId: p.id, message: grammarFailure(err, ACTION_FAILED, false).message });
    } finally {
      this.busy.set(null);
      this.busyAction.set(null);
    }
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
      this.lessons.update((all) => ({ ...all, [p.id]: summaryOf(lesson) }));
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

/** The row of a lesson: the summary fields; the content stays on the detail page. */
function summaryOf(lesson: GrammarLessonAdmin): GrammarLessonSummary {
  return {
    pointId: lesson.pointId,
    status: lesson.status,
    edited: lesson.edited,
    imported: lesson.imported,
    updatedAt: lesson.updatedAt,
    publishedAt: lesson.publishedAt,
    flags: lesson.checks.filter((c) => !c.confirmed).length,
    checked: lesson.checkedAt !== null,
    verified: lesson.verifiedAt !== null,
  };
}
