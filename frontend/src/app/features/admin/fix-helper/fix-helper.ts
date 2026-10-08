import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';
import { FormArray, FormControl, NonNullableFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { FixSuggestion, Lesson, LessonFlag } from '../../../core/models/lesson';
import { Icon } from '../../../shared/components/icon/icon';
import { AdminApiService } from '../admin-api.service';
import { lessonCheckFailure } from '../lesson-check-errors';

/**
 * "AI gợi ý sửa" under one flag of the AI check (F22): asks the AI for a corrected version of the
 * flagged item, lets the admin edit it, then applies it. The page gets the saved lesson.
 */
@Component({
  selector: 'lu-fix-helper',
  imports: [ReactiveFormsModule, Icon],
  templateUrl: './fix-helper.html',
  styleUrl: './fix-helper.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FixHelper {
  private readonly api = inject(AdminApiService);
  private readonly fb = inject(NonNullableFormBuilder);

  readonly lessonId = input.required<string>();
  readonly flag = input.required<LessonFlag>();
  /** Name of the flagged spot for screen readers, e.g. "câu hỏi 2". */
  readonly subject = input('chỗ này');
  readonly disabled = input(false);
  /** The lesson after the correction was saved. */
  readonly applied = output<Lesson>();

  protected readonly asking = signal(false);
  protected readonly applying = signal(false);
  protected readonly suggestion = signal<FixSuggestion | null>(null);
  protected readonly error = signal<string | null>(null);

  protected readonly form = this.fb.group({
    text: '',
    meaningVi: '',
    prompt: '',
    options: this.fb.array<FormControl<string>>([]),
    answerIndex: 0,
    explanationVi: '',
    vi: '',
    en: '',
  });

  protected get options(): FormArray<FormControl<string>> {
    return this.form.controls.options;
  }

  protected id(part: string): string {
    const f = this.flag();
    return `fix-${f.area}-${f.index}-${part}`;
  }

  protected async ask(): Promise<void> {
    if (this.asking() || this.applying()) {
      return;
    }
    const f = this.flag();
    this.asking.set(true);
    this.error.set(null);
    try {
      const s = await firstValueFrom(this.api.suggestFix(this.lessonId(), f.area, f.index));
      const q = s.question;
      this.options.clear();
      for (const o of q?.options ?? []) {
        this.options.push(this.fb.control(o));
      }
      this.form.patchValue({
        text: s.text ?? '',
        meaningVi: s.meaningVi ?? '',
        prompt: q?.prompt ?? '',
        answerIndex: q?.answerIndex ?? 0,
        explanationVi: q?.explanationVi ?? '',
        vi: s.vi ?? '',
        en: s.en ?? '',
      });
      this.suggestion.set(s);
    } catch (err) {
      this.error.set(lessonCheckFailure(err, 'AI chưa gợi ý được, vui lòng thử lại.', true));
    } finally {
      this.asking.set(false);
    }
  }

  protected dismiss(): void {
    this.suggestion.set(null);
    this.error.set(null);
  }

  protected async apply(): Promise<void> {
    const s = this.suggestion();
    if (!s || this.applying()) {
      return;
    }
    const v = this.form.getRawValue();
    const fix: FixSuggestion = { area: s.area, index: s.index };
    switch (s.area) {
      case 'sentence':
        fix.text = v.text;
        break;
      case 'annotation':
        fix.meaningVi = v.meaningVi;
        break;
      case 'question':
        fix.question = { prompt: v.prompt, options: v.options, answerIndex: v.answerIndex, explanationVi: v.explanationVi };
        break;
      case 'translation':
        fix.vi = v.vi;
        fix.en = v.en;
        break;
    }
    this.applying.set(true);
    this.error.set(null);
    try {
      const lesson = await firstValueFrom(this.api.applyFix(this.lessonId(), fix));
      this.suggestion.set(null);
      this.applied.emit(lesson);
    } catch (err) {
      const fields = err instanceof ApiError ? (err.body as { fields?: Record<string, string> } | null)?.fields : null;
      this.error.set(
        fields ? Object.values(fields).join(' ') : lessonCheckFailure(err, 'Không áp dụng được, vui lòng thử lại.'),
      );
    } finally {
      this.applying.set(false);
    }
  }
}
