import { DatePipe, DOCUMENT, NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { GrammarPoint } from '../../../core/models/grammar';
import {
  CHECK_LABELS,
  CHECK_SHORT,
  checkStatusText,
  openChecks,
  GrammarExerciseInput,
  GrammarExerciseKind,
  ExerciseReports,
  EXERCISE_KIND_LABELS,
  GrammarCheck,
  GrammarContent,
  GrammarExercise,
  GrammarLessonAdmin,
  grammarStatusText,
  reasonsText,
} from '../../../core/models/grammar-admin';
import { LEVELS } from '../../../core/models/lesson';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { Loading } from '../../../shared/components/loading/loading';
import { AdminApiService } from '../admin-api.service';
import { grammarFailure } from '../grammar-errors';

type LoadState = 'loading' | 'ready' | 'empty' | 'error';
type Action =
  | 'generate'
  | 'check'
  | 'publish'
  | 'unpublish'
  | 'save'
  | 'confirm'
  | 'verify'
  | 'exercise'
  | 'delete';

/** Fields of the exercise form that the server can complain about. */
const EXERCISE_FIELDS = ['promptVi', 'text', 'options', 'answerIndex', 'answers', 'words', 'sentence', 'explanationVi'];

/** "a1-to-be" belongs to level A1; null for an id that does not start with a level. */
function levelOf(pointId: string): string | null {
  const level = pointId.split('-')[0].toUpperCase();
  return (LEVELS as readonly string[]).includes(level) ? level : null;
}

/** Pretty JSON of the content, as shown in the editor. */
export function formatContent(content: GrammarContent): string {
  return JSON.stringify(content, null, 2);
}

/**
 * F20: one grammar lesson for the admin: the point of the curriculum, a preview of the content
 * by section, and a JSON editor for the whole content. The server validates on save and publish.
 */
@Component({
  selector: 'lu-grammar-detail',
  imports: [Loading, ConfirmDialog, DatePipe, NgTemplateOutlet, ReactiveFormsModule, RouterLink],
  templateUrl: './grammar-detail.html',
  styleUrl: './grammar-detail.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarDetail {
  private readonly api = inject(AdminApiService);
  private readonly doc = inject(DOCUMENT);
  protected readonly pointId = inject(ActivatedRoute).snapshot.paramMap.get('pointId') ?? '';

  protected readonly statusText = grammarStatusText;
  protected readonly state = signal<LoadState>('loading');
  protected readonly lesson = signal<GrammarLessonAdmin | null>(null);
  protected readonly point = signal<GrammarPoint | null>(null);
  protected readonly title = computed(() => this.point()?.titleVi ?? this.pointId);

  protected readonly busy = signal<Action | null>(null);
  protected readonly actionError = signal<string | null>(null);
  protected readonly note = signal('');
  protected readonly confirming = signal(false);
  protected readonly confirmingPublish = signal(false);
  /** Exercise whose reports are being closed. */
  protected readonly resolving = signal<string | null>(null);

  protected readonly checkText = computed(() => {
    const l = this.lesson();
    return l ? checkStatusText(l) : '';
  });
  /** Flags nobody confirmed yet. */
  protected readonly openFlags = computed(() => {
    const l = this.lesson();
    return l ? openChecks(l) : [];
  });
  protected readonly flagCount = computed(() => this.openFlags().length);
  protected readonly confirmingVerify = signal(false);
  protected readonly deleting = signal<string | null>(null);

  protected readonly optionSlots = [0, 1, 2, 3];
  protected readonly editing = signal<string | null>(null);
  protected readonly editKind = signal<GrammarExerciseKind>('choice');
  protected readonly editError = signal<string | null>(null);
  protected readonly editFields = signal<Record<string, string[]>>({});
  protected readonly editForm = new FormGroup({
    promptVi: new FormControl('', { nonNullable: true }),
    text: new FormControl('', { nonNullable: true }),
    o0: new FormControl('', { nonNullable: true }),
    o1: new FormControl('', { nonNullable: true }),
    o2: new FormControl('', { nonNullable: true }),
    o3: new FormControl('', { nonNullable: true }),
    answerIndex: new FormControl(0, { nonNullable: true }),
    answersText: new FormControl('', { nonNullable: true }),
    sentence: new FormControl('', { nonNullable: true }),
    wordsText: new FormControl('', { nonNullable: true }),
    explanationVi: new FormControl('', { nonNullable: true }),
  });
  private readonly checkById = computed(() => new Map((this.lesson()?.checks ?? []).map((c) => [c.exerciseId, c])));
  private readonly reportById = computed(() => new Map((this.lesson()?.reports ?? []).map((r) => [r.exerciseId, r])));

  protected readonly json = new FormControl('', { nonNullable: true });
  protected readonly form = new FormGroup({ json: this.json });
  protected readonly jsonError =signal<string | null>(null);
  protected readonly serverFields = signal<string[]>([]);
  protected readonly saveError = signal<string | null>(null);

  constructor() {
    this.load();
    this.loadPoint();
  }

  private load(): void {
    this.state.set('loading');
    this.api.grammarLesson(this.pointId).subscribe({
      next: (lesson) => {
        this.setLesson(lesson);
        this.state.set('ready');
      },
      error: (err: unknown) => this.state.set(err instanceof ApiError && err.status === 404 ? 'empty' : 'error'),
    });
  }

  /** The curriculum entry (name, pattern, hint); without it the page still works with the id. */
  private loadPoint(): void {
    const level = levelOf(this.pointId);
    if (!level) {
      return;
    }
    this.api.grammarPoints(level).subscribe({
      next: (points) => this.point.set(points.find((p) => p.id === this.pointId) ?? null),
      error: () => undefined,
    });
  }

  private setLesson(lesson: GrammarLessonAdmin): void {
    // Only the GET of one lesson carries the open reports; the other responses keep what we know.
    this.lesson.set({ ...lesson, reports: lesson.reports ?? this.lesson()?.reports ?? [] });
    this.json.setValue(formatContent(lesson.content));
    this.json.markAsPristine();
    this.jsonError.set(null);
    this.serverFields.set([]);
    this.saveError.set(null);
  }

  protected retry(): void {
    this.load();
  }

  protected reformat(): void {
    const parsed = this.parse();
    if (parsed !== null) {
      this.json.setValue(JSON.stringify(parsed, null, 2));
      this.json.markAsDirty();
    }
  }

  /** Parses the editor text; reports a syntax error or a non-object value and returns null. */
  private parse(): GrammarContent | null {
    this.jsonError.set(null);
    let value: unknown;
    try {
      value = JSON.parse(this.json.value);
    } catch (err) {
      const reason = err instanceof Error ? err.message : '';
      this.jsonError.set(`JSON sai cú pháp, chưa gửi lên máy chủ. ${reason}`.trim());
      return null;
    }
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
      this.jsonError.set('Nội dung phải là một đối tượng JSON { … }, chưa gửi lên máy chủ.');
      return null;
    }
    return value as GrammarContent;
  }

  protected askGenerate(): void {
    const lesson = this.lesson();
    if (this.busy()) {
      return;
    }
    if (lesson && (lesson.edited || lesson.status === 'published' || this.json.dirty)) {
      this.confirming.set(true);
      return;
    }
    void this.generate(false);
  }

  protected confirmGenerate(): void {
    this.confirming.set(false);
    void this.generate(true);
  }

  protected confirmMessage(): string {
    const lesson = this.lesson();
    const what =
      lesson?.status === 'published' ? 'đã đăng' : lesson?.edited ? 'đã sửa tay' : 'có thay đổi chưa lưu';
    return `Bài ngữ pháp này ${what}. Sinh lại sẽ thay toàn bộ nội dung bằng một bản nháp mới.`;
  }

  protected generate(force: boolean): Promise<void> {
    return this.run('generate', () => this.api.generateGrammarLesson(this.pointId, force), 'Đã sinh bản nháp mới.');
  }

  protected check(): Promise<void> {
    return this.run('check', () => this.api.checkGrammarLesson(this.pointId), 'Đã kiểm tra bằng AI.');
  }

  /** With flagged exercises left, ask first; the confirmation sends acknowledgeFlags. */
  protected askPublish(): void {
    if (this.busy()) {
      return;
    }
    if (this.flagCount() > 0) {
      this.confirmingPublish.set(true);
      return;
    }
    void this.publish(false);
  }

  protected confirmPublish(): void {
    this.confirmingPublish.set(false);
    void this.publish(true);
  }

  protected publishMessage(): string {
    return `Bài còn ${this.flagCount()} câu AI gắn cờ chưa xử lý. Đăng bây giờ người học sẽ thấy cả những câu này.`;
  }

  protected confirmFlag(exerciseId: string): Promise<void> {
    return this.run('confirm', () => this.api.confirmGrammarCheck(this.pointId, exerciseId), 'Đã xác nhận câu này đúng.');
  }

  /** Asks first when flags are left; confirming says they are all right. */
  protected askVerify(): void {
    if (this.busy()) {
      return;
    }
    if (this.flagCount() > 0) {
      this.confirmingVerify.set(true);
      return;
    }
    void this.verify();
  }

  protected confirmVerify(): void {
    this.confirmingVerify.set(false);
    void this.verify();
  }

  protected verifyMessage(): string {
    return `${this.flagCount()} câu còn cờ sẽ được tính là đúng. Chỉ xác nhận khi bạn đã tự xem các câu đó.`;
  }

  protected verify(): Promise<void> {
    return this.run('verify', () => this.api.verifyGrammarLesson(this.pointId), 'Đã xác nhận bài đã kiểm tra xong.');
  }

  /** "Luyện tập 2" / "Kiểm tra 1": where an exercise sits, for the list of flags. */
  protected exerciseLabel(id: string): string {
    const content = this.lesson()?.content;
    const practice = content?.practice.findIndex((e) => e.id === id) ?? -1;
    if (practice >= 0) {
      return `Luyện tập ${practice + 1}`;
    }
    const mastery = content?.mastery.findIndex((e) => e.id === id) ?? -1;
    return mastery >= 0 ? `Kiểm tra ${mastery + 1}` : id;
  }

  protected shortReason(check: GrammarCheck): string {
    return CHECK_SHORT[check.kind];
  }

  /** Scrolls to an exercise of the preview and moves focus there. */
  protected jump(exerciseId: string): void {
    const target = this.doc.getElementById(`ex-${exerciseId}`);
    target?.scrollIntoView?.({ block: 'center' });
    target?.focus();
  }

  protected startEdit(ex: GrammarExercise): void {
    if (this.busy()) {
      return;
    }
    const options = ex.options ?? [];
    this.editForm.setValue({
      promptVi: ex.promptVi ?? '',
      text: ex.text ?? '',
      o0: options[0] ?? '',
      o1: options[1] ?? '',
      o2: options[2] ?? '',
      o3: options[3] ?? '',
      answerIndex: ex.answerIndex ?? 0,
      answersText: (ex.answers ?? []).join('\n'),
      sentence: ex.sentence ?? '',
      wordsText: (ex.words ?? []).join(' '),
      explanationVi: ex.explanationVi ?? '',
    });
    this.editKind.set(ex.kind);
    this.editError.set(null);
    this.editFields.set({});
    this.editing.set(ex.id);
  }

  protected cancelEdit(): void {
    this.editing.set(null);
    this.editError.set(null);
    this.editFields.set({});
  }

  protected textLabel(): string {
    const kind = this.editKind();
    return kind === 'fill' ? 'Câu có chỗ trống (dùng ___)' : kind === 'reorder' ? 'Nghĩa tiếng Việt của câu' : 'Câu hỏi';
  }

  protected fieldErrors(name: string): string[] {
    return this.editFields()[name] ?? [];
  }

  private exerciseInput(): GrammarExerciseInput {
    const v = this.editForm.getRawValue();
    const kind = this.editKind();
    const base = { kind, promptVi: v.promptVi.trim() || undefined, text: v.text.trim(), explanationVi: v.explanationVi.trim() };
    const lines = (text: string) => text.split(/\r?\n/).map((x) => x.trim()).filter(Boolean);
    if (kind === 'choice') {
      return { ...base, options: [v.o0, v.o1, v.o2, v.o3].map((o) => o.trim()), answerIndex: Number(v.answerIndex) };
    }
    if (kind === 'fill') {
      return { ...base, answers: lines(v.answersText) };
    }
    return { ...base, sentence: v.sentence.trim(), words: v.wordsText.split(/\s+/).filter(Boolean) };
  }

  protected async saveEdit(): Promise<void> {
    const id = this.editing();
    if (!id || this.busy()) {
      return;
    }
    await this.run(
      'exercise',
      () => this.api.saveGrammarExercise(this.pointId, id, this.exerciseInput()),
      'Đã lưu câu. Nên bấm Kiểm tra bằng AI lại.',
    );
  }

  protected askDelete(id: string): void {
    if (!this.busy()) {
      this.deleting.set(id);
    }
  }

  protected deleteMessage(): string {
    const id = this.deleting();
    const content = this.lesson()?.content;
    const section = content?.practice.some((e) => e.id === id) ? content.practice : (content?.mastery ?? []);
    const name = content?.practice.some((e) => e.id === id) ? 'Bài luyện tập' : 'Bài kiểm tra';
    return `${name} còn ${Math.max(section.length - 1, 0)} câu sau khi xoá. Mỗi phần có số câu tối thiểu; nếu xuống dưới ngưỡng, máy chủ sẽ từ chối.`;
  }

  protected confirmDelete(): void {
    const id = this.deleting();
    this.deleting.set(null);
    if (id) {
      void this.run('delete', () => this.api.deleteGrammarExercise(this.pointId, id), 'Đã xoá câu. Nên bấm Kiểm tra bằng AI lại.');
    }
  }

  protected publish(acknowledgeFlags: boolean): Promise<void> {
    return this.run(
      'publish',
      () => this.api.publishGrammarLesson(this.pointId, acknowledgeFlags),
      'Đã đăng bài. Người học đã thấy bài này.',
    );
  }

  /** Closes the open reports of one exercise and drops them from the page. */
  protected async resolve(exerciseId: string): Promise<void> {
    if (this.resolving() || this.busy()) {
      return;
    }
    this.resolving.set(exerciseId);
    this.actionError.set(null);
    this.note.set('');
    try {
      await firstValueFrom(this.api.resolveGrammarReport(this.pointId, exerciseId));
      this.lesson.update((l) => (l ? { ...l, reports: (l.reports ?? []).filter((r) => r.exerciseId !== exerciseId) } : l));
      this.note.set('Đã đánh dấu báo lỗi là đã xử lý.');
    } catch (err) {
      this.actionError.set(grammarFailure(err, 'Không đánh dấu được, vui lòng thử lại.').message);
    } finally {
      this.resolving.set(null);
    }
  }

  protected unpublish(): Promise<void> {
    return this.run('unpublish', () => this.api.unpublishGrammarLesson(this.pointId), 'Đã gỡ bài, người học không còn thấy.');
  }

  protected async save(): Promise<void> {
    if (this.busy()) {
      return;
    }
    this.saveError.set(null);
    this.serverFields.set([]);
    const content = this.parse();
    if (!content) {
      return;
    }
    await this.run('save', () => this.api.saveGrammarLesson(this.pointId, content), 'Đã lưu nội dung.');
  }

  private async run(action: Action, call: () => ReturnType<AdminApiService['grammarLesson']>, done: string): Promise<void> {
    this.busy.set(action);
    this.actionError.set(null);
    this.saveError.set(null);
    this.serverFields.set([]);
    this.editError.set(null);
    this.editFields.set({});
    this.note.set('');
    try {
      const lesson = await firstValueFrom(call());
      this.setLesson(lesson);
      this.state.set('ready');
      this.note.set(done);
      if (action === 'exercise') {
        this.cancelEdit();
      }
    } catch (err) {
      const fallback =
        action === 'generate'
          ? 'Sinh bài thất bại, vui lòng thử lại.'
          : action === 'check'
            ? 'Kiểm tra thất bại, vui lòng thử lại.'
            : 'Không thực hiện được, vui lòng thử lại.';
      const failure = grammarFailure(err, fallback, action === 'generate' || action === 'check');
      if (action === 'exercise') {
        this.editError.set(failure.message);
        this.editFields.set(this.groupFields(failure.fields));
      } else if (action === 'delete') {
        this.actionError.set([failure.message, ...failure.fields].join(' '));
      } else if (action === 'save') {
        this.saveError.set(failure.message);
        this.serverFields.set(failure.fields);
      } else {
        this.actionError.set(failure.message);
        this.serverFields.set(failure.fields);
      }
    } finally {
      this.busy.set(null);
    }
  }

  /** "exercise.options: cần 4" lines into messages per form field ("_" for the rest). */
  private groupFields(lines: string[]): Record<string, string[]> {
    const out: Record<string, string[]> = {};
    for (const line of lines) {
      const at = line.indexOf(': ');
      const path = at < 0 ? '' : line.slice(0, at);
      const text = at < 0 ? line : line.slice(at + 2);
      const name = path.replace(/^exercise\./, '').replace(/\[.*$/, '');
      const key = EXERCISE_FIELDS.includes(name) ? name : '_';
      (out[key] ??= []).push(key === '_' && path ? `${path}: ${text}` : text);
    }
    return out;
  }

  protected kindLabel(ex: GrammarExercise): string {
    return EXERCISE_KIND_LABELS[ex.kind];
  }

  protected checkOf(ex: GrammarExercise): GrammarCheck | null {
    return this.checkById().get(ex.id) ?? null;
  }

  protected checkLabel(check: GrammarCheck): string {
    return CHECK_LABELS[check.kind];
  }

  protected reportsOf(ex: GrammarExercise): ExerciseReports | null {
    return this.reportById().get(ex.id) ?? null;
  }

  protected reasons(r: ExerciseReports): string {
    return reasonsText(r.reasons);
  }

  protected isCorrect(ex: GrammarExercise, index: number): boolean {
    return ex.answerIndex === index;
  }
}
