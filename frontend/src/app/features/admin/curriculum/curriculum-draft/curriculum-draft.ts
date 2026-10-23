import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import {
  CurriculumBoard,
  CurriculumPlan,
  CurriculumPlanSlot,
  CurriculumPlanTopic,
  MAIN_PER_TOPIC,
  MAX_EXTRA,
  planOf,
} from '../../../../core/models/curriculum';
import { ConfirmDialog } from '../../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../../admin-api.service';

const SAVE_FAILED = 'Không lưu được bản nháp, vui lòng thử lại.';
const WRITE_FAILED = 'AI soạn bài thất bại, vui lòng thử lại.';
const ACTION_FAILED = 'Có lỗi xảy ra, vui lòng thử lại.';

/** What the admin asked for, waiting for the confirmation. */
type Pending = 'approve' | 'discard' | 'redraft';

/** What is running: a save, an approval, or the AI writing topics one after another. */
interface Work {
  kind: 'save' | 'approve' | 'discard' | 'redraft' | 'write';
  /** For 'write': the topic being written, the topics done so far and in all. */
  topic?: number;
  done?: number;
  total?: number;
}

/**
 * The draft of a level's course (added 2026-10-23), written by the AI and edited here: the topics in
 * order with what the learner can do after each and the grammar it teaches, and each topic's
 * lessons (the first 3 are the main ones). Edits stay on the page until "Lưu bản nháp"; asking the
 * AI or approving saves them first. Every change from the server comes out through `changed`.
 */
@Component({
  selector: 'lu-curriculum-draft',
  imports: [ConfirmDialog],
  templateUrl: './curriculum-draft.html',
  styleUrl: './curriculum-draft.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CurriculumDraft {
  private readonly api = inject(AdminApiService);

  readonly board = input.required<CurriculumBoard>();
  /** Names of the catalogue's topics, suggested when naming a topic. */
  readonly topicNames = input<readonly string[]>([]);
  readonly changed = output<CurriculumBoard>();

  protected readonly plan = signal<CurriculumPlan>({ topics: [] });
  protected readonly dirty = signal(false);
  protected readonly work = signal<Work | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly note = signal<string | null>(null);
  protected readonly pending = signal<Pending | null>(null);

  protected readonly mainPerTopic = MAIN_PER_TOPIC;
  protected readonly maxExtra = MAX_EXTRA;
  protected readonly level = computed(() => this.board().level);
  protected readonly points = computed(() => this.board().points);
  protected readonly busy = computed(() => this.work() !== null);
  /** Numbers of the topics without lessons, which the AI should write. */
  protected readonly unwrittenNumbers = computed(() =>
    this.plan()
      .topics.map((t, i) => (t.slots.length === 0 ? i + 1 : 0))
      .filter((n) => n > 0),
  );
  protected readonly lessonCount = computed(() =>
    this.plan().topics.reduce((n, t) => n + t.slots.length, 0),
  );
  /**
   * Per topic of the plan, the titles of the lessons that filled a slot when the draft was read
   * and that no lesson of the edited topic is named after any more: once approved they are only
   * "bài khác" of the topic (lessons fill slots by title).
   */
  protected readonly unmatched = computed(() => {
    const filled = new Map<string, string[]>();
    for (const t of this.board().topics) {
      const titles = t.slots.filter((s) => s.lesson).map((s) => s.title);
      filled.set(t.topicId || nameKey(t.name), titles);
    }
    return this.plan().topics.map((t) => {
      const titles = new Set(t.slots.map((s) => nameKey(s.title)));
      return (filled.get(t.topicId || nameKey(t.name)) ?? []).filter(
        (x) => !titles.has(nameKey(x)),
      );
    });
  });
  protected readonly workText = computed(() => {
    const w = this.work();
    switch (w?.kind) {
      case 'save':
        return 'Đang lưu bản nháp…';
      case 'approve':
        return 'Đang duyệt giáo trình…';
      case 'discard':
        return 'Đang bỏ bản nháp…';
      case 'redraft':
        return 'AI đang soạn danh sách chủ đề… Có thể mất tới 2 phút.';
      case 'write':
        return w.total && w.total > 1
          ? `AI đang soạn bài cho chủ đề ${w.topic} (${(w.done ?? 0) + 1}/${w.total})… Mỗi chủ đề mất tới khoảng 1 phút.`
          : `AI đang soạn bài cho chủ đề ${w.topic}… Có thể mất tới 1 phút.`;
      default:
        return '';
    }
  });
  protected readonly confirmTitle = computed(() => {
    switch (this.pending()) {
      case 'approve':
        return `Duyệt giáo trình ${this.level()}?`;
      case 'discard':
        return 'Bỏ bản nháp?';
      case 'redraft':
        return 'Soạn lại toàn bộ bằng AI?';
      default:
        return '';
    }
  });
  protected readonly confirmMessage = computed(() => {
    switch (this.pending()) {
      case 'approve':
        return 'Người học sẽ thấy giáo trình này; chủ đề chưa có trong danh mục được tạo. Thứ tự chủ đề và lộ trình chỉ đổi khi bạn bấm “Áp dụng thứ tự”.';
      case 'discard':
        return this.board().hasApproved
          ? 'Mọi thay đổi trong bản nháp sẽ mất, giáo trình đã duyệt giữ nguyên.'
          : 'Bản nháp do AI soạn và mọi thay đổi sẽ mất.';
      case 'redraft':
        return 'AI soạn lại danh sách chủ đề từ đầu; các chủ đề và bài trong bản nháp hiện tại sẽ mất.';
      default:
        return '';
    }
  });

  constructor() {
    // A board from the server replaces the edited plan.
    effect(() => {
      this.plan.set(planOf(this.board()));
      this.dirty.set(false);
    });
  }

  protected kindLabel(slotIndex: number): string {
    const extra = slotIndex >= MAIN_PER_TOPIC;
    const n = extra ? slotIndex - MAIN_PER_TOPIC + 1 : slotIndex + 1;
    return `${extra ? 'Bài thêm' : 'Bài chính'} ${n}`;
  }

  protected extraCount(t: CurriculumPlanTopic): number {
    return Math.max(0, t.slots.length - MAIN_PER_TOPIC);
  }

  protected pointTitle(id: string): string {
    return this.points().find((p) => p.id === id)?.titleVi ?? id;
  }

  // --- edits, kept on the page until saved ---

  protected setTopic(index: number, patch: Partial<CurriculumPlanTopic>): void {
    this.edit((topics) => topics.map((t, i) => (i === index ? { ...t, ...patch } : t)));
  }

  /** A new name drops the catalogue's topic: the server finds it again by name on approval. */
  protected rename(index: number, name: string): void {
    this.setTopic(index, { name, topicId: '' });
  }

  protected toggleGrammar(index: number, id: string, on: boolean): void {
    const t = this.plan().topics[index];
    const grammar = on ? [...t.grammar, id] : t.grammar.filter((g) => g !== id);
    this.setTopic(index, {
      grammar: this.points()
        .map((p) => p.id)
        .filter((p) => grammar.includes(p)),
    });
  }

  protected moveTopic(index: number, step: -1 | 1): void {
    this.edit((topics) => move(topics, index, step));
    focusLater(`draft-topic-${index + step}`);
  }

  protected removeTopic(index: number): void {
    this.edit((topics) => topics.filter((_, i) => i !== index));
    this.note.set('Đã bỏ chủ đề khỏi bản nháp (chưa lưu).');
  }

  protected addTopic(): void {
    this.edit((topics) => [
      ...topics,
      { topicId: '', name: '', canDoVi: '', grammar: [], slots: [] },
    ]);
    focusLater(`draft-topic-name-${this.plan().topics.length - 1}`);
  }

  protected setSlot(index: number, slotIndex: number, patch: Partial<CurriculumPlanSlot>): void {
    const t = this.plan().topics[index];
    this.setTopic(index, {
      slots: t.slots.map((s, j) => (j === slotIndex ? { ...s, ...patch } : s)),
    });
  }

  /** The lesson's main point; its other points stay after it. */
  protected setSlotPoint(index: number, slotIndex: number, id: string): void {
    const s = this.plan().topics[index].slots[slotIndex];
    this.setSlot(index, slotIndex, {
      grammar: [id, ...s.grammar.slice(1).filter((g) => g !== id)],
    });
  }

  protected moveSlot(index: number, slotIndex: number, step: -1 | 1): void {
    const t = this.plan().topics[index];
    this.setTopic(index, { slots: move(t.slots, slotIndex, step) });
    focusLater(`draft-slot-${index}-${slotIndex + step}`);
  }

  protected removeSlot(index: number, slotIndex: number): void {
    const t = this.plan().topics[index];
    this.setTopic(index, { slots: t.slots.filter((_, j) => j !== slotIndex) });
  }

  protected addSlot(index: number): void {
    const t = this.plan().topics[index];
    const grammar = t.grammar.length > 0 ? [t.grammar[0]] : [];
    this.setTopic(index, { slots: [...t.slots, { title: '', idea: '', grammar }] });
    focusLater(`draft-slot-title-${index}-${t.slots.length}`);
  }

  // --- server actions ---

  protected async save(): Promise<void> {
    if (
      await this.run(
        { kind: 'save' },
        () => this.api.saveCurriculumDraft(this.level(), this.plan()),
        SAVE_FAILED,
      )
    ) {
      this.note.set('Đã lưu bản nháp.');
    }
  }

  /** Writes the lessons of one topic, or of every topic without lessons, one request each. */
  protected async write(numbers: number[], extraOnly = false): Promise<void> {
    if (this.busy() || !(await this.saveFirst())) {
      return;
    }
    let done = 0;
    for (const n of numbers) {
      const ok = await this.run(
        { kind: 'write', topic: n, done, total: numbers.length },
        () => this.api.writeCurriculumTopic(this.level(), n, extraOnly),
        WRITE_FAILED,
        numbers.length > 1 ? `Dừng ở chủ đề ${n}: ` : '',
      );
      if (!ok) {
        return;
      }
      done++;
    }
    this.note.set(
      extraOnly
        ? 'AI đã soạn thêm bài thêm cho chủ đề.'
        : done > 1
          ? `AI đã soạn bài cho ${done} chủ đề.`
          : 'AI đã soạn bài cho chủ đề.',
    );
  }

  protected ask(p: Pending): void {
    this.pending.set(p);
  }

  protected async confirm(): Promise<void> {
    const p = this.pending();
    this.pending.set(null);
    switch (p) {
      case 'approve':
        if (await this.saveFirst()) {
          await this.run(
            { kind: 'approve' },
            () => this.api.approveCurriculum(this.level()),
            ACTION_FAILED,
          );
        }
        return;
      case 'discard':
        await this.run(
          { kind: 'discard' },
          () => this.api.discardCurriculumDraft(this.level()),
          ACTION_FAILED,
        );
        return;
      case 'redraft':
        if (
          await this.run(
            { kind: 'redraft' },
            () => this.api.draftCurriculum(this.level()),
            ACTION_FAILED,
          )
        ) {
          this.note.set(
            'AI đã soạn lại danh sách chủ đề. Bấm “Soạn bài cho các chủ đề” để có bài.',
          );
        }
        return;
    }
  }

  private edit(f: (topics: CurriculumPlanTopic[]) => CurriculumPlanTopic[]): void {
    this.plan.update((p) => ({ topics: f(p.topics) }));
    this.dirty.set(true);
    this.note.set(null);
  }

  private saveFirst(): Promise<boolean> {
    if (!this.dirty()) {
      return Promise.resolve(true);
    }
    return this.run(
      { kind: 'save' },
      () => this.api.saveCurriculumDraft(this.level(), this.plan()),
      SAVE_FAILED,
    );
  }

  /** One request; its board goes out through `changed`. */
  private async run(
    work: Work,
    call: () => ReturnType<AdminApiService['saveCurriculumDraft']>,
    fallback: string,
    prefix = '',
  ): Promise<boolean> {
    this.work.set(work);
    this.error.set(null);
    this.note.set(null);
    try {
      this.changed.emit(await firstValueFrom(call()));
      return true;
    } catch (err) {
      this.error.set(prefix + (messageOf(err) ?? fallback));
      return false;
    } finally {
      this.work.set(null);
    }
  }
}

function move<T>(list: readonly T[], index: number, step: -1 | 1): T[] {
  const to = index + step;
  if (to < 0 || to >= list.length) {
    return [...list];
  }
  const out = [...list];
  [out[index], out[to]] = [out[to], out[index]];
  return out;
}

/** Keeps the focus on what moved or was added, once it is drawn. */
function focusLater(id: string): void {
  setTimeout(() => document.getElementById(id)?.focus());
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const message = (err.body as { message?: unknown } | null)?.message;
    return typeof message === 'string' && message ? message : null;
  }
  return null;
}

/** Compares titles and names as the server does: case, spaces and the kind of apostrophe. */
function nameKey(s: string): string {
  return s.replace(/’/g, "'").trim().split(/\s+/).join(' ').toLowerCase();
}
