import { ChangeDetectionStrategy, Component, computed, inject, input, linkedSignal, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { Quiz, QuizAnswer } from '../../../../core/models/reading';
import { ReadingApiService } from '../../reading-api.service';

export interface QuizScore {
  correct: number;
  total: number;
}

const SEND_FAILED = 'Không gửi được câu trả lời, vui lòng thử lại.';

/**
 * Comprehension questions of the Reading step (F15): one question at a time, the result
 * (text and symbol, not only colour) right after choosing, no second answer. Answers come
 * from the server, so reloading resumes at the first unanswered question.
 */
@Component({
  selector: 'lu-comprehension-quiz',
  templateUrl: './comprehension-quiz.html',
  styleUrl: './comprehension-quiz.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ComprehensionQuiz {
  private readonly api = inject(ReadingApiService);

  readonly lessonId = input.required<string>();
  readonly quiz = input.required<Quiz>();
  /** Review mode never reports completion. */
  readonly review = input(false);

  /** Emitted once, when the last question gets answered in this visit. */
  readonly allAnswered = output<QuizScore>();
  /** The questions changed on the server; the page should load them again. */
  readonly reload = output<void>();

  protected readonly answers = linkedSignal(() => [...this.quiz().answers]);
  protected readonly sending = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly changed = signal(false);
  protected readonly announcement = signal('');

  protected readonly byIndex = computed(() => new Map(this.answers().map((a) => [a.questionIndex, a])));
  protected readonly total = computed(() => this.quiz().questions.length);
  /** First unanswered question, or -1 when every question is answered. */
  protected readonly current = computed(() => {
    const answered = this.byIndex();
    return this.quiz().questions.findIndex((_, i) => !answered.has(i));
  });
  protected readonly correct = computed(() => this.answers().filter((a) => a.correct).length);

  protected async choose(questionIndex: number, choice: number): Promise<void> {
    if (this.sending() || this.byIndex().has(questionIndex)) {
      return;
    }
    this.sending.set(true);
    this.error.set(null);
    try {
      const res = await firstValueFrom(
        this.api.answer(this.lessonId(), { version: this.quiz().version, questionIndex, choice }),
      );
      this.record(res.answer);
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { error?: string; answer?: QuizAnswer } | null) : null;
      if (body?.error === 'already_answered' && body.answer) {
        this.record(body.answer);
      } else if (body?.error === 'quiz_changed') {
        this.changed.set(true);
      } else {
        this.error.set(SEND_FAILED);
      }
    } finally {
      this.sending.set(false);
    }
  }

  private record(answer: QuizAnswer): void {
    this.answers.update((list) =>
      [...list.filter((a) => a.questionIndex !== answer.questionIndex), answer].sort(
        (a, b) => a.questionIndex - b.questionIndex,
      ),
    );
    const done = this.current() === -1;
    this.announcement.set(
      `Câu ${answer.questionIndex + 1}: ${answer.correct ? 'Đúng' : 'Sai'}.` +
        (done ? ` Bạn trả lời đúng ${this.correct()}/${this.total()} câu.` : ''),
    );
    if (done && !this.review()) {
      this.allAnswered.emit({ correct: this.correct(), total: this.total() });
    }
  }
}
