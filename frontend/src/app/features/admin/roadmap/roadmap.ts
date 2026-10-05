import { CdkDrag, CdkDragDrop, CdkDragHandle, CdkDropList, moveItemInArray } from '@angular/cdk/drag-drop';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { CanLeave } from '../../../core/guards/unsaved-changes.guard';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import {
  AI_LICENSE,
  AI_SOURCE,
  DEFAULT_COUNT,
  DEFAULT_TARGET_WORDS,
  DEFAULT_WORDS,
  GenerateInput,
  GenerateResult,
} from '../../../core/models/generate';
import { Level, LessonSummary } from '../../../core/models/lesson';
import { groupByLevel, Topic, topicLabel, TopicRoadmap } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';
import { DraftChange, DraftList, DraftState } from '../draft-list/draft-list';
import { GenerateDialog, GenerateOptions, GenerateRequest } from '../generate-dialog/generate-dialog';
import { StatusChip } from '../status-chip/status-chip';
import { Loading } from '../../../shared/components/loading/loading';

const GENERATE_FAILED = 'Sinh bài thất bại, vui lòng thử lại.';
const SAVE_FAILED = 'Không lưu được, vui lòng thử lại.';
const DRAFT_FIELDS = ['title', 'content'];

/**
 * One roadmap per topic (F14): choose a topic, then add, remove and reorder its lessons.
 * F7: generate lesson drafts with AI, review them here and save them to the end of the roadmap.
 * Drafts live only in this page; leaving or switching topic with drafts asks first.
 */
@Component({
  selector: 'lu-roadmap',
  imports: [Loading, RouterLink, CdkDropList, CdkDrag, CdkDragHandle, StatusChip, GenerateDialog, DraftList, ConfirmDialog],
  templateUrl: './roadmap.html',
  styleUrl: './roadmap.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(window:beforeunload)': 'onBeforeUnload($event)' },
})
export class Roadmap implements CanLeave {
  private readonly api = inject(AdminApiService);
  private readonly router = inject(Router);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);
  private readonly topicSelect = viewChild<ElementRef<HTMLSelectElement>>('topicSelect');

  protected readonly topicLabel = topicLabel;
  protected readonly topics = signal<Topic[] | null>(null);
  protected readonly topicGroups = computed(() => groupByLevel(this.topics() ?? []));
  protected readonly lowTopics = computed(() => (this.topics() ?? []).filter((t) => t.warning));
  private readonly query = inject(ActivatedRoute).snapshot.queryParamMap;
  protected readonly selectedId = signal(this.query.get('topicId') ?? '');
  /** ?generate=1 (from the Chủ đề page) opens Sinh bài bằng AI once the topic is loaded. */
  private generateOnLoad = this.query.get('generate') === '1';
  protected readonly selectedTopic = computed(() => this.topics()?.find((t) => t.id === this.selectedId()) ?? null);

  protected readonly data = signal<TopicRoadmap | null>(null);
  /** All lessons of the selected topic, to offer those not in the roadmap. */
  private readonly topicLessons = signal<LessonSummary[]>([]);
  protected readonly lessons = computed(() => this.data()?.lessons ?? []);
  protected readonly available = computed(() => {
    const inRoadmap = new Set(this.lessons().map((l) => l.id));
    return this.topicLessons().filter((l) => !inRoadmap.has(l.id));
  });
  protected readonly error = signal<string | null>(null);
  protected readonly saving = signal(false);

  // --- F7: AI drafts ---
  protected readonly dialogOpen = signal(false);
  protected readonly generating = signal(false);
  protected readonly generateError = signal<string | null>(null);
  protected readonly generateNote = signal<string | null>(null);
  /** Last options used, shown again when the dialog reopens. */
  protected readonly options = signal<GenerateOptions>({
    count: DEFAULT_COUNT,
    words: DEFAULT_WORDS.A1,
    kind: 'reading',
    idea: '',
    perLesson: DEFAULT_TARGET_WORDS.A1,
  });
  /** F18: the selected topic's vocabulary, loaded each time the dialog opens (coverage changes). */
  protected readonly topicWords = signal<string[]>([]);
  private openingGenerate = false;
  private optionsLevel: Level | null = null;
  protected readonly drafts = signal<DraftState[]>([]);
  private nextKey = 1;
  /** True during Lưu tất cả. */
  protected readonly savingDrafts = signal(false);

  protected readonly leaveOpen = signal(false);
  private leaveAnswer: ((leave: boolean) => void) | null = null;

  protected readonly warning = computed(() => {
    const d = this.data();
    if (!d?.warning) {
      return null;
    }
    return d.remaining === 0
      ? 'Lộ trình chưa có bài. Thêm bài của chủ đề này bên dưới.'
      : `Lộ trình chỉ còn ${d.remaining} bài chưa học. Hãy thêm bài mới.`;
  });

  constructor() {
    firstValueFrom(this.api.topics())
      .then((list) => {
        this.topics.set(list);
        if (this.selectedId()) {
          void this.load(this.selectedId()).then(() => {
            if (this.generateOnLoad && this.data()) {
              this.generateOnLoad = false;
              void this.openGenerate();
            }
          });
        }
      })
      .catch(() => this.error.set('Không tải được danh sách chủ đề.'));
  }

  protected warningText(t: Topic): string {
    return t.remaining === 0
      ? `${topicLabel(t)}: lộ trình chưa có bài`
      : `${topicLabel(t)}: còn ${t.remaining} bài chưa học`;
  }

  /** Switches topic; with unsaved drafts (they belong to the current topic) asks first. */
  protected async select(id: string): Promise<void> {
    if (id === this.selectedId()) {
      return;
    }
    if (this.drafts().length > 0 && !(await this.askLeave())) {
      const select = this.topicSelect()?.nativeElement;
      if (select) {
        select.value = this.selectedId();
      }
      return;
    }
    this.drafts.set([]);
    this.generateNote.set(null);
    this.selectedId.set(id);
    void this.router.navigate([], { queryParams: { topicId: id || null }, replaceUrl: true });
    this.data.set(null);
    this.topicLessons.set([]);
    if (id) {
      void this.load(id);
    }
  }

  private async load(id: string): Promise<void> {
    this.error.set(null);
    try {
      const [roadmap, lessons] = await Promise.all([
        firstValueFrom(this.api.topicRoadmap(id)),
        firstValueFrom(this.api.list({ topicId: id })),
      ]);
      if (this.selectedId() === id) {
        this.data.set(roadmap);
        this.topicLessons.set(lessons);
        this.updateTopic(roadmap.topic);
      }
    } catch {
      this.error.set('Không tải được lộ trình.');
    }
  }

  private updateTopic(topic: Topic): void {
    this.topics.update((list) => list?.map((t) => (t.id === topic.id ? topic : t)) ?? null);
  }

  protected drop(event: CdkDragDrop<LessonSummary[]>): void {
    if (event.previousIndex === event.currentIndex) {
      return;
    }
    const next = [...this.lessons()];
    moveItemInArray(next, event.previousIndex, event.currentIndex);
    void this.save(next);
  }

  protected move(index: number, delta: -1 | 1): void {
    const next = [...this.lessons()];
    const id = next[index].id;
    moveItemInArray(next, index, index + delta);
    void this.save(next).then(() => this.refocus(id, delta));
  }

  protected remove(index: number): void {
    void this.save(this.lessons().filter((_, i) => i !== index));
  }

  protected add(lesson: LessonSummary): void {
    void this.save([...this.lessons(), lesson]);
  }

  /** Shows the new order at once, then saves it; on failure restores the previous order. */
  private async save(next: LessonSummary[]): Promise<void> {
    const previous = this.data();
    const id = this.selectedId();
    if (!previous || !id) {
      return;
    }
    this.data.set({ ...previous, lessons: next });
    this.error.set(null);
    this.saving.set(true);
    try {
      const saved = await firstValueFrom(this.api.setTopicRoadmap(id, next.map((l) => l.id)));
      this.data.set(saved);
      this.updateTopic(saved.topic);
    } catch {
      this.data.set(previous);
      this.error.set('Không lưu được lộ trình. Thứ tự đã được khôi phục, vui lòng thử lại.');
    } finally {
      this.saving.set(false);
    }
  }

  /**
   * Keeps keyboard focus on the moved lesson after the list re-renders: the same direction
   * button, or the opposite one when the lesson reached the top or bottom.
   */
  private refocus(id: string, delta: -1 | 1): void {
    afterNextRender(
      () => {
        const row = Array.from(this.host.nativeElement.querySelectorAll<HTMLElement>('li[data-id]')).find(
          (li) => li.dataset['id'] === id,
        );
        const same = row?.querySelector<HTMLButtonElement>(`button[data-move="${delta < 0 ? 'up' : 'down'}"]`);
        const other = row?.querySelector<HTMLButtonElement>(`button[data-move="${delta < 0 ? 'down' : 'up'}"]`);
        (same && !same.disabled ? same : other)?.focus();
      },
      { injector: this.injector },
    );
  }

  // --- F7: generating ---

  /** Loads the topic's words first so the dialog can suggest target words; without them it still opens. */
  protected async openGenerate(): Promise<void> {
    const topic = this.selectedTopic();
    if (!topic || this.openingGenerate) {
      return;
    }
    if (topic.level !== this.optionsLevel) {
      this.options.update((o) => ({
        ...o,
        words: DEFAULT_WORDS[topic.level],
        perLesson: DEFAULT_TARGET_WORDS[topic.level],
      }));
      this.optionsLevel = topic.level;
    }
    this.openingGenerate = true;
    let words: string[] = [];
    try {
      words = (await firstValueFrom(this.api.topicWords(topic.id))).map((w) => w.text);
    } catch {
      // Generating without target words is still possible.
    } finally {
      this.openingGenerate = false;
    }
    if (this.selectedId() !== topic.id) {
      return;
    }
    this.topicWords.set(words);
    this.generateError.set(null);
    this.dialogOpen.set(true);
  }

  protected closeGenerate(): void {
    if (!this.generating()) {
      this.dialogOpen.set(false);
    }
  }

  protected async generate(request: GenerateRequest): Promise<void> {
    const id = this.selectedId();
    if (!id || this.generating()) {
      return;
    }
    const { perLesson, targetWords, ...rest } = request;
    const input: GenerateInput = { ...rest, targetWords };
    this.options.set({ ...rest, perLesson });
    this.generating.set(true);
    this.generateError.set(null);
    try {
      const result = await firstValueFrom(this.api.generateLessons(id, input));
      if (this.selectedId() !== id) {
        return;
      }
      this.drafts.update((list) => [
        ...list,
        ...result.drafts.map((d) => ({
          key: this.nextKey++,
          title: d.title,
          content: d.content,
          targetWords: d.targetWords ?? [],
          missingWords: d.missingWords ?? [],
          grammarPointId: d.grammarPointId ?? input.grammarPointId ?? '',
          targetLength: input.words,
          saving: false,
          error: null,
          fields: {},
        })),
      ]);
      this.generateNote.set(generateNote(result));
      this.dialogOpen.set(false);
    } catch (err) {
      this.generateError.set(messageOf(err) ?? GENERATE_FAILED);
    } finally {
      this.generating.set(false);
    }
  }

  // --- F7: reviewing and saving drafts ---

  protected changeDraft(change: DraftChange): void {
    this.patchDraft(change.key, {
      ...(change.title !== undefined ? { title: change.title } : {}),
      ...(change.content !== undefined ? { content: change.content } : {}),
    });
  }

  protected discardDraft(key: number): void {
    this.drafts.update((list) => list.filter((d) => d.key !== key));
  }

  protected async saveDraft(key: number): Promise<void> {
    if (this.savingDrafts()) {
      return;
    }
    if (await this.saveOne(key)) {
      await this.load(this.selectedId());
    }
  }

  /** Saves drafts one after another in display order so the roadmap keeps that order. */
  protected async saveAllDrafts(): Promise<void> {
    if (this.savingDrafts()) {
      return;
    }
    this.savingDrafts.set(true);
    let saved = 0;
    try {
      for (const key of this.drafts().map((d) => d.key)) {
        if (await this.saveOne(key)) {
          saved++;
        }
      }
      if (saved > 0) {
        await this.load(this.selectedId());
      }
    } finally {
      this.savingDrafts.set(false);
    }
  }

  private async saveOne(key: number): Promise<boolean> {
    const draft = this.drafts().find((d) => d.key === key);
    const topicId = this.selectedId();
    if (!draft || draft.saving || !topicId) {
      return false;
    }
    this.patchDraft(key, { saving: true, error: null, fields: {} });
    try {
      await firstValueFrom(
        this.api.create({
          title: draft.title,
          content: draft.content,
          topicId,
          source: AI_SOURCE,
          license: AI_LICENSE,
          appendToRoadmap: true,
          grammarPointId: draft.grammarPointId ?? '',
        }),
      );
      this.discardDraft(key);
      return true;
    } catch (err) {
      const fields = fieldsOf(err);
      const own = Object.fromEntries(Object.entries(fields).filter(([k]) => DRAFT_FIELDS.includes(k)));
      const other = Object.entries(fields)
        .filter(([k]) => !DRAFT_FIELDS.includes(k))
        .map(([, v]) => v);
      const hasFields = Object.keys(fields).length > 0;
      this.patchDraft(key, {
        saving: false,
        fields: own,
        error: other.length ? other.join(' ') : hasFields ? null : (messageOf(err) ?? SAVE_FAILED),
      });
      return false;
    }
  }

  private patchDraft(key: number, patch: Partial<DraftState>): void {
    this.drafts.update((list) => list.map((d) => (d.key === key ? { ...d, ...patch } : d)));
  }

  // --- F7: leaving with unsaved drafts ---

  canLeave(): boolean | Promise<boolean> {
    return this.drafts().length === 0 || this.askLeave();
  }

  private askLeave(): Promise<boolean> {
    this.leaveAnswer?.(false);
    this.leaveOpen.set(true);
    return new Promise((resolve) => (this.leaveAnswer = resolve));
  }

  protected answerLeave(leave: boolean): void {
    this.leaveOpen.set(false);
    this.leaveAnswer?.(leave);
    this.leaveAnswer = null;
  }

  protected onBeforeUnload(event: BeforeUnloadEvent): void {
    if (this.drafts().length > 0) {
      event.preventDefault();
    }
  }
}

function generateNote(result: GenerateResult): string {
  const got = result.drafts.length;
  let note = `Đã thêm ${got} bản nháp.`;
  if (result.dropped > 0) {
    note += ` Đã loại ${result.dropped} bản: ${dropReasonsText(result)}.`;
  } else if (got < result.requested) {
    note += ` AI chỉ trả về ${got} trên ${result.requested} bản.`;
  }
  return note;
}

/** "1 trùng tiêu đề, 2 thiếu tiêu đề hoặc nội dung". */
function dropReasonsText(result: GenerateResult): string {
  const r = result.dropReasons;
  if (!r) {
    return 'không đạt yêu cầu';
  }
  const parts = [
    r.duplicateTitle > 0 ? `${r.duplicateTitle} trùng tiêu đề` : '',
    r.empty > 0 ? `${r.empty} thiếu tiêu đề hoặc nội dung` : '',
    r.tooLong > 0 ? `${r.tooLong} quá dài` : '',
  ].filter(Boolean);
  return parts.length ? parts.join(', ') : 'không đạt yêu cầu';
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const message = (err.body as { message?: unknown } | null)?.message;
    return typeof message === 'string' && message ? message : null;
  }
  return null;
}

function fieldsOf(err: unknown): Record<string, string> {
  if (err instanceof ApiError && err.status === 400) {
    const fields = (err.body as { fields?: Record<string, string> } | null)?.fields;
    return fields ?? {};
  }
  return {};
}
