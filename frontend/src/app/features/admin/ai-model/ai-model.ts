import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { AiModelOutcome, AiModelState, AiModelTest } from '../../../core/models/ai-model';
import { AdminApiService } from '../admin-api.service';
import { opLabel } from '../ai-usage/ai-usage-catalog';

/** What the admin is told of each model, from tests made on 2026-10-23. */
const MODEL_INFO: Readonly<Record<string, { name: string; note: string }>> = {
  'gemini-3.5-flash-lite': {
    name: 'Gemini 3.5 Flash Lite',
    note: 'Nhanh và ổn định nhất (1–3 giây), nhưng viết kém tự nhiên nhất.',
  },
  'gemini-3.5-flash': {
    name: 'Gemini 3.5 Flash',
    note: 'Thông minh hơn bản Lite, khoảng 5 giây; đôi khi quá tải.',
  },
  'gemini-3.6-flash': {
    name: 'Gemini 3.6 Flash',
    note: 'Viết tốt nhất trong các model đủ nhanh (5–10 giây), nhưng hay quá tải.',
  },
  'gemini-3.7-flash': {
    name: 'Gemini 3.7 Flash',
    note: 'Rất chậm khi thử (trên 2 phút), dễ hết thời gian chờ.',
  },
  'gemini-3.8-flash': {
    name: 'Gemini 3.8 Flash',
    note: 'Rất chậm khi thử (30–100 giây), dễ hết thời gian chờ.',
  },
  // Groq (added 2026-10-23): offered when GROQ_API_KEY is set; pictures stay with Gemini.
  'openai/gpt-oss-120b': {
    name: 'GPT-OSS 120B · Groq',
    note: 'Nhanh (1–8 giây), viết tốt và nghĩa tiếng Việt chính xác; đôi khi dùng từ hơi khó so với trình độ.',
  },
  'openai/gpt-oss-20b': {
    name: 'GPT-OSS 20B · Groq',
    note: 'Rất nhanh nhưng hay lạc đề và dịch sai nghĩa khi thử.',
  },
  'qwen/qwen3.8-27b': {
    name: 'Qwen 3.8 27B · Groq',
    note: 'Nhanh nhất (khoảng 1 giây) nhưng nghĩa tiếng Việt kém khi thử.',
  },
  // OpenAI (added 2026-10-26): offered when OPENAI_API_KEY is set; not tested yet.
  'gpt-6-sol': {
    name: 'GPT-6 Sol · ChatGPT',
    note: 'Model mạnh nhất của OpenAI, tính phí theo token; chưa thử trong Luna.',
  },
  'gpt-6-luna': {
    name: 'GPT-6 Luna · ChatGPT',
    note: 'Bản nhẹ và rẻ của GPT-6; chưa thử trong Luna.',
  },
  // Added 2026-10-26 at the admin's request; not tested yet.
  'gpt-6-astra': {
    name: 'GPT-6 Astra · ChatGPT',
    note: 'Thuộc dòng GPT-6, tính phí theo token; chưa thử trong Luna.',
  },
  'gpt-5.5-pro': {
    name: 'GPT-5.5 Pro · ChatGPT',
    note: 'Bản Pro: rất chậm và đắt, có thể không gọi được qua Chat API; chưa thử trong Luna.',
  },
  'gpt-5.5': {
    name: 'GPT-5.5 · ChatGPT',
    note: 'Thế hệ trước GPT-6, tính phí theo token; chưa thử trong Luna.',
  },
  'gpt-5.4-mini': {
    name: 'GPT-5.4 mini · ChatGPT',
    note: 'Bản nhỏ, nhanh và rẻ; chưa thử trong Luna.',
  },
};

const OUTCOME_TEXT: Record<Exclude<AiModelOutcome, 'ok'>, string> = {
  timeout: 'Không trả lời trong 60 giây',
  quota: 'Hết lượt (429)',
  busy: 'Đang quá tải (503)',
  error: 'Lỗi',
};

const dateTimeFormat = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
});

/**
 * The ops a learner waits on, listed first so they are easy to find; it only orders the page, every
 * op's model is the admin's choice.
 */
const LEARNER_OPS: readonly string[] = ['explain', 'talk', 'talk_review', 'speak_check', 'grade'];

/** The Groq model the advice means, offered when GROQ_API_KEY is set. */
export const ADVISED_GROQ = 'openai/gpt-oss-120b';

/**
 * Which provider suits each op, and why, from the tests of 2026-10-23: Groq answers in seconds, so
 * it suits what a learner waits on; Gemini keeps to the CEFR level better, for content written
 * ahead. Only advice shown next to each op: the model stays the admin's choice.
 */
export const OP_ADVICE: Readonly<Record<string, { provider: 'gemini' | 'groq'; why: string }>> = {
  explain: { provider: 'groq', why: 'người học đang chờ câu trả lời' },
  talk: { provider: 'groq', why: 'mỗi lượt hội thoại cần trả lời ngay' },
  talk_review: { provider: 'groq', why: 'người học chờ nhận xét cuối buổi' },
  speak_check: { provider: 'groq', why: 'người học chờ nhận xét bước Nói' },
  grade: { provider: 'groq', why: 'người học chờ điểm bài viết' },
  generate: { provider: 'gemini', why: 'bài cần đúng trình độ, không cần nhanh' },
  annotate: { provider: 'gemini', why: 'chạy nền, cần nghĩa và câu hỏi chính xác' },
  practice: { provider: 'gemini', why: 'chạy nền, nội dung dài' },
  glosses: { provider: 'gemini', why: 'chạy nền, cần nghĩa chính xác' },
  lesson_review: { provider: 'gemini', why: 'kiểm tra bài cần cẩn thận' },
  lesson_fix: { provider: 'gemini', why: 'sửa bài cần đúng trình độ' },
  lesson_simplify: { provider: 'gemini', why: 'chọn từ đúng trình độ' },
  curriculum_draft: { provider: 'gemini', why: 'cần hiểu khung CEFR' },
  curriculum_draft_topic: { provider: 'gemini', why: 'cần hiểu khung CEFR' },
  grammar_lesson: { provider: 'gemini', why: 'bài ngữ pháp cần chính xác' },
  grammar_solve: { provider: 'gemini', why: 'giải bài tập cần chính xác' },
  suggest_words: { provider: 'gemini', why: 'chọn từ đúng trình độ' },
  word_meanings: { provider: 'gemini', why: 'điền nhiều từ một lần, chạy lâu được' },
};

/** The advice of op, in words; "" when there is none. */
export function opAdvice(op: string): string {
  const a = OP_ADVICE[op];
  return a ? `Nên dùng ${a.provider === 'groq' ? 'Groq' : 'Gemini'}: ${a.why}` : '';
}

/** The ops' models the advice gives: Groq's model where advised and offered, the model of every op elsewhere. */
export function advisedOps(
  ops: readonly string[],
  options: readonly string[],
): Record<string, string> {
  const out: Record<string, string> = {};
  if (options.includes(ADVISED_GROQ)) {
    for (const op of ops) {
      if (OP_ADVICE[op]?.provider === 'groq') {
        out[op] = ADVISED_GROQ;
      }
    }
  }
  return out;
}

export interface OpGroup {
  title: string;
  ops: string[];
}

/** The ops in two groups: those a learner waits on, then content writing. */
export function opGroups(ops: readonly string[]): OpGroup[] {
  return [
    { title: 'Người học đang chờ', ops: ops.filter((o) => LEARNER_OPS.includes(o)) },
    { title: 'Soạn nội dung và việc chạy nền', ops: ops.filter((o) => !LEARNER_OPS.includes(o)) },
  ].filter((g) => g.ops.length > 0);
}

function sameOps(
  a: Readonly<Record<string, string>>,
  b: Readonly<Record<string, string>>,
): boolean {
  const keys = Object.keys(a).filter((k) => a[k]);
  return (
    keys.length === Object.keys(b).filter((k) => b[k]).length && keys.every((k) => a[k] === b[k])
  );
}

/** Groq model ids name the maker ("openai/gpt-oss-120b"); Gemini's never have a slash. */
function isGroq(model: string): boolean {
  return model.includes('/');
}

/** OpenAI model ids start with "gpt-" ("gpt-6-sol"). */
function isOpenAI(model: string): boolean {
  return model.startsWith('gpt-');
}

export function modelName(model: string): string {
  return MODEL_INFO[model]?.name ?? model;
}

export function modelNote(model: string): string {
  return MODEL_INFO[model]?.note ?? '';
}

/** What a test said, in words. */
export function testText(t: AiModelTest): string {
  const seconds = (t.durationMs / 1000).toLocaleString('vi-VN', { maximumFractionDigits: 1 });
  if (t.outcome === 'ok') {
    return `Trả lời sau ${seconds} giây`;
  }
  if (t.outcome === 'timeout') {
    return OUTCOME_TEXT.timeout;
  }
  return `${OUTCOME_TEXT[t.outcome]} sau ${seconds} giây`;
}

/**
 * The admin AI model page (added 2026-10-23): picks the model that writes every text, and if wanted
 * a model of its own for each op (what the text is for), instead of changing GEMINI_MODEL and
 * deploying. Each model can be tested first with one short
 * request, since a model may be too slow or overloaded on the day.
 */
@Component({
  selector: 'lu-ai-model',
  templateUrl: './ai-model.html',
  styleUrl: './ai-model.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AiModel {
  private readonly api = inject(AdminApiService);

  protected readonly state = signal<AiModelState | null>(null);
  protected readonly loadError = signal(false);
  /** The model ticked and the ops' own models, saved only with the button. */
  protected readonly selected = signal('');
  protected readonly byOp = signal<Readonly<Record<string, string>>>({});
  protected readonly saving = signal(false);
  protected readonly message = signal('');
  protected readonly saveError = signal(false);
  /** Test results by model, and the models being tested. */
  protected readonly tests = signal<Readonly<Record<string, { text: string; ok: boolean }>>>({});
  protected readonly testing = signal<ReadonlySet<string>>(new Set());

  protected readonly modelName = modelName;
  protected readonly modelNote = modelNote;
  protected readonly opLabel = opLabel;
  protected readonly opAdvice = opAdvice;
  protected readonly ownCount = computed(() => Object.values(this.byOp()).filter((m) => m).length);
  /** Something differs from what is saved. */
  protected readonly dirty = computed(() => {
    const s = this.state();
    return !!s && (this.selected() !== s.model || !sameOps(this.byOp(), s.byOp));
  });

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api.aiModel().subscribe({
      next: (s) => this.show(s),
      error: () => this.loadError.set(true),
    });
  }

  /** The options by provider, with the model in use added first when it is not one of them. */
  protected providerGroups(s: AiModelState): { title: string; models: string[] }[] {
    const all = s.options.includes(s.model) ? s.options : [s.model, ...s.options];
    return [
      { title: 'Gemini', models: all.filter((m) => !isGroq(m) && !isOpenAI(m)) },
      { title: 'Groq', models: all.filter(isGroq) },
      { title: 'ChatGPT', models: all.filter(isOpenAI) },
    ].filter((g) => g.models.length > 0);
  }

  /** How many ops have their own model in what is saved. */
  protected savedOwn(s: AiModelState): number {
    return Object.keys(s.byOp).length;
  }

  protected opGroups(s: AiModelState): OpGroup[] {
    return opGroups(s.ops);
  }

  protected hasOwn(s: AiModelState): boolean {
    return Object.keys(s.byOp).length > 0;
  }

  /** "" puts the op back on the model of every op. */
  protected setOp(op: string, model: string): void {
    this.byOp.update((b) => {
      const next = { ...b };
      if (model) {
        next[op] = model;
      } else {
        delete next[op];
      }
      return next;
    });
  }

  /** Every op as the advice says; saved only with the button. */
  protected applyAdvice(s: AiModelState): void {
    this.byOp.set(advisedOps(s.ops, s.options));
  }

  protected canAdvise(s: AiModelState): boolean {
    return s.options.includes(ADVISED_GROQ);
  }

  protected clearOps(): void {
    this.byOp.set({});
  }

  /** Every op back on GEMINI_MODEL. */
  protected reset(): void {
    this.byOp.set({});
    this.save('');
  }

  protected updatedAt(s: AiModelState): string {
    return s.updatedAt ? dateTimeFormat.format(new Date(s.updatedAt)) : '';
  }

  protected save(model: string): void {
    this.saving.set(true);
    this.saveError.set(false);
    this.message.set('');
    this.api.setAiModel(model, this.byOp()).subscribe({
      next: (s) => {
        this.show(s);
        this.saving.set(false);
        const own = Object.keys(s.byOp).length;
        this.message.set(
          `Đã lưu. Model chung: ${modelName(s.model)}` +
            (own ? `; ${own} việc dùng model riêng.` : '.'),
        );
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set(true);
      },
    });
  }

  protected test(model: string): void {
    this.testing.update((s) => new Set(s).add(model));
    this.api.testAiModel(model).subscribe({
      next: (t) => this.testDone(model, testText(t), t.outcome === 'ok'),
      error: () => this.testDone(model, 'Không thử được, vui lòng thử lại', false),
    });
  }

  private testDone(model: string, text: string, ok: boolean): void {
    this.tests.update((t) => ({ ...t, [model]: { text, ok } }));
    this.testing.update((s) => {
      const next = new Set(s);
      next.delete(model);
      return next;
    });
  }

  private show(s: AiModelState): void {
    // A server not restarted since the ops were added sends neither ops nor byOp.
    s = { ...s, ops: s.ops ?? [], byOp: s.byOp ?? {} };
    this.state.set(s);
    this.selected.set(s.model);
    this.byOp.set({ ...s.byOp });
  }
}
