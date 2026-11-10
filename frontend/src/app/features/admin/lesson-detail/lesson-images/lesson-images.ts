import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import { EnginePicker } from '../../../../shared/components/engine-picker/engine-picker';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { serverUrl } from '../../../../core/api-url';
import { ApiError } from '../../../../core/interceptors/error-interceptor';
import {
  IMAGE_STYLE_PLACEHOLDER,
  MAX_IMAGE_STYLE,
  MAX_SCENE_DESCRIPTION,
  MAX_UPLOAD_BYTES,
  SCENE_KEY,
  UPLOAD_TYPES,
} from '../../../../core/models/generate';
import { ImageWord, LessonImages as ImageState, SceneBatch } from '../../../../core/models/lesson';
import { engineLabel, ImageEngine } from '../../../../core/models/word-bank';
import { AdminApiService } from '../../admin-api.service';
import { SceneBatchService } from '../../curriculum/scene-batch.service';

const SAVE_FAILED = 'Không lưu được, vui lòng thử lại.';

/**
 * F23: the pictures of a lesson. The AI draws only the scene picture, when the admin asks, with
 * Gemini or the Codex CLI of the server (chosen as in the word bank, since 2026-11-06). Word
 * pictures are drawn in the word bank; here the admin can only upload, link or remove a lesson's
 * own picture of a word. The server scales the pictures down.
 */
@Component({
  selector: 'lu-lesson-images',
  imports: [NgTemplateOutlet, ReactiveFormsModule, EnginePicker],
  templateUrl: './lesson-images.html',
  styleUrl: './lesson-images.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonImages {
  private readonly api = inject(AdminApiService);
  private readonly scenes = inject(SceneBatchService);

  readonly lessonId = input.required<string>();
  /** The lesson's title, named in the banner of its scene batch. */
  readonly lessonTitle = input('');

  protected readonly maxStyle = MAX_IMAGE_STYLE;
  protected readonly maxScene = MAX_SCENE_DESCRIPTION;
  protected readonly stylePlaceholder = IMAGE_STYLE_PLACEHOLDER;
  protected readonly sceneKey = SCENE_KEY;
  protected readonly state = signal<ImageState | null>(null);
  protected readonly loadError = signal(false);
  /** What the scene form is doing: saving the settings, drawing the picture, or nothing. */
  protected readonly busy = signal<'save' | 'draw' | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly note = signal<string | null>(null);
  /** The word whose picture is being uploaded or removed, or null. */
  protected readonly busyWord = signal<string | null>(null);
  /** Result of the last upload or removal, shown under that word. */
  protected readonly wordNote = signal<{ lemma: string; text: string; error: boolean } | null>(
    null,
  );
  /** Changes after each upload so the browser shows the new picture under the same URL. */
  private readonly version = signal(Date.now());
  protected readonly accept = UPLOAD_TYPES.join(',');
  /** The word whose "Dán link ảnh" field is open, or null; one at a time. */
  protected readonly linkFor = signal<string | null>(null);

  /** The engine chosen; none at first: the admin picks one each time (asked 2026-11-10). */
  protected readonly imageEngine = signal<ImageEngine | ''>('');
  /** What "Sinh ảnh cảnh" draws with: Codex only where the server offers it; '' while none is chosen. */
  protected readonly engine = computed<ImageEngine | ''>(() => {
    const e = this.imageEngine();
    return e === 'codex' && !this.state()?.codexImages ? '' : e;
  });
  /** The optional fields are shown on request, or when they already hold a saved value. */
  protected readonly showScene = signal(false);
  protected readonly showStyle = signal(false);

  protected readonly form = new FormGroup({
    sceneDescription: new FormControl('', {
      nonNullable: true,
      validators: [Validators.maxLength(MAX_SCENE_DESCRIPTION)],
    }),
    style: new FormControl('', {
      nonNullable: true,
      validators: [Validators.maxLength(MAX_IMAGE_STYLE)],
    }),
  });

  private readonly values = toSignal(this.form.valueChanges, { initialValue: this.form.value });
  /** Lengths shown in the counters of the two text fields. */
  protected readonly sceneLength = computed(() => (this.values().sceneDescription ?? '').length);
  protected readonly styleLength = computed(() => (this.values().style ?? '').length);

  /** The scene picture as a row of the picture list: uploaded, linked and removed like a word's. */
  protected readonly sceneItem = computed<ImageWord | null>(() => {
    const s = this.state();
    return s ? { lemma: SCENE_KEY, meaningVi: '', imageUrl: s.sceneImageUrl } : null;
  });

  /** How a picture is named in the labels read aloud and in the notes. */
  protected nameOf(w: ImageWord): string {
    return w.lemma === SCENE_KEY ? 'ảnh cảnh' : `từ ${w.lemma}`;
  }

  /** The key of this lesson's scene batch. */
  protected readonly batchKey = computed(() => `lesson:${this.lessonId()}`);
  /** This lesson's scene is being drawn in the background. */
  protected readonly drawingNow = computed(() => {
    const b = this.scenes.status();
    return !!b?.running && b.key === this.batchKey();
  });
  /** Another scene batch runs (one runs at a time): what it is for, or null. */
  protected readonly otherBatch = computed(() => {
    const b = this.scenes.status();
    return b?.running && b.key !== this.batchKey() ? b : null;
  });

  constructor() {
    effect(() => {
      const id = this.lessonId();
      untracked(() => void this.load(id));
    });
    inject(DestroyRef).onDestroy(this.scenes.watch());
    // When this lesson's batch ends, the new picture is read (or the reason shown).
    let wasDrawing = false;
    effect(() => {
      const drawing = this.drawingNow();
      if (wasDrawing && !drawing) {
        const b = untracked(() => this.scenes.status());
        if (b) {
          untracked(() => this.finished(b));
        }
      }
      wasDrawing = drawing;
    });
  }

  protected async load(id = this.lessonId()): Promise<void> {
    this.loadError.set(false);
    try {
      this.apply(await firstValueFrom(this.api.lessonImages(id)));
    } catch {
      this.loadError.set(true);
    }
  }

  protected thumb(w: ImageWord): string {
    return `${serverUrl(w.imageUrl)}?v=${this.version()}`;
  }

  protected setEngine(value: string): void {
    this.imageEngine.set(value === 'codex' ? 'codex' : 'gemini');
  }

  /** A file was chosen for a word: checked here, then scaled down and stored by the server. */
  protected async upload(w: ImageWord, event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = ''; // choosing the same file again fires change again
    if (!file || this.busyWord() !== null) {
      return;
    }
    if (!UPLOAD_TYPES.includes(file.type)) {
      this.wordNote.set({ lemma: w.lemma, text: 'Chỉ nhận ảnh JPEG, PNG hoặc GIF.', error: true });
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      this.wordNote.set({ lemma: w.lemma, text: 'Ảnh tối đa 5 MB.', error: true });
      return;
    }
    await this.changeImage(w, 'Đã tải ảnh lên.', () =>
      this.api.uploadWordImage(this.lessonId(), w.lemma, file),
    );
  }

  protected toggleLink(w: ImageWord): void {
    this.linkFor.update((l) => (l === w.lemma ? null : w.lemma));
    this.wordNote.set(null);
  }

  /** Lấy ảnh: the server downloads the picture behind the link, scales it down and stores it. */
  protected async importLink(w: ImageWord, input: HTMLInputElement): Promise<void> {
    const url = input.value.trim();
    if (this.busyWord() !== null) {
      return;
    }
    if (!/^https?:\/\/\S+$/i.test(url)) {
      this.wordNote.set({
        lemma: w.lemma,
        text: 'Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://',
        error: true,
      });
      return;
    }
    await this.changeImage(w, 'Đã lấy ảnh từ link.', () =>
      this.api.importWordImage(this.lessonId(), w.lemma, url),
    );
    if (!this.wordNote()?.error) {
      this.linkFor.set(null);
    }
  }

  protected async remove(w: ImageWord): Promise<void> {
    if (this.busyWord() === null) {
      await this.changeImage(w, 'Đã xoá ảnh.', () =>
        this.api.deleteWordImage(this.lessonId(), w.lemma),
      );
    }
  }

  private async changeImage(
    w: ImageWord,
    done: string,
    call: () => ReturnType<AdminApiService['deleteWordImage']>,
  ): Promise<void> {
    this.busyWord.set(w.lemma);
    this.wordNote.set(null);
    try {
      const s = await firstValueFrom(call());
      this.version.set(Date.now());
      // Only the picture list changes: a description being typed stays in the form.
      this.state.set(s);
      this.wordNote.set({ lemma: w.lemma, text: done, error: false });
    } catch (err) {
      this.wordNote.set({ lemma: w.lemma, text: messageOf(err) ?? SAVE_FAILED, error: true });
    } finally {
      this.busyWord.set(null);
    }
  }

  /** Lưu: keeps the description and style of the scene picture, without drawing it. */
  protected async save(): Promise<void> {
    if (!this.check()) {
      return;
    }
    const v = this.form.getRawValue();
    await this.run('save', 'Đã lưu mô tả ảnh cảnh.', () =>
      this.api.setLessonImages(this.lessonId(), {
        sceneDescription: v.sceneDescription.trim(),
        style: v.style.trim(),
      }),
    );
  }

  /**
   * Tạo ảnh cảnh: saves the form, then has the server draw the scene in the background, as a batch
   * of this one lesson (since 2026-11-06): the request answers at once, so no time limit of the
   * server or Nginx cuts it, and the drawing goes on when the admin leaves the page.
   */
  protected async draw(): Promise<void> {
    if (!this.check() || this.drawingNow() || this.otherBatch()) {
      return;
    }
    const v = this.form.getRawValue();
    const engine = this.engine();
    if (engine === '') {
      this.error.set('Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.');
      return;
    }
    this.busy.set('draw');
    this.error.set(null);
    this.note.set(null);
    try {
      this.apply(
        await firstValueFrom(
          this.api.setLessonImages(this.lessonId(), {
            sceneDescription: v.sceneDescription.trim(),
            style: v.style.trim(),
          }),
        ),
      );
      const b = await firstValueFrom(
        this.api.startSceneBatch({
          lessonIds: [this.lessonId()],
          replace: true,
          engine,
          key: this.batchKey(),
          label: `bài “${this.lessonTitle() || 'này'}”`,
        }),
      );
      this.scenes.set(b);
      this.note.set(
        `Máy chủ đang vẽ ảnh cảnh ngầm bằng ${engineLabel(engine)}. Có thể rời trang, ảnh tự hiện khi vẽ xong.`,
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu tạo ảnh cảnh được, vui lòng thử lại.');
    } finally {
      this.busy.set(null);
    }
  }

  /** A batch of this lesson ended: shows the new picture, or why there is none. */
  private finished(b: SceneBatch): void {
    void this.load();
    this.version.set(Date.now());
    const failure = b.failures[0]?.message ?? b.skipped[0]?.message ?? b.stopMessage;
    if (b.saved.length > 0) {
      this.error.set(null);
      this.note.set(`Đã tạo ảnh cảnh bằng ${this.batchEngine(b)}.`);
    } else {
      this.note.set(null);
      this.error.set(
        b.stopped && !failure
          ? 'Đã dừng tạo ảnh cảnh.'
          : (failure ?? 'Sinh ảnh thất bại, vui lòng thử lại.'),
      );
    }
  }

  protected batchEngine(b: SceneBatch): string {
    return engineLabel(b.engine === 'codex' ? 'codex' : 'gemini');
  }

  /** The form is valid, or the error says what is too long. */
  private check(): boolean {
    if (this.busy() !== null) {
      return false;
    }
    if (this.form.controls.sceneDescription.invalid) {
      this.error.set(`Mô tả cảnh tối đa ${MAX_SCENE_DESCRIPTION} ký tự`);
      return false;
    }
    if (this.form.controls.style.invalid) {
      this.error.set(`Mô tả ảnh tối đa ${MAX_IMAGE_STYLE} ký tự`);
      return false;
    }
    return true;
  }

  private async run(
    kind: 'save' | 'draw',
    done: string,
    call: () => ReturnType<AdminApiService['setLessonImages']>,
  ): Promise<boolean> {
    this.busy.set(kind);
    this.error.set(null);
    this.note.set(null);
    try {
      this.apply(await firstValueFrom(call()));
      this.note.set(done);
      return true;
    } catch (err) {
      this.error.set(
        messageOf(err) ?? (kind === 'draw' ? 'Sinh ảnh thất bại, vui lòng thử lại.' : SAVE_FAILED),
      );
      return false;
    } finally {
      this.busy.set(null);
    }
  }

  private apply(s: ImageState): void {
    this.state.set(s);
    this.form.reset({ sceneDescription: s.sceneDescription ?? '', style: s.style ?? '' });
    if (s.sceneDescription) {
      this.showScene.set(true);
    }
    if (s.style) {
      this.showStyle.set(true);
    }
  }
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field =
      body?.fields?.['imageStyle'] ??
      body?.fields?.['sceneDescription'] ??
      body?.fields?.['engine'] ??
      body?.fields?.['image'] ??
      body?.fields?.['url'];
    if (field) {
      return field;
    }
    return typeof body?.message === 'string' && body.message ? body.message : null;
  }
  return null;
}
