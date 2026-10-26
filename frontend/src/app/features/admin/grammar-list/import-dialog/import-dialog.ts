import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { GrammarContent, GrammarLessonAdmin } from '../../../../core/models/grammar-admin';
import { ConfirmDialog } from '../../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../../admin-api.service';
import { errorBody, fieldLines, parseJson, trapTab } from '../../json-import/json-import';
import { JsonSource } from '../../json-import/json-source';

/** What a grammar lesson holds, section by section, for the preview. */
export function grammarSections(c: GrammarContent): { name: string; count: number }[] {
  return [
    { name: 'Mục tiêu', count: c.objective ? 1 : 0 },
    { name: 'Giải thích', count: c.explanation.length },
    { name: 'Khi nào dùng', count: c.usage.length },
    { name: 'Cấu trúc', count: c.structures.length },
    { name: 'Ví dụ', count: c.examples.length },
    { name: 'Lỗi hay gặp', count: c.mistakes.length },
    { name: 'Luyện tập', count: c.practice.length },
    { name: 'Kiểm tra cuối', count: c.mastery.length },
  ];
}

/**
 * FR-030 (added 2026-10-26): imports a grammar lesson written outside the app into a point. The JSON
 * is the content as the lesson editor saves it, pasted or from a file; it is checked on the server
 * without saving, and a point that has a lesson is replaced only after a confirmation.
 */
@Component({
  selector: 'lu-grammar-import-dialog',
  imports: [JsonSource, ConfirmDialog],
  templateUrl: './import-dialog.html',
  styleUrls: [
    '../../../../shared/components/confirm-dialog/confirm-dialog.css',
    './import-dialog.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class GrammarImportDialog {
  private readonly api = inject(AdminApiService);

  readonly open = input(false);
  readonly pointId = input('');
  readonly pointTitle = input('');

  readonly saved = output<GrammarLessonAdmin>();
  readonly closed = output<void>();

  protected readonly text = signal('');
  protected readonly checking = signal(false);
  protected readonly saving = signal(false);
  protected readonly parseError = signal('');
  protected readonly failure = signal('');
  protected readonly fields = signal<Record<string, string> | null>(null);
  protected readonly checked = signal<{
    text: string;
    replaced: boolean;
    content: GrammarContent;
  } | null>(null);
  /** Waiting for the admin to confirm replacing the point's lesson. */
  protected readonly confirming = signal(false);

  protected readonly fieldList = computed(() => fieldLines(this.fields() ?? {}));
  protected readonly sections = computed(() => {
    const c = this.checked()?.content;
    return c ? grammarSections(c) : [];
  });
  protected readonly canSave = computed(
    () =>
      this.checked()?.text === this.text() &&
      this.fields() === null &&
      !this.saving() &&
      !this.checking(),
  );
  protected readonly status = computed(() => {
    const c = this.checked();
    if (this.checking()) {
      return 'Đang kiểm tra…';
    }
    if (this.saving()) {
      return 'Đang lưu…';
    }
    if (this.fields()) {
      return 'Bài chưa hợp lệ, chưa lưu được.';
    }
    if (c && c.text !== this.text()) {
      return 'Nội dung đã đổi sau lần kiểm tra, hãy kiểm tra lại.';
    }
    if (c) {
      return c.replaced
        ? 'Đúng định dạng. Điểm này đã có bài: lưu sẽ thay bài cũ.'
        : 'Đúng định dạng. Đã kiểm tra, chưa lưu.';
    }
    return '';
  });
  protected readonly confirmMessage = computed(
    () =>
      `Điểm “${this.pointTitle()}” đã có bài. Nhập sẽ thay toàn bộ nội dung bằng bài này; bài đã đăng sẽ về nháp.`,
  );

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        this.pointId();
        untracked(() => this.reset());
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        untracked(() => this.dialog()?.nativeElement.querySelector<HTMLElement>('h2')?.focus());
      }
    });
  }

  private reset(): void {
    this.text.set('');
    this.parseError.set('');
    this.failure.set('');
    this.fields.set(null);
    this.checked.set(null);
    this.confirming.set(false);
  }

  protected onEscape(): void {
    // The confirmation closes itself first.
    if (this.open() && !this.confirming()) {
      this.closed.emit();
    }
  }

  protected onKeydown(event: KeyboardEvent): void {
    if (!this.confirming()) {
      trapTab(event, this.dialog()?.nativeElement);
    }
  }

  protected async check(): Promise<void> {
    const content = this.parse();
    if (!content) {
      return;
    }
    const text = this.text();
    this.checking.set(true);
    try {
      const res = await firstValueFrom(
        this.api.importGrammarLesson(this.pointId(), content.value, true, true),
      );
      this.fields.set(null);
      this.checked.set({ text, replaced: res.replaced, content: res.lesson.content });
    } catch (err) {
      this.showError(err);
    } finally {
      this.checking.set(false);
    }
  }

  /** Saves at once, or first asks when the point has a lesson. */
  protected askSave(): void {
    if (!this.canSave()) {
      return;
    }
    if (this.checked()?.replaced) {
      this.confirming.set(true);
      return;
    }
    void this.save();
  }

  protected confirmReplace(): void {
    this.confirming.set(false);
    void this.save();
  }

  protected async save(): Promise<void> {
    const content = this.parse();
    if (!content) {
      return;
    }
    this.saving.set(true);
    try {
      const res = await firstValueFrom(
        this.api.importGrammarLesson(
          this.pointId(),
          content.value,
          !!this.checked()?.replaced,
          false,
        ),
      );
      this.saved.emit(res.lesson);
    } catch (err) {
      this.showError(err);
    } finally {
      this.saving.set(false);
    }
  }

  /** The import JSON is the content itself, or an object with it under "content". */
  private parse(): { value: unknown } | null {
    this.failure.set('');
    const parsed = parseJson(this.text());
    if ('error' in parsed) {
      this.parseError.set(parsed.error);
      return null;
    }
    this.parseError.set('');
    const v = parsed.value as Record<string, unknown>;
    return { value: 'content' in v && Object.keys(v).length <= 2 ? v['content'] : v };
  }

  private showError(err: unknown): void {
    const e = errorBody(err);
    if (e.fields) {
      this.fields.set(e.fields);
      this.checked.set(null);
      return;
    }
    if (e.error === 'grammar_lesson_exists') {
      this.failure.set('Điểm này vừa có bài. Hãy kiểm tra lại trước khi lưu.');
      this.checked.set(null);
      return;
    }
    this.failure.set(e.message ?? 'Không gửi được bài, vui lòng thử lại.');
  }
}
