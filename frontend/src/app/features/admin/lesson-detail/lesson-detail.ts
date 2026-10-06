import { afterNextRender, ChangeDetectionStrategy, Component, computed, inject, Injector, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormArray, FormControl, FormGroup, NonNullableFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, EMPTY, firstValueFrom, Subscription } from 'rxjs';

import { DatePipe } from '@angular/common';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import { SpeechService } from '../../../core/services/speech.service';
import { AnnotationInput, flagsOf, isRunning, JobKind, Lesson, LessonFlag, openFlagCount } from '../../../core/models/lesson';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { pollWhile } from '../../../shared/utils/poll-while';
import { AdminApiService } from '../admin-api.service';
import { lessonCheckFailure } from '../lesson-check-errors';
import { FlagNote } from '../flag-note/flag-note';
import { LessonExtras } from '../lesson-extras/lesson-extras';
import { StatusChip } from '../status-chip/status-chip';
import { LessonImages } from './lesson-images/lesson-images';
import { PracticeSection } from './practice-section/practice-section';
import { Loading } from '../../../shared/components/loading/loading';

type AnnotationRow = FormGroup<{
  text: FormControl<string>;
  lemma: FormControl<string>;
  meaningVi: FormControl<string>;
}>;

@Component({
  selector: 'lu-lesson-detail',
  imports: [Loading, RouterLink, ReactiveFormsModule, StatusChip, ConfirmDialog, LessonExtras, PracticeSection, LessonImages, FlagNote, DatePipe],
  templateUrl: './lesson-detail.html',
  styleUrl: './lesson-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonDetail {
  private readonly api = inject(AdminApiService);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);
  private readonly fb = inject(NonNullableFormBuilder);
  protected readonly id = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';

  protected readonly lesson = signal<Lesson | null>(null);
  protected readonly notFound = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly busy = signal(false);
  protected readonly deleteOpen = signal(false);

  protected readonly editing = signal(false);
  protected readonly existingCount = signal(0);
  protected readonly rowErrors = signal<Record<string, string>>({});
  protected readonly rows = new FormArray<AnnotationRow>([]);
  /** Position each edited row had in the saved list, so its flag stays with it after removals. */
  private readonly origin = new WeakMap<AnnotationRow, number>();
  protected readonly deleteAnnotationAt = signal<number | null>(null);

  private readonly speech = inject(SpeechService);
  private readonly destroy = takeUntilDestroyed<Lesson>();
  private poll?: Subscription;

  constructor() {
    this.startPolling();
  }

  /** Loads the lesson and refreshes it every 5 s while its work is running. */
  private startPolling(): void {
    this.poll?.unsubscribe();
    this.poll = pollWhile(() => this.api.get(this.id), isRunning)
      .pipe(
        catchError((err: unknown) => {
          if (err instanceof ApiError && err.status === 404) {
            this.notFound.set(true);
          } else {
            this.error.set('Không tải được bài học.');
          }
          return EMPTY;
        }),
        this.destroy,
      )
      .subscribe((l) => this.lesson.set(l));
  }

  /** Reads a sentence with the browser's voice, as learners hear it. */
  protected play(text: string): void {
    this.speech.speak(text, 1, { failed: () => this.error.set('Trình duyệt chưa đọc được câu này.') });
  }

  // --- F15: running annotation again replaces admin edits, so it asks first ---

  protected readonly retryOpen = signal(false);
  /** Hand-edited parts that running annotation again would replace. */
  private readonly editedParts = computed(() => {
    const l = this.lesson();
    const parts: string[] = [];
    if (l?.annotations.some((a) => a.editedByAdmin)) {
      parts.push('chú thích từ đã sửa tay');
    }
    if (l?.extrasEditedByAdmin) {
      parts.push('câu hỏi, ngữ pháp, đề viết đã sửa tay');
    }
    return parts;
  });
  protected readonly retryMessage = computed(
    () => `AI sẽ viết lại toàn bộ chú thích, câu hỏi, ngữ pháp và đề viết. Sẽ bị thay: ${this.editedParts().join('; ')}.`,
  );

  protected askRetryAnnotate(): void {
    if (this.editedParts().length > 0) {
      this.retryOpen.set(true);
    } else {
      void this.retry('annotate');
    }
  }

  protected confirmRetryAnnotate(): void {
    this.retryOpen.set(false);
    void this.retry('annotate');
  }

  protected async retry(job: JobKind): Promise<void> {
    await this.act(async () => {
      this.lesson.set(await firstValueFrom(this.api.retry(this.id, job)));
      this.startPolling();
    });
  }

  /** F17: regeneration was queued; poll until the practice is done. */
  protected onPracticeRegenerated(lesson: Lesson): void {
    this.lesson.set(lesson);
    this.startPolling();
  }

  // --- F22: AI check ---

  protected readonly checking = signal(false);
  protected readonly checkError = signal<string | null>(null);
  protected readonly flagBusy = signal(false);

  protected readonly sentenceFlags = computed(() => flagsOf(this.lesson()?.review, 'sentence'));
  protected readonly annotationFlags = computed(() => flagsOf(this.lesson()?.review, 'annotation'));
  protected readonly questionFlags = computed(() => flagsOf(this.lesson()?.review, 'question'));
  protected readonly translationFlags = computed(() => flagsOf(this.lesson()?.review, 'translation'));
  protected readonly openFlags = computed(() => openFlagCount(this.lesson()?.review));

  /** Why the check cannot run now, or null. */
  protected readonly checkBlocked = computed(() => {
    const l = this.lesson();
    return l && l.annotationStatus !== 'done' ? 'Cần chú thích xong trước' : null;
  });

  protected readonly checkText = computed(() => {
    const r = this.lesson()?.review;
    if (!r) {
      return 'Chưa kiểm tra';
    }
    if (r.verifiedAt) {
      return 'Đã xác nhận kiểm tra xong';
    }
    const open = this.openFlags();
    if (open > 0) {
      return `${open} chỗ cần xem`;
    }
    return r.flags.length === 0 ? 'Đã kiểm tra, không có chỗ nào bị gắn cờ' : 'Đã xem hết các chỗ bị gắn cờ';
  });

  protected flagAt(flags: LessonFlag[], index: number): LessonFlag | null {
    return flags.find((f) => f.index === index) ?? null;
  }

  protected check(): Promise<void> {
    return this.runCheck(
      this.checking,
      () => this.api.checkLesson(this.id),
      'Không kiểm tra được, vui lòng thử lại.',
      true,
    );
  }

  protected confirmFlag(flag: LessonFlag): Promise<void> {
    return this.runCheck(
      this.flagBusy,
      () => this.api.confirmLessonFlag(this.id, flag.area, flag.index),
      'Không xác nhận được, vui lòng thử lại.',
    );
  }

  protected verify(): Promise<void> {
    return this.runCheck(
      this.flagBusy,
      () => this.api.verifyLesson(this.id),
      'Không xác nhận được, vui lòng thử lại.',
    );
  }

  private async runCheck(
    flag: { set(v: boolean): void; (): boolean },
    call: () => ReturnType<AdminApiService['checkLesson']>,
    fallback: string,
    ai = false,
  ): Promise<void> {
    if (flag() || this.checking() || this.flagBusy()) {
      return;
    }
    flag.set(true);
    this.checkError.set(null);
    try {
      this.lesson.set(await firstValueFrom(call()));
    } catch (err) {
      this.checkError.set(lessonCheckFailure(err, fallback, ai));
    } finally {
      flag.set(false);
    }
  }

  // --- annotations ---

  protected startEditing(): void {
    const l = this.lesson();
    if (!l) {
      return;
    }
    this.rows.clear();
    for (const a of l.annotations) {
      const row = this.row(a.text, a.lemma, a.meaningVi);
      this.origin.set(row, this.rows.length);
      this.rows.push(row);
    }
    this.existingCount.set(l.annotations.length);
    this.rowErrors.set({});
    this.editing.set(true);
  }

  /** Flag of the annotation a row in the editor started as; null for new rows. */
  protected flagOfRow(row: AnnotationRow): LessonFlag | null {
    const at = this.origin.get(row);
    return at === undefined ? null : this.flagAt(this.annotationFlags(), at);
  }

  /** Opens the editor (if needed) and puts the cursor in the meaning of that annotation's row. */
  protected editAnnotation(index: number, row?: AnnotationRow): void {
    if (!this.editing()) {
      this.startEditing();
    }
    const position = row ? this.rows.controls.indexOf(row) : index;
    afterNextRender(() => document.getElementById('ann-meaning-' + position)?.focus(), { injector: this.injector });
  }

  protected askDeleteAnnotation(index: number): void {
    this.deleteAnnotationAt.set(index);
  }

  /** Saves the list without that annotation, using what is in the editor when it is open. */
  protected async confirmDeleteAnnotation(): Promise<void> {
    const index = this.deleteAnnotationAt();
    this.deleteAnnotationAt.set(null);
    const l = this.lesson();
    if (index === null || !l) {
      return;
    }
    let list: AnnotationInput[];
    if (this.editing()) {
      list = this.rows.controls.filter((r) => this.origin.get(r) !== index).map((r) => r.getRawValue());
    } else {
      list = l.annotations.filter((_, i) => i !== index).map((a) => ({ text: a.text, lemma: a.lemma, meaningVi: a.meaningVi }));
    }
    this.rowErrors.set({});
    await this.act(async () => {
      this.lesson.set(await firstValueFrom(this.api.saveAnnotations(this.id, list)));
      this.cancelEditing();
    });
  }

  protected annotationAt(index: number): string {
    return this.lesson()?.annotations[index]?.text ?? '';
  }

  protected addRow(): void {
    this.rows.push(this.row('', '', ''));
  }

  protected removeRow(index: number): void {
    this.rows.removeAt(index);
    if (index < this.existingCount()) {
      this.existingCount.update((n) => n - 1);
    }
  }

  protected cancelEditing(): void {
    this.editing.set(false);
    this.rows.clear();
  }

  protected rowError(index: number): string | null {
    const errs = this.rowErrors();
    return errs[`annotations.${index}.text`] ?? errs[`annotations.${index}.lemma`] ?? errs[`annotations.${index}.meaningVi`] ?? null;
  }

  protected async saveAnnotations(): Promise<void> {
    this.rowErrors.set({});
    await this.act(async () => {
      this.lesson.set(await firstValueFrom(this.api.saveAnnotations(this.id, this.rows.getRawValue())));
      this.cancelEditing();
    });
  }

  private row(text: string, lemma: string, meaningVi: string): AnnotationRow {
    return this.fb.group({ text: [text], lemma: [lemma], meaningVi: [meaningVi] });
  }

  // --- delete ---

  protected async confirmDelete(): Promise<void> {
    this.deleteOpen.set(false);
    await this.act(async () => {
      await firstValueFrom(this.api.remove(this.id));
      await this.router.navigateByUrl('/admin');
    });
  }

  /** Runs an action, showing server messages and field errors. */
  private async act(action: () => Promise<void>): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      await action();
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      if (body?.fields) {
        this.rowErrors.set(body.fields);
        this.error.set('Vui lòng sửa các chú thích được đánh dấu.');
      } else {
        this.error.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
      }
    } finally {
      this.busy.set(false);
    }
  }
}
