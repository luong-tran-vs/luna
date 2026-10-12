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
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { CanLeave } from '../../../core/guards/unsaved-changes.guard';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import {
  AI_LICENSE,
  AI_SOURCE,
  DEFAULT_COUNT,
  DEFAULT_IMAGE_STYLE,
  DEFAULT_TARGET_WORDS,
  DEFAULT_WORDS,
  GenerateInput,
  GenerateResult,
} from '../../../core/models/generate';
import { Level, LEVELS, LessonSummary } from '../../../core/models/lesson';
import { lowRoadmaps, roadmapLabel, sortTopics, Topic, TopicLevel, TopicRoadmap, TopicWord } from '../../../core/models/topic';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { LevelOption, LevelPicker } from '../../../shared/components/level-picker/level-picker';
import { clampPage, pageCountOf, Pager, pageSlice, rangeText } from '../../../shared/components/pager/pager';
import { SearchOption, SearchSelect } from '../../../shared/components/search-select/search-select';
import { AdminApiService } from '../admin-api.service';
import { DraftChange, DraftList, DraftState } from '../draft-list/draft-list';
import { GenerateDialog, GenerateOptions, GenerateRequest } from '../generate-dialog/generate-dialog';
import { StatusChip } from '../status-chip/status-chip';
import { Loading } from '../../../shared/components/loading/loading';

const GENERATE_FAILED = 'Sinh bài thất bại, vui lòng thử lại.';
const SAVE_FAILED = 'Không lưu được, vui lòng thử lại.';
const DRAFT_FIELDS = ['title', 'content'];
/** Roadmaps per page of "Tất cả lộ trình". */
const OVERVIEW_PAGE_SIZE = 10;
/** Lessons per page of "chưa có trong lộ trình". */
const AVAILABLE_PAGE_SIZE = 20;

/**
 * One roadmap per topic and level (F14): choose a topic and a level, then add, remove and reorder
 * the lessons of that topic at that level.
 * F7: generate lesson drafts with AI, review them here and save them to the end of the roadmap of
 * their level. Drafts live only in this page; leaving or switching topic with drafts asks first.
 */
@Component({
  selector: 'lu-roadmap',
  imports: [Loading, RouterLink, CdkDropList, CdkDrag, CdkDragHandle, StatusChip, GenerateDialog, DraftList, ConfirmDialog, LevelPicker, SearchSelect, Pager],
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

  protected readonly levels = LEVELS;
  protected readonly topics = signal<Topic[] | null>(null);
  /**
   * The topic choices, "Tất cả lộ trình" (no topic) first, then by name; a topic with lessons
   * outside its roadmaps says how many.
   */
  protected readonly topicChoices = computed<SearchOption[]>(() => [
    { value: '', label: 'Tất cả lộ trình' },
    ...sortTopics(this.topics() ?? []).map((t) => {
      const outside = notInRoadmap(t);
      return { value: t.id, label: t.name, ...(outside > 0 ? { hint: `${outside} bài chưa vào lộ trình` } : {}) };
    }),
  ]);
  protected readonly lowTopics = computed(() => lowRoadmaps(this.topics() ?? []));
  private readonly query = inject(ActivatedRoute).snapshot.queryParamMap;
  protected readonly selectedId = signal(this.query.get('topicId') ?? '');
  /** Level of the roadmap; ?level= or, once the topic is known, its first level with lessons. */
  protected readonly selectedLevel = signal<Level | null>(asLevel(this.query.get('level')));
  /** ?generate=1 (from the Chủ đề page) opens Sinh bài bằng AI once the topic is loaded. */
  private generateOnLoad = this.query.get('generate') === '1';
  protected readonly selectedTopic = computed(() => this.topics()?.find((t) => t.id === this.selectedId()) ?? null);
  /** One card per level, with how many lessons of the topic it has. */
  protected readonly levelOptions = computed<LevelOption[]>(() => {
    const t = this.selectedTopic();
    return this.levels.map((level) => {
      const n = t?.levels.find((l) => l.level === level)?.lessonCount ?? 0;
      return { value: level, label: level, hint: n > 0 ? `${n} bài` : 'Chưa có bài' };
    });
  });
  /** "A1 · Gia đình" */
  protected readonly selectedLabel = computed(() => {
    const t = this.selectedTopic();
    const level = this.selectedLevel();
    return t && level ? roadmapLabel(level, t) : '';
  });

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
  /** Without a chosen topic: every roadmap that has lessons, by topic name then level. */
  protected readonly overview = signal<TopicRoadmap[] | null>(null);
  protected readonly overviewCount = computed(() => this.overview()?.reduce((n, r) => n + r.lessons.length, 0) ?? 0);
  protected readonly overviewPage = signal(1);
  protected readonly overviewPageCount = computed(() => pageCountOf(this.overview()?.length ?? 0, OVERVIEW_PAGE_SIZE));
  protected readonly overviewCurrentPage = computed(() =>
    clampPage(this.overviewPage(), this.overview()?.length ?? 0, OVERVIEW_PAGE_SIZE),
  );
  protected readonly overviewShown = computed(() => pageSlice(this.overview() ?? [], this.overviewPage(), OVERVIEW_PAGE_SIZE));
  protected readonly overviewRange = computed(
    () => `${rangeText(this.overviewPage(), this.overview()?.length ?? 0, OVERVIEW_PAGE_SIZE)} lộ trình`,
  );
  /** Lessons ticked in the overview, to take out of their roadmaps at once. */
  protected readonly ticked = signal<ReadonlySet<string>>(new Set());
  protected readonly tickedCount = computed(() => {
    const ids = this.ticked();
    return this.overview()?.reduce((n, r) => n + r.lessons.filter((l) => ids.has(l.id)).length, 0) ?? 0;
  });
  protected readonly allTicked = computed(() => this.overviewCount() > 0 && this.tickedCount() === this.overviewCount());
  protected readonly removeOpen = signal(false);
  protected readonly overviewNote = signal<string | null>(null);

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
    images: false,
    imageStyle: DEFAULT_IMAGE_STYLE,
  });
  /** F18: the selected topic's vocabulary, loaded each time the dialog opens (coverage changes). */
  protected readonly topicWords = signal<TopicWord[]>([]);
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
      ? 'Lộ trình chưa có bài. Thêm bài của chủ đề ở trình độ này bên dưới.'
      : `Lộ trình chỉ còn ${d.remaining} bài chưa học. Hãy thêm bài mới.`;
  });

  constructor() {
    firstValueFrom(this.api.topics())
      .then((list) => {
        this.topics.set(list);
        if (this.selectedId()) {
          this.selectedLevel.update((l) => l ?? firstLevel(this.selectedTopic()));
          void this.load(this.selectedId()).then(() => {
            if (this.generateOnLoad && this.data()) {
              this.generateOnLoad = false;
              void this.openGenerate();
            }
          });
        } else {
          void this.loadOverview();
        }
      })
      .catch(() => this.error.set('Không tải được danh sách chủ đề.'));
  }

  protected warningText(t: Topic, l: TopicLevel): string {
    return l.remaining === 0
      ? `${roadmapLabel(l.level, t)}: lộ trình chưa có bài`
      : `${roadmapLabel(l.level, t)}: còn ${l.remaining} bài chưa học`;
  }

  /** Switches level of the same topic. Drafts stay: each one is saved to the roadmap of its own level. */
  protected selectLevel(value: string): void {
    const level = asLevel(value);
    if (!level || level === this.selectedLevel()) {
      return;
    }
    this.selectedLevel.set(level);
    void this.router.navigate([], { queryParams: { topicId: this.selectedId() || null, level }, replaceUrl: true });
    this.data.set(null);
    this.topicLessons.set([]);
    this.availablePage.set(1);
    if (this.selectedId()) {
      void this.load(this.selectedId());
    }
  }

  /** Switches topic (and level, from a warning); with unsaved drafts (they belong to the current topic) asks first. */
  protected async select(id: string, level?: Level): Promise<void> {
    if (id === this.selectedId()) {
      if (level) {
        this.selectLevel(level);
      }
      return;
    }
    // Refused: the topic box keeps showing selectedId().
    if (this.drafts().length > 0 && !(await this.askLeave())) {
      return;
    }
    this.drafts.set([]);
    this.generateNote.set(null);
    this.selectedId.set(id);
    this.selectedLevel.set(level ?? firstLevel(this.selectedTopic()));
    void this.router.navigate([], { queryParams: { topicId: id || null, level: id ? this.selectedLevel() : null }, replaceUrl: true });
    this.data.set(null);
    this.topicLessons.set([]);
    this.availablePage.set(1);
    if (id) {
      void this.load(id);
    } else {
      void this.loadOverview();
    }
  }

  /** Loads every roadmap with lessons (one request each, in parallel). */
  private async loadOverview(): Promise<void> {
    const wanted = sortTopics(this.topics() ?? []).flatMap((t) =>
      t.levels.filter((l) => l.roadmapCount > 0).map((l) => ({ id: t.id, level: l.level })),
    );
    this.overview.set(null);
    this.ticked.set(new Set());
    this.overviewPage.set(1);
    this.overviewNote.set(null);
    this.error.set(null);
    try {
      const list = await Promise.all(wanted.map((w) => firstValueFrom(this.api.topicRoadmap(w.id, w.level))));
      if (!this.selectedId()) {
        this.overview.set(list.filter((r) => r.lessons.length > 0));
      }
    } catch {
      this.error.set('Không tải được danh sách bài của lộ trình.');
    }
  }

  protected overviewLabel(r: TopicRoadmap): string {
    return roadmapLabel(r.level, r.topic);
  }

  protected tick(ids: string[], checked: boolean): void {
    this.ticked.update((s) => {
      const next = new Set(s);
      for (const id of ids) {
        if (checked) {
          next.add(id);
        } else {
          next.delete(id);
        }
      }
      return next;
    });
  }

  /** Roadmaps opened in the overview (topic id + level); all folded at first. */
  private readonly expanded = signal<ReadonlySet<string>>(new Set());

  protected isCollapsed(r: TopicRoadmap): boolean {
    return !this.expanded().has(r.topic.id + r.level);
  }

  /** Clicking a roadmap's name opens it; clicking again folds it. */
  protected toggleGroup(r: TopicRoadmap): void {
    const key = r.topic.id + r.level;
    this.expanded.update((s) => {
      const next = new Set(s);
      if (!next.delete(key)) {
        next.add(key);
      }
      return next;
    });
  }

  protected tickGroup(r: TopicRoadmap, checked: boolean): void {
    this.tick(r.lessons.map((l) => l.id), checked);
  }

  protected tickAll(checked: boolean): void {
    this.ticked.set(checked ? new Set(this.overview()?.flatMap((r) => r.lessons.map((l) => l.id))) : new Set());
  }

  protected groupState(r: TopicRoadmap): 'all' | 'some' | 'none' {
    const n = r.lessons.filter((l) => this.ticked().has(l.id)).length;
    return n === 0 ? 'none' : n === r.lessons.length ? 'all' : 'some';
  }

  /**
   * Takes the ticked lessons out of their roadmaps: one save per roadmap with the rest in the same
   * order. The lessons are kept (still listed on the Bài học page).
   */
  protected removeTicked(): Promise<void> {
    this.removeOpen.set(false);
    return this.removeFromRoadmaps(this.ticked());
  }

  /** Gỡ next to one lesson of the overview: no tick needed. */
  protected removeOne(id: string): Promise<void> {
    return this.removeFromRoadmaps(new Set([id]));
  }

  private async removeFromRoadmaps(ids: ReadonlySet<string>): Promise<void> {
    const roadmaps = this.overview();
    if (!roadmaps || this.saving()) {
      return;
    }
    const changed = roadmaps.filter((r) => r.lessons.some((l) => ids.has(l.id)));
    this.saving.set(true);
    this.error.set(null);
    this.overviewNote.set(null);
    let removed = 0;
    let failed = 0;
    for (const r of changed) {
      const keep = r.lessons.filter((l) => !ids.has(l.id));
      try {
        const saved = await firstValueFrom(this.api.setTopicRoadmap(r.topic.id, r.level, keep.map((l) => l.id)));
        removed += r.lessons.length - keep.length;
        this.updateTopic(saved.topic);
        this.overview.update((list) => list?.map((x) => (x === r ? saved : x)) ?? null);
        this.tick(r.lessons.map((l) => l.id), false);
      } catch {
        failed += r.lessons.length - keep.length;
      }
    }
    this.overview.update((list) => list?.filter((r) => r.lessons.length > 0) ?? null);
    this.saving.set(false);
    if (removed > 0) {
      this.overviewNote.set(`Đã gỡ ${removed} bài khỏi lộ trình.`);
    }
    if (failed > 0) {
      this.error.set(`Không gỡ được ${failed} bài, vui lòng thử lại.`);
    }
  }

  private async load(id: string): Promise<void> {
    const level = this.selectedLevel() ?? 'A1';
    this.error.set(null);
    try {
      const [roadmap, lessons] = await Promise.all([
        firstValueFrom(this.api.topicRoadmap(id, level)),
        firstValueFrom(this.api.list({ topicId: id, level })),
      ]);
      if (this.selectedId() === id && this.selectedLevel() === level) {
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

  // --- quick add: lessons ticked in "chưa có trong lộ trình" ---
  protected readonly availablePage = signal(1);
  protected readonly availablePageCount = computed(() => pageCountOf(this.available().length, AVAILABLE_PAGE_SIZE));
  protected readonly availableCurrentPage = computed(() =>
    clampPage(this.availablePage(), this.available().length, AVAILABLE_PAGE_SIZE),
  );
  protected readonly availableShown = computed(() => pageSlice(this.available(), this.availablePage(), AVAILABLE_PAGE_SIZE));
  protected readonly availableRange = computed(
    () => `${rangeText(this.availablePage(), this.available().length, AVAILABLE_PAGE_SIZE)} bài`,
  );
  protected readonly toAdd = signal<ReadonlySet<string>>(new Set());
  protected readonly toAddLessons = computed(() => this.available().filter((l) => this.toAdd().has(l.id)));
  protected readonly allToAdd = computed(
    () => this.available().length > 0 && this.toAddLessons().length === this.available().length,
  );

  protected tickToAdd(id: string, checked: boolean): void {
    this.toAdd.update((s) => {
      const next = new Set(s);
      if (checked) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }

  protected tickAllToAdd(checked: boolean): void {
    this.toAdd.set(checked ? new Set(this.available().map((l) => l.id)) : new Set());
  }

  /** Adds the ticked lessons at the end of the roadmap, in the order they are listed. */
  protected async addTicked(): Promise<void> {
    const picked = this.toAddLessons();
    if (picked.length === 0 || this.saving()) {
      return;
    }
    await this.save([...this.lessons(), ...picked]);
    // Added ones left `available`; failed ones stay ticked.
    this.toAdd.update((s) => new Set([...s].filter((id) => this.available().some((l) => l.id === id))));
  }

  /** Shows the new order at once, then saves it; on failure restores the previous order. */
  private async save(next: LessonSummary[]): Promise<void> {
    const previous = this.data();
    const id = this.selectedId();
    const level = this.selectedLevel();
    if (!previous || !id || !level) {
      return;
    }
    this.data.set({ ...previous, lessons: next });
    this.error.set(null);
    this.saving.set(true);
    try {
      const saved = await firstValueFrom(this.api.setTopicRoadmap(id, level, next.map((l) => l.id)));
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
    const level = this.selectedLevel();
    if (!topic || !level || this.openingGenerate) {
      return;
    }
    if (level !== this.optionsLevel) {
      this.options.update((o) => ({
        ...o,
        words: DEFAULT_WORDS[level],
        perLesson: DEFAULT_TARGET_WORDS[level],
      }));
      this.optionsLevel = level;
    }
    this.openingGenerate = true;
    let words: TopicWord[] = [];
    try {
      words = await firstValueFrom(this.api.topicWords(topic.id));
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
    const { perLesson, targetWords, images, imageStyle, level, ...rest } = request;
    const input: GenerateInput = { ...rest, level, targetWords };
    this.options.set({ ...rest, perLesson, images, imageStyle });
    this.optionsLevel = level;
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
          level,
          title: d.title,
          content: d.content,
          targetWords: d.targetWords ?? [],
          missingWords: d.missingWords ?? [],
          grammarPointId: d.grammarPointId ?? input.grammarPointId ?? '',
          targetLength: input.words,
          images: images ? { enabled: true, style: imageStyle } : null,
          saving: false,
          error: null,
          fields: {},
        })),
      ]);
      this.generateNote.set(generateNote(result, level === this.selectedLevel() ? null : level));
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
          level: draft.level,
          source: AI_SOURCE,
          license: AI_LICENSE,
          appendToRoadmap: true,
          draft: true,
          grammarPointId: draft.grammarPointId ?? '',
          ...(draft.targetWords.length > 0 ? { targetWords: draft.targetWords } : {}),
          ...(draft.images ? { images: draft.images } : {}),
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

/** Lessons of the topic, every level, that are in none of its roadmaps. */
function notInRoadmap(t: Topic): number {
  return t.levels.reduce((n, l) => n + Math.max(0, l.lessonCount - l.roadmapCount), 0);
}

/** A level from a query parameter, or null. */
function asLevel(value: string | null): Level | null {
  return LEVELS.find((l) => l === value) ?? null;
}

/** The first level of a topic that has lessons, else A1. */
function firstLevel(topic: Topic | null): Level {
  return topic?.levels.find((l) => l.lessonCount > 0)?.level ?? topic?.levels.at(0)?.level ?? 'A1';
}

/** otherLevel: the drafts were written for another level than the roadmap shown. */
function generateNote(result: GenerateResult, otherLevel: Level | null): string {
  const got = result.drafts.length;
  let note = `Đã thêm ${got} bản nháp${otherLevel ? ` trình độ ${otherLevel} (lưu vào lộ trình ${otherLevel})` : ''}.`;
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
