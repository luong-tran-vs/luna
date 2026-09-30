import { ChangeDetectionStrategy, Component, ElementRef, inject, signal, viewChild } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormArray, FormControl, FormGroup, NonNullableFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, EMPTY, firstValueFrom, Subscription } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { isRunning, JobKind, Lesson } from '../../../core/models/lesson';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { pollWhile } from '../../../shared/utils/poll-while';
import { AdminApiService } from '../admin-api.service';
import { StatusChip } from '../status-chip/status-chip';

type AnnotationRow = FormGroup<{
  text: FormControl<string>;
  lemma: FormControl<string>;
  meaningVi: FormControl<string>;
}>;

@Component({
  selector: 'lu-lesson-detail',
  imports: [RouterLink, ReactiveFormsModule, StatusChip, ConfirmDialog],
  templateUrl: './lesson-detail.html',
  styleUrl: './lesson-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonDetail {
  private readonly api = inject(AdminApiService);
  private readonly router = inject(Router);
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

  private readonly player = viewChild<ElementRef<HTMLAudioElement>>('player');
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

  protected play(url: string): void {
    const audio = this.player()?.nativeElement;
    if (!audio) {
      return;
    }
    audio.src = url;
    void audio.play().catch(() => this.error.set('Không phát được audio.'));
  }

  protected async retry(job: JobKind): Promise<void> {
    await this.act(async () => {
      this.lesson.set(await firstValueFrom(this.api.retry(this.id, job)));
      this.startPolling();
    });
  }

  // --- annotations ---

  protected startEditing(): void {
    const l = this.lesson();
    if (!l) {
      return;
    }
    this.rows.clear();
    for (const a of l.annotations) {
      this.rows.push(this.row(a.text, a.lemma, a.meaningVi));
    }
    this.existingCount.set(l.annotations.length);
    this.rowErrors.set({});
    this.editing.set(true);
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
