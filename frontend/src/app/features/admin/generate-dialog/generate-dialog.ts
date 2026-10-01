import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';

import {
  GenerateInput,
  LessonKind,
  MAX_COUNT,
  MAX_IDEA,
  MAX_WORDS,
  MIN_COUNT,
  MIN_WORDS,
  wordRange,
} from '../../../core/models/generate';

const INTEGER = /^\d+$/;

/**
 * "Sinh bài bằng AI" dialog (F7). A native <dialog> opened with showModal(), so focus stays
 * inside and Escape closes it (except while generating). The parent owns the request; the
 * dialog only validates and emits.
 */
@Component({
  selector: 'lu-generate-dialog',
  imports: [ReactiveFormsModule],
  templateUrl: './generate-dialog.html',
  styleUrl: './generate-dialog.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GenerateDialog {
  readonly open = input(false);
  /** "A1 · Gia đình". */
  readonly topicLabel = input.required<string>();
  /** Values shown when the dialog opens (the last ones used). */
  readonly options = input.required<GenerateInput>();
  readonly busy = input(false);
  readonly error = input<string | null>(null);

  readonly generate = output<GenerateInput>();
  readonly closed = output<void>();

  protected readonly maxIdea = MAX_IDEA;
  protected readonly minCount = MIN_COUNT;
  protected readonly maxCount = MAX_COUNT;
  protected readonly minWords = MIN_WORDS;
  protected readonly maxWords = MAX_WORDS;

  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');
  private readonly submitted = signal(false);

  protected readonly form = new FormGroup({
    count: new FormControl<number | null>(null, [
      Validators.required,
      Validators.min(MIN_COUNT),
      Validators.max(MAX_COUNT),
      Validators.pattern(INTEGER),
    ]),
    words: new FormControl<number | null>(null, [
      Validators.required,
      Validators.min(MIN_WORDS),
      Validators.max(MAX_WORDS),
      Validators.pattern(INTEGER),
    ]),
    kind: new FormControl<LessonKind>('reading', { nonNullable: true }),
    idea: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_IDEA)] }),
  });

  private readonly value = toSignal(this.form.valueChanges, { initialValue: this.form.value });
  private readonly status = toSignal(this.form.statusChanges, { initialValue: this.form.status });

  protected readonly ideaLength = computed(() => (this.value().idea ?? '').length);
  protected readonly range = computed(() => {
    const words = Number(this.value().words);
    return Number.isInteger(words) && words >= MIN_WORDS && words <= MAX_WORDS ? wordRange(words) : null;
  });

  protected readonly errors = computed(() => {
    this.status();
    if (!this.submitted()) {
      return {} as Record<string, string>;
    }
    const c = this.form.controls;
    const out: Record<string, string> = {};
    if (c.count.invalid) {
      out['count'] = `Số bài từ ${MIN_COUNT} đến ${MAX_COUNT}`;
    }
    if (c.words.invalid) {
      out['words'] = `Độ dài từ ${MIN_WORDS} đến ${MAX_WORDS} từ`;
    }
    if (c.idea.invalid) {
      out['idea'] = `Ý chính tối đa ${MAX_IDEA} ký tự`;
    }
    return out;
  });

  constructor() {
    // Opening resets the form to the given options; closing closes the native dialog.
    effect(() => {
      const el = this.dialog().nativeElement;
      if (this.open()) {
        untracked(() => {
          this.submitted.set(false);
          this.form.reset(this.options());
        });
        if (!el.open) {
          el.showModal();
        }
      } else if (el.open) {
        el.close();
      }
    });
    effect(() => {
      if (this.busy()) {
        this.form.disable();
      } else {
        this.form.enable();
      }
    });
  }

  protected describedBy(field: string, hint?: string): string | null {
    const ids = [hint, this.errors()[field] ? `generate-${field}-error` : undefined].filter(Boolean);
    return ids.length ? ids.join(' ') : null;
  }

  protected submit(): void {
    this.submitted.set(true);
    this.form.updateValueAndValidity();
    if (this.form.invalid || this.busy()) {
      return;
    }
    const v = this.form.getRawValue();
    this.generate.emit({ count: Number(v.count), words: Number(v.words), kind: v.kind, idea: v.idea.trim() });
  }

  /** Escape: the parent decides; never while generating. */
  protected onCancel(event: Event): void {
    event.preventDefault();
    if (!this.busy()) {
      this.closed.emit();
    }
  }

  protected cancel(): void {
    if (!this.busy()) {
      this.closed.emit();
    }
  }
}
