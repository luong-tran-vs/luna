import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import {
  DeleteGroupsReport,
  GroupAdmin,
  TestAdmin,
  ToeicPart,
  ToeicStatus,
} from '../../../../core/models/toeic';
import { ConfirmDialog } from '../../../../shared/components/confirm-dialog/confirm-dialog';
import { ToeicApiService } from '../../../../core/services/toeic-api.service';
import { DraftDialog } from '../draft-dialog';
import {
  clampPage,
  pageCountOf,
  pageSlice,
  Pager,
  rangeText,
} from '../../../../shared/components/pager/pager';
import { BankPictures } from './bank-pictures';
import { GroupPreview } from '../group-preview';
import { ToeicImportDialog } from '../import-dialog';
import {
  atLevel,
  byLevel,
  bySource,
  byTest,
  GROUPS_PAGE_SIZE,
  GROUPS_PAGE_SIZES,
  LevelSort,
  NO_TEST,
  SourceFilter,
  STATUS_LABELS,
  TOEIC_LEVELS,
} from '../toeic-admin.logic';

/**
 * Kho cụm câu (FR-040…FR-042): every imported group, filtered by part, status and test; preview,
 * publish, hide, publish the drafts shown, and "Nhập JSON". Tests are made from these groups.
 */
@Component({
  selector: 'lu-toeic-groups-page',
  imports: [
    RouterLink,
    ToeicImportDialog,
    GroupPreview,
    DraftDialog,
    BankPictures,
    Pager,
    ConfirmDialog,
  ],
  templateUrl: './groups-page.html',
  styleUrls: ['../toeic-admin.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GroupsPage {
  private readonly api = inject(ToeicApiService);

  protected readonly groups = signal<GroupAdmin[] | null>(null);
  protected readonly tests = signal<TestAdmin[]>([]);
  protected readonly part = signal<ToeicPart | null>(null);
  protected readonly status = signal<ToeicStatus | ''>('');
  private readonly query = inject(ActivatedRoute).snapshot.queryParamMap;
  protected readonly test = signal(this.query.get('test') ?? '');
  protected readonly source = signal<SourceFilter>(sourceParam(this.query.get('source')));
  protected readonly level = signal<number | null>(null);
  protected readonly levels = TOEIC_LEVELS;
  protected readonly sort = signal<LevelSort>('');
  /** The page of the list shown (from 1); back to 1 whenever a filter or the order changes. */
  protected readonly page = signal(1);
  protected readonly pageSize = signal<number>(GROUPS_PAGE_SIZE);
  protected readonly pageSizes = GROUPS_PAGE_SIZES;
  protected readonly error = signal('');
  protected readonly importing = signal(false);
  protected readonly drafting = signal(false);
  protected readonly preview = signal<string | null>(null);
  protected readonly busy = signal<string | null>(null);
  private readonly pictures = viewChild(BankPictures);

  protected readonly parts: ToeicPart[] = [1, 2, 3, 4, 5, 6, 7];
  protected readonly noTest = NO_TEST;
  protected readonly shown = computed(() => {
    const list = this.groups();
    return list
      ? byLevel(
          atLevel(bySource(byTest(list, this.test()), this.source()), this.level()),
          this.sort(),
        )
      : null;
  });
  private readonly total = computed(() => this.shown()?.length ?? 0);
  protected readonly pageCount = computed(() => pageCountOf(this.total(), this.pageSize()));
  /** The page shown, kept inside the list when it gets shorter (a group hidden, a reload). */
  protected readonly currentPage = computed(() =>
    clampPage(this.page(), this.total(), this.pageSize()),
  );
  protected readonly pageGroups = computed(() =>
    pageSlice(this.shown() ?? [], this.page(), this.pageSize()),
  );
  protected readonly pageSummary = computed(
    () => `${rangeText(this.page(), this.total(), this.pageSize())} cụm`,
  );
  protected readonly drafts = computed(() =>
    (this.shown() ?? []).filter((g) => g.status === 'draft'),
  );

  // --- deleting (one group or the ticked ones) ---
  /** The ticked groups; only those the filters show are deleted. */
  protected readonly selected = signal<ReadonlySet<string>>(new Set());
  protected readonly selectedShown = computed(() =>
    (this.shown() ?? []).filter((g) => this.selected().has(g.key)).map((g) => g.key),
  );
  protected readonly pageAllSelected = computed(() => {
    const page = this.pageGroups();
    return page.length > 0 && page.every((g) => this.selected().has(g.key));
  });
  protected readonly pageSomeSelected = computed(
    () => !this.pageAllSelected() && this.pageGroups().some((g) => this.selected().has(g.key)),
  );
  /** The keys the confirm dialog is about; null when it is closed. */
  protected readonly deleting = signal<string[] | null>(null);
  protected readonly deleteReport = signal<DeleteGroupsReport | null>(null);
  protected readonly deleteMessage = computed(() => {
    const keys = this.deleting() ?? [];
    const what = keys.length === 1 ? `cụm ${keys[0]}` : `${keys.length} cụm đã chọn`;
    return `Xoá hẳn ${what} khỏi kho, cùng ảnh của chúng; không hoàn tác được. Cụm đang thuộc đề hoặc đã có bài làm của người học sẽ được giữ lại (hãy ẩn chúng thay vì xoá).`;
  });
  protected readonly emptyText = computed(() =>
    this.groups()?.length === 0 && !this.part() && !this.status()
      ? 'Chưa có cụm nào. Bấm "Nhập JSON".'
      : 'Không có cụm nào khớp bộ lọc.',
  );

  constructor() {
    this.load();
    this.api.tests().subscribe({ next: (t) => this.tests.set(t), error: () => undefined });
  }

  protected load(): void {
    this.error.set('');
    this.api.groups({ part: this.part(), status: this.status() }).subscribe({
      next: (g) => this.groups.set(g),
      error: () => this.error.set('Không tải được danh sách cụm.'),
    });
  }

  protected setPart(event: Event): void {
    const v = (event.target as HTMLSelectElement).value;
    this.part.set(v ? (Number(v) as ToeicPart) : null);
    this.page.set(1);
    this.load();
  }

  protected setStatus(event: Event): void {
    this.status.set((event.target as HTMLSelectElement).value as ToeicStatus | '');
    this.page.set(1);
    this.load();
  }

  protected setSource(event: Event): void {
    this.source.set(sourceParam((event.target as HTMLSelectElement).value));
    this.page.set(1);
  }

  protected setTest(event: Event): void {
    this.test.set((event.target as HTMLSelectElement).value);
    this.page.set(1);
  }

  protected setLevel(event: Event): void {
    const v = Number((event.target as HTMLSelectElement).value);
    this.level.set(TOEIC_LEVELS.some((l) => l === v) ? v : null);
    this.page.set(1);
  }

  protected setPageSize(event: Event): void {
    const v = Number((event.target as HTMLSelectElement).value);
    this.pageSize.set(GROUPS_PAGE_SIZES.some((s) => s === v) ? v : GROUPS_PAGE_SIZE);
    this.page.set(1);
  }

  protected setSort(event: Event): void {
    const v = (event.target as HTMLSelectElement).value;
    this.sort.set(v === 'asc' || v === 'desc' ? v : '');
    this.page.set(1);
  }

  /** Another page of the list: back to the top of the table. */
  protected goToPage(page: number): void {
    this.page.set(page);
    document.getElementById('groups-list')?.scrollIntoView({ block: 'start' });
  }

  /** The "Ảnh" column: ✓ / "Chưa có" for a Part 1 photo, "1/3" for the texts of Part 6/7, — without slots. */
  protected pictureLabel(g: GroupAdmin): string {
    if (g.needsPhoto) {
      return g.hasPhoto ? '✓' : 'Chưa có';
    }
    return g.pictureSlots ? `${g.pictures}/${g.pictureSlots}` : '—';
  }

  protected statusLabel(s: ToeicStatus): string {
    return STATUS_LABELS[s];
  }

  protected setGroupStatus(g: GroupAdmin, status: ToeicStatus): void {
    this.busy.set(g.key);
    this.api.setGroupStatus(g.key, status).subscribe({
      next: () => {
        this.busy.set(null);
        this.groups.update((list) =>
          (list ?? []).map((x) => (x.key === g.key ? { ...x, status } : x)),
        );
      },
      error: () => {
        this.busy.set(null);
        this.error.set(`Chưa đổi được trạng thái của ${g.key}.`);
      },
    });
  }

  protected toggle(key: string): void {
    this.selected.update((s) => {
      const next = new Set(s);
      if (!next.delete(key)) {
        next.add(key);
      }
      return next;
    });
  }

  /** The box in the table's head: ticks every group of the page, or unticks them all. */
  protected togglePage(): void {
    const keys = this.pageGroups().map((g) => g.key);
    const all = this.pageAllSelected();
    this.selected.update((s) => {
      const next = new Set(s);
      for (const k of keys) {
        if (all) {
          next.delete(k);
        } else {
          next.add(k);
        }
      }
      return next;
    });
  }

  protected clearSelection(): void {
    this.selected.set(new Set());
  }

  protected askDelete(keys: string[]): void {
    if (keys.length > 0) {
      this.deleteReport.set(null);
      this.deleting.set(keys);
    }
  }

  protected confirmDelete(): void {
    const keys = this.deleting();
    this.deleting.set(null);
    if (!keys) {
      return;
    }
    this.busy.set('*');
    this.error.set('');
    this.api.deleteGroups(keys).subscribe({
      next: (rep) => {
        this.busy.set(null);
        this.deleteReport.set(rep);
        const gone = new Set(rep.deleted);
        this.selected.update((s) => new Set([...s].filter((k) => !gone.has(k))));
        this.groups.update((list) => (list ?? []).filter((g) => !gone.has(g.key)));
        this.pictures()?.refresh();
        // The report is above the table: bring it into view after a row far down was deleted.
        setTimeout(() =>
          document.querySelector('.delete-report')?.scrollIntoView({ block: 'center' }),
        );
      },
      error: () => {
        this.busy.set(null);
        this.error.set('Chưa xoá được cụm nào.');
      },
    });
  }

  protected publishDrafts(): void {
    const keys = this.drafts().map((g) => g.key);
    if (keys.length === 0) {
      return;
    }
    this.busy.set('*');
    this.api.setGroupsStatus(keys, 'published').subscribe({
      next: () => {
        this.busy.set(null);
        this.load();
      },
      error: () => {
        this.busy.set(null);
        this.error.set('Chưa đăng được các cụm nháp.');
      },
    });
  }

  /** The preview may have drawn a picture: the list shows it. */
  protected previewClosed(): void {
    this.preview.set(null);
    this.load();
  }

  /** After "Nhờ AI soạn": show the drafts to check them. */
  protected draftClosed(made: boolean): void {
    this.drafting.set(false);
    this.pictures()?.refresh();
    if (made) {
      this.status.set('draft');
      this.source.set('ai');
      this.page.set(1);
      this.load();
    }
  }

  protected importClosed(changed: boolean): void {
    this.importing.set(false);
    if (changed) {
      this.load();
    }
  }
}

/** The "Nguồn" filter of a query value. */
function sourceParam(v: string | null): SourceFilter {
  return v === 'ai' || v === 'file' ? v : '';
}
