import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { GroupAdmin, TestAdmin, ToeicStatus } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { errorBody, fieldLines, parseJson } from '../json-import/json-import';
import { JsonSource } from '../json-import/json-source';
import { ComposeDialog } from './compose-dialog';
import { ToeicImportDialog } from './import-dialog';
import { bankSummary, NO_TEST, STATUS_LABELS, testCard } from './toeic-admin.logic';

/**
 * The TOEIC home (FR-043): the tests as cards (what each has and misses; a card opens the test),
 * "Tạo đề", and the group bank in numbers with "Nhập JSON" and a link to Kho cụm câu.
 */
@Component({
  selector: 'lu-toeic-admin',
  imports: [RouterLink, ToeicImportDialog, JsonSource, ComposeDialog],
  templateUrl: './toeic-admin.html',
  styleUrls: [
    '../../../shared/components/confirm-dialog/confirm-dialog.css',
    './toeic-admin.css',
    './toeic-home.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToeicAdmin {
  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);

  protected readonly tests = signal<TestAdmin[] | null>(null);
  protected readonly groups = signal<GroupAdmin[] | null>(null);
  protected readonly error = signal('');
  protected readonly importing = signal(false);
  protected readonly creating = signal(false);
  protected readonly composing = signal(false);
  protected readonly testText = signal('');
  protected readonly testError = signal('');
  protected readonly testFields = signal<{ key: string; text: string }[]>([]);

  protected readonly cards = computed(() => this.tests()?.map(testCard) ?? null);
  protected readonly bank = computed(() => {
    const g = this.groups();
    return g ? bankSummary(g) : null;
  });
  protected readonly noTest = NO_TEST;

  constructor() {
    this.loadTests();
    this.loadGroups();
  }

  private loadTests(): void {
    this.api.tests().subscribe({
      next: (t) => this.tests.set(t),
      error: () => this.error.set('Không tải được danh sách đề.'),
    });
  }

  private loadGroups(): void {
    this.api.groups({}).subscribe({ next: (g) => this.groups.set(g), error: () => undefined });
  }

  protected statusLabel(s: ToeicStatus): string {
    return STATUS_LABELS[s];
  }

  protected importClosed(changed: boolean): void {
    this.importing.set(false);
    if (changed) {
      this.loadGroups();
      this.loadTests();
    }
  }

  protected createTest(): void {
    this.testError.set('');
    this.testFields.set([]);
    const parsed = parseJson(this.testText());
    if ('error' in parsed) {
      this.testError.set(parsed.error);
      return;
    }
    this.api.createTest(parsed.value).subscribe({
      next: (t) => void this.router.navigate(['/admin/toeic/tests', t.key]),
      error: (err: unknown) => {
        const e = errorBody(err);
        this.testError.set(
          e.error === 'exists'
            ? 'Mã đề đã có: mở đề đó để sửa.'
            : (e.message ?? 'Chưa tạo được đề.'),
        );
        if (e.fields) {
          this.testFields.set(fieldLines(e.fields));
        }
      },
    });
  }
}
