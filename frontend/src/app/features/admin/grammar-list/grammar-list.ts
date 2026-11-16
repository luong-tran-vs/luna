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
  PICTURE_MASTERY,
  PICTURE_PRACTICE,
} from '../../../core/models/grammar-admin';
import { SpotCount } from '../../../core/models/games';
import { Level, LEVELS } from '../../../core/models/lesson';
import { GamesApiService } from '../../../core/services/games-api.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { Loading } from '../../../shared/components/loading/loading';
import { AdminApiService } from '../admin-api.service';
import { grammarFailure } from '../grammar-errors';
import { GrammarImportDialog } from './import-dialog/import-dialog';
import { GrammarLevelMedia } from '../grammar-images/grammar-level-media';

const GENERATE_FAILED = 'Sinh bài thất bại, vui lòng thử lại.';
const ACTION_FAILED = 'Không thực hiện được, vui lòng thử lại.';
/** A point with fewer "Bắn lỗi" sentences is flagged (FR-026a). */
const SPOT_ENOUGH = 10;

/** F20: the curriculum by level with the state of each point's grammar lesson and its actions. */
@Component({
  selector: 'lu-grammar-list',
  imports: [Loading, ConfirmDialog, RouterLink, GrammarImportDialog, GrammarLevelMedia],
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

  /** The drawable points of the level that have a lesson, whose pictures the batch draws. */
  protected readonly lessonIds = computed(() => {
    const lessons = this.lessons();
    // Only the points that can be drawn get pictures (added 2026-11-06).
    return (this.points() ?? []).filter((p) => p.drawable && !!lessons[p.id]).map((p) => p.id);
  });

  /** Every point of the level that has a lesson: the hint pictures are for all of them (added 2026-11-16). */
  protected readonly allLessonIds = computed(() => {
    const lessons = this.lessons();
    return (this.points() ?? []).filter((p) => !!lessons[p.id]).map((p) => p.id);
  });

  /** The title of each point of the level, by id (the list of "Gỡ ảnh ví dụ"). */
  protected readonly titles = computed(() =>
    Object.fromEntries((this.points() ?? []).map((p) => [p.id, p.titleVi])),
  );

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

  /** F26: the "Bắn lỗi" sentences of each published point (FR-026a); a failed load shows "—". */
  private readonly games = inject(GamesApiService);
  private readonly spotCounts = signal<Record<string, SpotCount>>({});

  private loadSeq = 0;

  constructor() {
    this.load();
    this.loadReports();
    this.loadSpotCounts();
  }

  private loadSpotCounts(): void {
    this.games.spotCounts().subscribe({
      next: (r) => this.spotCounts.set(Object.fromEntries(r.points.map((c) => [c.pointId, c]))),
      error: () => this.spotCounts.set({}),
    });
  }

  /** The level's "Bắn lỗi" sentences, drafts, and how many published points have under 10. */
  protected readonly spotLevel = computed(() => {
    const list = Object.values(this.spotCounts()).filter((c) => c.level === this.level());
    return list.length
      ? {
          total: list.reduce((n, c) => n + c.total, 0),
          draft: list.reduce((n, c) => n + c.draft, 0),
          low: list.filter((c) => c.total < SPOT_ENOUGH).length,
          pointIds: list.filter((c) => c.draft > 0).map((c) => c.pointId),
        }
      : null;
  });

  /** The point's "Bắn lỗi" count: null without a published lesson; low under 10 sentences. */
  protected spotOf(p: GrammarPoint): { total: number; draft: number; low: boolean } | null {
    const c = this.spotCounts()[p.id];
    return c ? { total: c.total, draft: c.draft, low: c.total < SPOT_ENOUGH } : null;
  }

  /** "Duyệt nhanh" (added 2026-11-16): the drafts waiting for the admin's confirmation. */
  protected readonly approvingDrafts = signal<{
    pointIds: string[];
    count: number;
    label: string;
  } | null>(null);
  /** 'level', or the point whose drafts are being approved. */
  protected readonly spotBusy = signal<string | null>(null);

  protected askApproveLevel(): void {
    const s = this.spotLevel();
    if (s && s.draft > 0 && !this.spotBusy()) {
      this.approvingDrafts.set({
        pointIds: s.pointIds,
        count: s.draft,
        label: `trình độ ${this.level()}`,
      });
    }
  }

  protected askApprovePoint(p: GrammarPoint): void {
    const c = this.spotCounts()[p.id];
    if (c && c.draft > 0 && !this.spotBusy()) {
      this.approvingDrafts.set({ pointIds: [p.id], count: c.draft, label: `“${p.titleVi}”` });
    }
  }

  protected approveDraftsMessage(): string {
    const a = this.approvingDrafts();
    return a
      ? `Duyệt ${a.count} câu Bắn lỗi nháp của ${a.label}? Câu đã duyệt vào trò chơi ngay; vẫn sửa hoặc xoá được ở trang từng điểm.`
      : '';
  }

  protected async confirmApproveDrafts(): Promise<void> {
    const a = this.approvingDrafts();
    this.approvingDrafts.set(null);
    if (!a) {
      return;
    }
    this.spotBusy.set(a.pointIds.length === 1 ? a.pointIds[0] : 'level');
    this.note.set('');
    this.rowError.set(null);
    try {
      const r = await firstValueFrom(this.games.approveSpotDrafts(a.pointIds));
      this.note.set(`Đã duyệt ${r.approved} câu Bắn lỗi của ${a.label}.`);
      this.loadSpotCounts();
    } catch (err) {
      this.note.set(grammarFailure(err, ACTION_FAILED).message);
    } finally {
      this.spotBusy.set(null);
    }
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

  /**
   * The flag of a lesson's "Nhìn ảnh chọn từ" exercises (added 2026-11-06), for a point that can be
   * drawn: whether they were all added, then whether every one has its picture. ok is false while
   * something is missing.
   */
  protected pictureBadge(
    p: GrammarPoint,
    lesson: GrammarLessonSummary,
  ): { text: string; ok: boolean } | null {
    if (!p.drawable) {
      return null;
    }
    const total = lesson.pictures ?? 0;
    if (!lesson.picturesComplete) {
      return {
        text: `Thiếu bài nhìn ảnh (${total}/${PICTURE_PRACTICE + PICTURE_MASTERY})`,
        ok: false,
      };
    }
    const missing = total - (lesson.picturesDrawn ?? 0);
    return missing > 0
      ? { text: `Đủ bài nhìn ảnh · thiếu ${missing} ảnh`, ok: false }
      : { text: 'Đủ bài nhìn ảnh · đủ ảnh', ok: true };
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
      this.rowError.set({
        pointId: p.id,
        message: grammarFailure(err, ACTION_FAILED, false).message,
      });
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
    await this.run(
      p,
      'generate',
      () => this.api.generateGrammarLesson(p.id, force),
      `Đã sinh bản nháp cho “${p.titleVi}”.`,
    );
  }

  protected publish(p: GrammarPoint): Promise<void> {
    return this.run(
      p,
      'publish',
      () => this.api.publishGrammarLesson(p.id),
      `Đã đăng “${p.titleVi}”.`,
    );
  }

  protected unpublish(p: GrammarPoint): Promise<void> {
    return this.run(
      p,
      'unpublish',
      () => this.api.unpublishGrammarLesson(p.id),
      `Đã gỡ “${p.titleVi}”.`,
    );
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
      const failure = grammarFailure(
        err,
        action === 'generate' ? GENERATE_FAILED : ACTION_FAILED,
        action === 'generate',
      );
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
