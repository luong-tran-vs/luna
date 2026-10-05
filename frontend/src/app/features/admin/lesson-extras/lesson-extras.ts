import { ChangeDetectionStrategy, Component, effect, inject, input, output, signal, untracked } from '@angular/core';
import { FormArray, FormControl, FormGroup, NonNullableFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { ExtrasInput, Lesson, LessonFlag, Question } from '../../../core/models/lesson';
import { AdminApiService } from '../admin-api.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { FlagNote } from '../flag-note/flag-note';

const MAX_QUESTIONS = 5;
const MAX_EXAMPLES = 3;

type QuestionForm = FormGroup<{
  prompt: FormControl<string>;
  options: FormArray<FormControl<string>>;
  answerIndex: FormControl<number>;
  explanationVi: FormControl<string>;
}>;

/**
 * Editor for a lesson's comprehension questions, grammar note and writing prompt (F15).
 * The server checks everything (examples must appear in the lesson) and reports by field.
 */
@Component({
  selector: 'lu-lesson-extras',
  imports: [ReactiveFormsModule, FlagNote, ConfirmDialog],
  templateUrl: './lesson-extras.html',
  styleUrl: './lesson-extras.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonExtras {
  private readonly api = inject(AdminApiService);
  private readonly fb = inject(NonNullableFormBuilder);

  readonly lesson = input.required<Lesson>();
  readonly saved = output<Lesson>();
  /** F22: flags on the questions (index = question position); empty when not checked. */
  readonly flags = input<LessonFlag[]>([]);
  readonly flagBusy = input(false);
  readonly confirm = output<LessonFlag>();

  protected readonly maxQuestions = MAX_QUESTIONS;
  protected readonly maxExamples = MAX_EXAMPLES;
  protected readonly optionIndexes = [0, 1, 2, 3];

  protected readonly questions = new FormArray<QuestionForm>([]);
  protected readonly hasGrammar = new FormControl(false, { nonNullable: true });
  protected readonly grammar = this.fb.group({
    title: [''],
    bodyVi: [''],
    examples: this.fb.array<FormControl<string>>([]),
  });
  protected readonly writingPrompt = new FormControl('', { nonNullable: true });

  protected readonly deleteAt = signal<number | null>(null);
  protected readonly saving = signal(false);
  protected readonly status = signal<'idle' | 'saved'>('idle');
  protected readonly error = signal<string | null>(null);
  protected readonly fields = signal<Record<string, string>>({});

  constructor() {
    // The form follows the lesson: first load and every save.
    effect(() => {
      const l = this.lesson();
      untracked(() => this.reset(l));
    });
  }

  private reset(l: Lesson): void {
    this.questions.clear();
    for (const q of l.questions) {
      this.questions.push(this.question(q));
    }
    this.hasGrammar.setValue(!!l.grammarNote);
    this.grammar.controls.examples.clear();
    this.grammar.patchValue({ title: l.grammarNote?.title ?? '', bodyVi: l.grammarNote?.bodyVi ?? '' });
    for (const e of l.grammarNote?.examples ?? ['']) {
      this.grammar.controls.examples.push(this.fb.control(e));
    }
    this.writingPrompt.setValue(l.writingPrompt);
  }

  private question(q?: Question): QuestionForm {
    return this.fb.group({
      prompt: [q?.prompt ?? ''],
      options: this.fb.array((q?.options ?? ['', '', '', '']).map((o) => this.fb.control(o))),
      answerIndex: [q?.answerIndex ?? -1],
      explanationVi: [q?.explanationVi ?? ''],
    });
  }

  protected flagFor(i: number): LessonFlag | null {
    return this.flags().find((f) => f.index === i) ?? null;
  }

  /** Puts the cursor in the question's text box. */
  protected editQuestion(index: number): void {
    document.getElementById('q-' + index + '-prompt')?.focus();
  }

  /** Removes the question from the form (unsaved changes included) and saves the rest. */
  protected async confirmDeleteQuestion(): Promise<void> {
    const index = this.deleteAt();
    this.deleteAt.set(null);
    if (index === null || index >= this.questions.length) {
      return;
    }
    this.removeQuestion(index);
    await this.save();
  }

  protected err(key: string): string | null {
    return this.fields()[key] ?? null;
  }

  protected describedBy(key: string): string | null {
    return this.err(key) ? `extras-${key.replace(/\./g, '-')}-error` : null;
  }

  protected errorId(key: string): string {
    return `extras-${key.replace(/\./g, '-')}-error`;
  }

  protected addQuestion(): void {
    if (this.questions.length < MAX_QUESTIONS) {
      this.questions.push(this.question());
    }
  }

  protected removeQuestion(index: number): void {
    this.questions.removeAt(index);
  }

  protected addExample(): void {
    if (this.grammar.controls.examples.length < MAX_EXAMPLES) {
      this.grammar.controls.examples.push(this.fb.control(''));
    }
  }

  protected removeExample(index: number): void {
    this.grammar.controls.examples.removeAt(index);
  }

  private input(): ExtrasInput {
    const g = this.grammar.getRawValue();
    return {
      questions: this.questions.getRawValue(),
      grammarNote: this.hasGrammar.value ? { title: g.title, bodyVi: g.bodyVi, examples: g.examples } : null,
      writingPrompt: this.writingPrompt.value,
    };
  }

  protected async save(): Promise<void> {
    if (this.saving() || this.lesson().annotationStatus === 'running') {
      return;
    }
    this.saving.set(true);
    this.status.set('idle');
    this.error.set(null);
    this.fields.set({});
    try {
      const lesson = await firstValueFrom(this.api.updateExtras(this.lesson().id, this.input()));
      this.status.set('saved');
      this.saved.emit(lesson);
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
      if (body?.fields) {
        this.fields.set(body.fields);
        this.error.set('Vui lòng sửa các ô được đánh dấu.');
      } else {
        this.error.set(body?.message ?? 'Không lưu được, vui lòng thử lại.');
      }
    } finally {
      this.saving.set(false);
    }
  }
}
