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
import { catchError, map, Observable, of, Subject, switchMap, timer } from 'rxjs';

import {
  GenerateInput,
  LessonKind,
  MAX_COUNT,
  MAX_IDEA,
  MAX_TARGET_WORDS,
  MAX_WORDS,
  MIN_COUNT,
  MIN_WORDS,
  wordRange,
} from '../../../core/models/generate';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AdminApiService } from '../admin-api.service';

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
}

/** What the dialog emits: the request body plus the words per lesson to remember. */
export type GenerateRequest = GenerateInput & { perLesson: number };

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
  imports: [ReactiveFormsModule],
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
  /** The topic's vocabulary list; empty hides the target words. */
  readonly topicWords = input<string[]>([]);
  readonly options = input.required<GenerateOptions>();
  readonly busy = input(false);
  readonly error = input<string | null>(null);

  readonly generate = output<GenerateRequest>();
  /** F18: the topic's whole word list after the AI added words to it. */
  readonly wordsChanged = output<string[]>();
  readonly closed = output<void>();

  protected readonly maxIdea = MAX_IDEA;
  protected readonly minCount = MIN_COUNT;
  protected readonly maxCount = MAX_COUNT;
  protected readonly minWords = MIN_WORDS;
  protected readonly maxWords = MAX_WORDS;
  protected readonly maxTarget = MAX_TARGET_WORDS;

  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
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
    perLesson: new FormControl<number | null>(null, [
      Validators.required,
      Validators.min(0),
      Validators.max(MAX_TARGET_WORDS),
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

  // --- F18: target words ---
  protected readonly hasWords = computed(() => this.topicWords().length > 0);
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
  private readonly planRequests = new Subject<{ count: number; perLesson: number; delay: number }>();
  /** "count:perLesson" of the split shown or being loaded; avoids asking again for the same one. */
  private plannedKey: string | null = null;
  private resetting = false;

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
          this.form.reset(this.options());
          this.resetting = false;
          this.plannedKey = null;
          this.groups.set([]);
          this.addErrors.set({});
          this.planState.set('idle');
          this.shortage.set(0);
          this.suggestNote.set(null);
          this.requestPlan(0);
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
    // switchMap drops a pending or running request when a newer one comes.
    this.planRequests
      .pipe(
        switchMap((r) => timer(r.delay).pipe(switchMap(() => this.loadPlan(r.count, r.perLesson)))),
        takeUntilDestroyed(),
      )
      .subscribe((result) => {
        this.groups.set(result.groups);
        this.shortage.set(result.shortage);
        this.addErrors.set({});
        this.planState.set(result.ok ? 'idle' : 'error');
      });
  }

  private loadPlan(count: number, perLesson: number): Observable<PlanResult> {
    if (perLesson === 0) {
      return of({ ok: true, groups: [], shortage: 0 });
    }
    return this.api.wordPlan(this.topicId(), count, perLesson).pipe(
      map((plan) => ({ ok: true, groups: plan.groups, shortage: plan.shortage })),
      catchError(() => of({ ok: false, groups: [], shortage: 0 })),
    );
  }

  /** Asks for a new split when the number of lessons or of words per lesson changed. */
  private requestPlan(delay: number): void {
    if (!this.open() || this.busy() || !this.hasWords()) {
      return;
    }
    const { count, perLesson } = this.form.controls;
    if (count.invalid || perLesson.invalid) {
      return;
    }
    const key = `${Number(count.value)}:${Number(perLesson.value)}`;
    if (key === this.plannedKey) {
      return;
    }
    this.plannedKey = key;
    this.planState.set(Number(perLesson.value) === 0 ? 'idle' : 'loading');
    this.planRequests.next({ count: Number(count.value), perLesson: Number(perLesson.value), delay });
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
    this.api.suggestTopicWords(this.topicId(), count).subscribe({
      next: (r) => {
        this.suggesting.set(false);
        this.wordsChanged.emit(r.words.map((w) => w.text));
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

  /** Topic words not yet in the group, offered by the add box. */
  protected suggestions(group: readonly string[]): string[] {
    const taken = new Set(group.map((w) => w.toLowerCase()));
    return this.topicWords().filter((w) => !taken.has(w.toLowerCase()));
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
    const word = this.topicWords().find((w) => w.toLowerCase() === lower);
    const group = this.groups()[groupIndex] ?? [];
    if (!word) {
      this.setAddError(groupIndex, 'Từ không có trong danh sách');
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
    const invalid = c.count.invalid || c.words.invalid || c.idea.invalid || (this.hasWords() && c.perLesson.invalid);
    if (invalid || this.busy() || this.planState() === 'loading') {
      return;
    }
    const v = this.form.getRawValue();
    const count = Number(v.count);
    const perLesson = c.perLesson.valid ? Number(v.perLesson) : this.options().perLesson;
    const groups = this.groups();
    const targetWords = this.hasWords() && perLesson > 0 && groups.length === count ? groups.map((g) => [...g]) : [];
    this.generate.emit({
      count,
      words: Number(v.words),
      kind: v.kind,
      idea: v.idea.trim(),
      targetWords,
      perLesson,
    });
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
