import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { catchError, map, Observable, of, Subject, Subscription, switchMap, timer } from 'rxjs';

import {
  DEFAULT_TARGET_WORDS,
  DEFAULT_WORDS,
  GenerateInput,
  LessonKind,
  MAX_COUNT,
  MAX_IDEA,
  MAX_IMAGE_STYLE,
  MAX_TARGET_WORDS,
  MAX_WORDS,
  MIN_COUNT,
  MIN_WORDS,
  wordRange,
} from '../../../core/models/generate';
import { GrammarPoint, grammarOptionLabel } from '../../../core/models/grammar';
import { Level, LEVELS } from '../../../core/models/lesson';
import { TopicWord } from '../../../core/models/topic';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AdminApiService } from '../admin-api.service';
import { Loading } from '../../../shared/components/loading/loading';
import { LevelPicker } from '../../../shared/components/level-picker/level-picker';

const INTEGER = /^\d+$/;
/** Wait after the admin stops typing a number before asking for a new split. */
export const PLAN_DEBOUNCE_MS = 300;

/** Values shown when the dialog opens (the last ones used). */
export interface GenerateOptions {
  count: number;
  words: number;
  kind: LessonKind;
  idea: string;
  /** F18: target words per lesson (used only when the topic has words). */
  perLesson: number;
  /** F23: draw a picture for each vocabulary word of the saved lessons. */
  images: boolean;
  /** F23: how the pictures should look. */
  imageStyle: string;
}

/**
 * What the dialog emits: the request body plus what the page keeps beside it: the words per lesson
 * and the picture settings given to each lesson saved from the drafts (F23).
 */
export type GenerateRequest = GenerateInput & { perLesson: number; images: boolean; imageStyle: string };

type GrammarState = 'idle' | 'loading' | 'error';

type PlanState = 'idle' | 'loading' | 'error';

interface PlanResult {
  ok: boolean;
  groups: string[][];
  shortage: number;
}

/** Most words one AI suggestion may add (backend MaxSuggestWords). */
const MAX_SUGGEST = 50;

/**
 * "Sinh bài bằng AI" dialog (F7). A native <dialog> opened with showModal(), so focus stays
 * inside and Escape closes it (except while generating). The parent owns the request; the
 * dialog only validates and emits.
 *
 * F18: when the topic has a vocabulary list, the server suggests target words for each lesson
 * (least used first); the admin can remove or add words per lesson before generating.
 */
@Component({
  selector: 'lu-generate-dialog',
  imports: [Loading, ReactiveFormsModule, LevelPicker],
  templateUrl: './generate-dialog.html',
  styleUrl: './generate-dialog.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GenerateDialog {
  private readonly api = inject(AdminApiService);

  readonly open = input(false);
  /** "A1 · Gia đình". */
  readonly topicLabel = input.required<string>();
  readonly topicId = input('');
  /** Level the dialog opens with (the roadmap's); the admin may pick another. It picks the grammar points offered. */
  readonly level = input<Level>('A1');
  /** The topic's vocabulary list; empty hides the target words. */
  readonly topicWords = input<TopicWord[]>([]);
  readonly options = input.required<GenerateOptions>();
  readonly busy = input(false);
  readonly error = input<string | null>(null);

  readonly generate = output<GenerateRequest>();
  /** F18: the topic's whole word list after the AI added words to it. */
  readonly wordsChanged = output<TopicWord[]>();
  readonly closed = output<void>();

  protected readonly maxIdea = MAX_IDEA;
  protected readonly minCount = MIN_COUNT;
  protected readonly maxCount = MAX_COUNT;
  protected readonly minWords = MIN_WORDS;
  protected readonly maxWords = MAX_WORDS;
  protected readonly maxTarget = MAX_TARGET_WORDS;
  protected readonly maxImageStyle = MAX_IMAGE_STYLE;

  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly submitted = signal(false);

  protected readonly form = new FormGroup({
    level: new FormControl<Level>('A1', { nonNullable: true }),
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
    perLesson: new FormControl<number | null>(null, [
      Validators.required,
      Validators.min(0),
      Validators.max(MAX_TARGET_WORDS),
      Validators.pattern(INTEGER),
    ]),
    kind: new FormControl<LessonKind>('reading', { nonNullable: true }),
    idea: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_IDEA)] }),
    grammarPointId: new FormControl('', { nonNullable: true }),
    images: new FormControl(false, { nonNullable: true }),
    imageStyle: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_IMAGE_STYLE)] }),
  });

  private readonly value = toSignal(this.form.valueChanges, { initialValue: this.form.value });
  private readonly status = toSignal(this.form.statusChanges, { initialValue: this.form.status });

  protected readonly ideaLength = computed(() => (this.value().idea ?? '').length);
  protected readonly imageStyleLength = computed(() => (this.value().imageStyle ?? '').length);
  protected readonly imagesOn = computed(() => !!this.value().images);
  protected readonly range = computed(() => {
    const words = Number(this.value().words);
    return Number.isInteger(words) && words >= MIN_WORDS && words <= MAX_WORDS ? wordRange(words) : null;
  });

  // --- Grammar point of the curriculum ---
  protected readonly grammarPoints = signal<GrammarPoint[]>([]);
  protected readonly grammarState = signal<GrammarState>('idle');
  protected readonly grammarLabel = grammarOptionLabel;
  protected readonly selectedGrammar = computed(
    () => this.grammarPoints().find((p) => p.id === this.value().grammarPointId) ?? null,
  );
  private grammarSub: Subscription | null = null;

  // --- F18: target words ---
  protected readonly hasWords = computed(() => this.topicWords().length > 0);
  /** Topic words that may go to a lesson of the chosen level: of that level or lower, or of any level. */
  private readonly fitWords = computed(() => {
    const rank = LEVELS.indexOf((this.value().level ?? 'A1') as Level);
    return this.topicWords()
      .filter((w) => !w.level || LEVELS.indexOf(w.level) <= rank)
      .map((w) => w.text);
  });
  /** One group of target words per lesson, as suggested then edited. */
  protected readonly groups = signal<string[][]>([]);
  protected readonly planState = signal<PlanState>('idle');
  /** How many more unused words the topic needs for this split (F18). */
  protected readonly shortage = signal(0);
  protected readonly suggesting = signal(false);
  /** Result or error of the last AI suggestion, announced to screen readers. */
  protected readonly suggestNote = signal<{ ok: boolean; text: string } | null>(null);
  /** Error of the add box of each group, by group index. */
  protected readonly addErrors = signal<Record<number, string>>({});
  private readonly planRequests = new Subject<{ level: Level; count: number; perLesson: number; delay: number }>();
  /** "level:count:perLesson" of the split shown or being loaded; avoids asking again for the same one. */
  private plannedKey: string | null = null;
  private resetting = false;
  /** Level the form shows, to tell a level change from enable()/disable() emitting it again. */
  private shownLevel: Level = 'A1';

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
    if (this.hasWords() && c.perLesson.invalid) {
      out['perLesson'] = `Số từ mục tiêu từ 0 đến ${MAX_TARGET_WORDS}`;
    }
    if (c.idea.invalid) {
      out['idea'] = `Ý chính tối đa ${MAX_IDEA} ký tự`;
    }
    if (c.images.value && c.imageStyle.invalid) {
      out['imageStyle'] = `Mô tả ảnh tối đa ${MAX_IMAGE_STYLE} ký tự`;
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
          this.resetting = true;
          this.form.reset({ ...this.options(), level: this.level() });
          this.shownLevel = this.level();
          this.resetting = false;
          this.plannedKey = null;
          this.groups.set([]);
          this.addErrors.set({});
          this.planState.set('idle');
          this.shortage.set(0);
          this.suggestNote.set(null);
          this.requestPlan(0);
          this.loadGrammar();
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

    this.form.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
      if (!this.resetting) {
        this.requestPlan(PLAN_DEBOUNCE_MS);
      }
    });
    // Another level: its default length and words per lesson, and its grammar points. enable()
    // and disable() emit the same level again, which changes nothing.
    this.form.controls.level.valueChanges.pipe(takeUntilDestroyed()).subscribe((level) => {
      if (this.resetting || level === this.shownLevel) {
        return;
      }
      this.shownLevel = level;
      this.form.patchValue({ words: DEFAULT_WORDS[level], perLesson: DEFAULT_TARGET_WORDS[level], grammarPointId: '' });
      this.loadGrammar();
    });
    // switchMap drops a pending or running request when a newer one comes.
    this.planRequests
      .pipe(
        switchMap((r) => timer(r.delay).pipe(switchMap(() => this.loadPlan(r.level, r.count, r.perLesson)))),
        takeUntilDestroyed(),
      )
      .subscribe((result) => {
        this.groups.set(result.groups);
        this.shortage.set(result.shortage);
        this.addErrors.set({});
        this.planState.set(result.ok ? 'idle' : 'error');
      });
  }

  /** Loads the level's grammar points; the default stays "AI tự chọn". A failure only hides the select. */
  private loadGrammar(): void {
    this.grammarSub?.unsubscribe();
    this.grammarPoints.set([]);
    const level = this.form.controls.level.value;
    if (!this.topicId()) {
      this.grammarState.set('idle');
      return;
    }
    this.grammarState.set('loading');
    this.grammarSub = this.api.grammarPoints(level, this.topicId() || undefined).subscribe({
      next: (points) => {
        this.grammarPoints.set(points);
        this.grammarState.set('idle');
      },
      error: () => this.grammarState.set('error'),
    });
  }

  private loadPlan(level: Level, count: number, perLesson: number): Observable<PlanResult> {
    if (perLesson === 0) {
      return of({ ok: true, groups: [], shortage: 0 });
    }
    return this.api.wordPlan(this.topicId(), level, count, perLesson).pipe(
      map((plan) => ({ ok: true, groups: plan.groups, shortage: plan.shortage })),
      catchError(() => of({ ok: false, groups: [], shortage: 0 })),
    );
  }

  /** Asks for a new split when the number of lessons or of words per lesson changed. */
  private requestPlan(delay: number): void {
    if (!this.open() || this.busy() || !this.hasWords()) {
      return;
    }
    const { level, count, perLesson } = this.form.controls;
    if (count.invalid || perLesson.invalid) {
      return;
    }
    const key = `${level.value}:${Number(count.value)}:${Number(perLesson.value)}`;
    if (key === this.plannedKey) {
      return;
    }
    this.plannedKey = key;
    this.planState.set(Number(perLesson.value) === 0 ? 'idle' : 'loading');
    this.planRequests.next({ level: level.value, count: Number(count.value), perLesson: Number(perLesson.value), delay });
  }

  protected retryPlan(): void {
    this.plannedKey = null;
    this.requestPlan(0);
  }

  /** Asks the AI for the missing words, adds them to the topic, then splits again (F18). */
  protected suggestWords(): void {
    const count = Math.min(this.shortage(), MAX_SUGGEST);
    if (count < 1 || this.suggesting() || this.busy()) {
      return;
    }
    this.suggesting.set(true);
    this.suggestNote.set(null);
    this.api.suggestTopicWords(this.topicId(), this.form.controls.level.value, count).subscribe({
      next: (r) => {
        this.suggesting.set(false);
        this.wordsChanged.emit(r.words);
        this.suggestNote.set({ ok: true, text: `Đã thêm ${r.added.length} từ: ${r.added.join(', ')}.` });
        this.retryPlan();
      },
      error: (err: unknown) => {
        this.suggesting.set(false);
        const message = err instanceof ApiError ? (err.body as { message?: unknown } | null)?.message : null;
        const text = typeof message === 'string' && message ? message : 'Không bổ sung được từ, thử lại.';
        this.suggestNote.set({ ok: false, text });
      },
    });
  }

  /** Topic words of the chosen level not yet in the group, offered by the add box. */
  protected suggestions(group: readonly string[]): string[] {
    const taken = new Set(group.map((w) => w.toLowerCase()));
    return this.fitWords().filter((w) => !taken.has(w.toLowerCase()));
  }

  protected removeWord(groupIndex: number, word: string): void {
    this.groups.update((groups) => groups.map((g, i) => (i === groupIndex ? g.filter((w) => w !== word) : g)));
    this.setAddError(groupIndex, null);
    this.host.nativeElement.querySelector<HTMLInputElement>(`#generate-add-${groupIndex}`)?.focus();
  }

  /** Enter in the add box adds the word instead of submitting the form. */
  protected onAddKey(event: KeyboardEvent, groupIndex: number, box: HTMLInputElement): void {
    if (event.key === 'Enter') {
      event.preventDefault();
      this.addWord(groupIndex, box);
    }
  }

  protected addWord(groupIndex: number, box: HTMLInputElement): void {
    const typed = box.value.trim().replace(/\s+/g, ' ');
    if (!typed) {
      return;
    }
    const lower = typed.toLowerCase();
    const word = this.fitWords().find((w) => w.toLowerCase() === lower);
    const group = this.groups()[groupIndex] ?? [];
    if (!word) {
      const other = this.topicWords().find((w) => w.text.toLowerCase() === lower);
      this.setAddError(groupIndex, other ? `Từ này thuộc trình độ ${other.level}` : 'Từ không có trong danh sách');
      return;
    }
    if (group.some((w) => w.toLowerCase() === lower)) {
      this.setAddError(groupIndex, 'Từ đã có trong bài này');
      return;
    }
    if (group.length >= MAX_TARGET_WORDS) {
      this.setAddError(groupIndex, `Tối đa ${MAX_TARGET_WORDS} từ mỗi bài`);
      return;
    }
    this.groups.update((groups) => groups.map((g, i) => (i === groupIndex ? [...g, word] : g)));
    this.setAddError(groupIndex, null);
    box.value = '';
  }

  private setAddError(groupIndex: number, message: string | null): void {
    this.addErrors.update((errors) => {
      const next = { ...errors };
      if (message) {
        next[groupIndex] = message;
      } else {
        delete next[groupIndex];
      }
      return next;
    });
  }

  protected describedBy(field: string, hint?: string): string | null {
    const ids = [hint, this.errors()[field] ? `generate-${field}-error` : undefined].filter(Boolean);
    return ids.length ? ids.join(' ') : null;
  }

  protected submit(): void {
    this.submitted.set(true);
    this.form.updateValueAndValidity();
    const c = this.form.controls;
    const invalid =
      c.count.invalid ||
      c.words.invalid ||
      c.idea.invalid ||
      (this.hasWords() && c.perLesson.invalid) ||
      (c.images.value && c.imageStyle.invalid);
    if (invalid || this.busy() || this.planState() === 'loading') {
      return;
    }
    const v = this.form.getRawValue();
    const count = Number(v.count);
    const perLesson = c.perLesson.valid ? Number(v.perLesson) : this.options().perLesson;
    const groups = this.groups();
    const targetWords = this.hasWords() && perLesson > 0 && groups.length === count ? groups.map((g) => [...g]) : [];
    this.generate.emit({
      level: v.level,
      count,
      words: Number(v.words),
      kind: v.kind,
      idea: v.idea.trim(),
      targetWords,
      grammarPointId: this.grammarState() === 'idle' ? v.grammarPointId : '',
      perLesson,
      images: v.images,
      imageStyle: v.imageStyle.trim(),
    });
  }

  /** F23: the "Sinh ảnh cho từ vựng" switch. */
  protected toggleImages(): void {
    const c = this.form.controls.images;
    c.setValue(!c.value);
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
