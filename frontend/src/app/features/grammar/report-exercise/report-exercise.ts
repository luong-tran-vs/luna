import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';

import { MAX_REPORT_NOTE, ReportReason, REPORT_REASONS } from '../../../core/models/grammar-study';
import { GrammarApiService } from '../../../core/services/grammar-api.service';

/**
 * F21: the "Báo lỗi câu này" button of one grammar exercise and the native <dialog> it opens
 * (reason + optional note). Once sent, the button turns into an "already reported" state for the
 * rest of the session. A failed send keeps what the learner typed and offers to try again.
 */
@Component({
  selector: 'lu-report-exercise',
  imports: [ReactiveFormsModule],
  templateUrl: './report-exercise.html',
  styleUrl: './report-exercise.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ReportExercise {
  private readonly api = inject(GrammarApiService);

  readonly pointId = input.required<string>();
  readonly exerciseId = input.required<string>();

  protected readonly reasons = REPORT_REASONS;
  protected readonly maxNote = MAX_REPORT_NOTE;

  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');
  private readonly trigger = viewChild.required<ElementRef<HTMLButtonElement>>('trigger');

  protected readonly form = new FormGroup({
    reason: new FormControl<ReportReason | null>(null, [Validators.required]),
    note: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_REPORT_NOTE)] }),
  });
  private readonly value = toSignal(this.form.valueChanges, { initialValue: this.form.value });

  protected readonly sending = signal(false);
  protected readonly failed = signal(false);
  protected readonly submitted = signal(false);
  protected readonly noteLength = computed(() => (this.value().note ?? '').length);
  protected readonly reasonMissing = computed(() => this.submitted() && !this.value().reason);
  protected readonly reported = computed(() => this.api.reported().has(`${this.pointId()}/${this.exerciseId()}`));

  protected open(): void {
    if (this.reported()) {
      return;
    }
    this.submitted.set(false);
    this.failed.set(false);
    const el = this.dialog().nativeElement;
    if (!el.open) {
      el.showModal();
    }
  }

  protected cancel(): void {
    if (!this.sending()) {
      this.close();
    }
  }

  /** Escape: closes the dialog unless a send is running. */
  protected onCancel(event: Event): void {
    if (this.sending()) {
      event.preventDefault();
    }
  }

  /** Fires for every way the dialog closes; hands the focus back to the button. */
  protected onClosed(): void {
    this.trigger().nativeElement.focus();
  }

  protected submit(): void {
    this.submitted.set(true);
    const { reason, note } = this.form.getRawValue();
    if (!reason || this.form.controls.note.invalid || this.sending()) {
      return;
    }
    this.sending.set(true);
    this.failed.set(false);
    const pointId = this.pointId();
    const exerciseId = this.exerciseId();
    this.api.reportExercise(pointId, { exerciseId, reason, note: note.trim() }).subscribe({
      next: () => {
        this.sending.set(false);
        this.api.markReported(pointId, exerciseId);
        this.form.reset({ reason: null, note: '' });
        this.close();
      },
      error: () => {
        this.sending.set(false);
        this.failed.set(true);
      },
    });
  }

  private close(): void {
    const el = this.dialog().nativeElement;
    if (el.open) {
      el.close();
    }
  }
}
