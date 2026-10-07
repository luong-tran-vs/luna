import { ChangeDetectionStrategy, Component, computed, inject, input, linkedSignal, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { Quiz, QuizAnswer } from '../../../../core/models/reading';
import { Icon } from '../../../../shared/components/icon/icon';
import { ReadingApiService } from '../../reading-api.service';

export interface QuizScore {
  correct: number;
  total: number;
}

const SEND_FAILED = 'Không gửi được câu trả lời, vui lòng thử lại.';

/**
 * Comprehension questions of the Reading step (F15): one question on screen at a time, with ← →
 * to move between them; pick an option then Kiểm tra; the result (text and symbol, not only
 * colour) shows at once, no second answer. Answers come from the server, so reloading resumes at
 * the first unanswered question.
 */
@Component({
  selector: 'lu-comprehension-quiz',
  imports: [Icon],
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
  /** The option picked for the current question, checked with Kiểm tra (design screen 6). */
  protected readonly picked = signal<number | null>(null);
  protected readonly letters = ['A', 'B', 'C', 'D', 'E', 'F'];

  protected readonly byIndex = computed(() => new Map(this.answers().map((a) => [a.questionIndex, a])));
  protected readonly total = computed(() => this.quiz().questions.length);
  /** First unanswered question, or -1 when every question is answered. */
  protected readonly current = computed(() => {
    const answered = this.byIndex();
    return this.quiz().questions.findIndex((_, i) => !answered.has(i));
  });
  /**
   * The question on screen: one at a time; it opens on the first one not answered, and stays put
   * after an answer so its result shows.
   */
  protected readonly shown = linkedSignal({
    source: this.quiz,
    computation: (q) => Math.max(0, q.questions.findIndex((_, i) => !q.answers.some((a) => a.questionIndex === i))),
  });
  /** The next question not answered after the one on screen (wrapping round), or -1. */
  protected readonly nextOpen = computed(() => {
    const n = this.total();
    const answered = this.byIndex();
    for (let k = 1; k < n; k++) {
      const i = (this.shown() + k) % n;
      if (!answered.has(i)) {
        return i;
      }
    }
    return -1;
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

  /** Trả lời lại: the question on screen opens again; the new answer replaces the stored one. */
  protected redo(index: number): void {
    this.picked.set(null);
    this.answers.update((list) => list.filter((a) => a.questionIndex !== index));
  }

  /** Làm lại tất cả: every answer is forgotten on the server too, back to the first question. */
  protected async restart(): Promise<void> {
    if (this.sending()) {
      return;
    }
    this.sending.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(this.api.resetAnswers(this.lessonId()));
      this.answers.set([]);
      this.picked.set(null);
      this.shown.set(0);
      this.announcement.set('Đã xoá các câu trả lời, làm lại từ câu 1.');
    } catch {
      this.error.set('Chưa làm lại được, vui lòng thử lại.');
    } finally {
      this.sending.set(false);
    }
  }

  protected show(index: number): void {
    if (index >= 0 && index < this.total()) {
      this.picked.set(null);
      this.shown.set(index);
    }
  }

  private record(answer: QuizAnswer): void {
    this.picked.set(null);
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
