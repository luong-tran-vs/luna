import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import {
  DEFAULT_IMAGE_STYLE,
  MAX_IMAGE_STYLE,
  MAX_UPLOAD_BYTES,
  UPLOAD_TYPES,
} from '../../../../core/models/generate';
import { ImageWord, LessonImages as ImageState } from '../../../../core/models/lesson';
import { AdminApiService } from '../../admin-api.service';
import { StatusChip } from '../../status-chip/status-chip';

const SAVE_FAILED = 'Không lưu được, vui lòng thử lại.';

/**
 * F23: the pictures of the lesson's vocabulary words. The admin turns the AI drawing on or off and
 * sets its style (saving with pictures on draws the words that have none, so it also retries a
 * failed drawing), and can upload, replace or remove the picture of each word. The server scales
 * uploaded pictures down; the AI never draws over them.
 */
@Component({
  selector: 'lu-lesson-images',
  imports: [ReactiveFormsModule, StatusChip],
  templateUrl: './lesson-images.html',
  styleUrl: './lesson-images.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LessonImages {
  private readonly api = inject(AdminApiService);

  readonly lessonId = input.required<string>();

  protected readonly maxStyle = MAX_IMAGE_STYLE;
  protected readonly state = signal<ImageState | null>(null);
  protected readonly loadError = signal(false);
  protected readonly saving = signal(false);
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

  protected readonly form = new FormGroup({
    enabled: new FormControl(false, { nonNullable: true }),
    style: new FormControl(DEFAULT_IMAGE_STYLE, {
      nonNullable: true,
      validators: [Validators.maxLength(MAX_IMAGE_STYLE)],
    }),
  });

  protected readonly status = computed(() => {
    const s = this.state()?.status;
    return s ? s : null;
  });

  constructor() {
    effect(() => {
      const id = this.lessonId();
      untracked(() => void this.load(id));
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
    return `${w.imageUrl}?v=${this.version()}`;
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
      // Only the word list changes: an unsaved style being typed stays in the form.
      this.state.set(s);
      this.wordNote.set({ lemma: w.lemma, text: done, error: false });
    } catch (err) {
      this.wordNote.set({ lemma: w.lemma, text: messageOf(err) ?? SAVE_FAILED, error: true });
    } finally {
      this.busyWord.set(null);
    }
  }

  /** The "Sinh ảnh cho từ vựng" switch; nothing is saved until Lưu. */
  protected toggle(): void {
    const c = this.form.controls.enabled;
    c.setValue(!c.value);
  }

  protected async save(): Promise<void> {
    if (this.saving()) {
      return;
    }
    if (this.form.controls.style.invalid) {
      this.error.set(`Mô tả ảnh tối đa ${MAX_IMAGE_STYLE} ký tự`);
      return;
    }
    const v = this.form.getRawValue();
    this.saving.set(true);
    this.error.set(null);
    this.note.set(null);
    try {
      this.apply(
        await firstValueFrom(
          this.api.setLessonImages(this.lessonId(), { enabled: v.enabled, style: v.style.trim() }),
        ),
      );
      this.note.set(
        v.enabled ? 'Đã lưu. Ảnh của các từ còn thiếu đang được sinh.' : 'Đã tắt ảnh từ vựng.',
      );
    } catch (err) {
      this.error.set(messageOf(err) ?? SAVE_FAILED);
    } finally {
      this.saving.set(false);
    }
  }

  private apply(s: ImageState): void {
    this.state.set(s);
    this.form.reset({ enabled: s.enabled, style: s.style || DEFAULT_IMAGE_STYLE });
  }
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field = body?.fields?.['imageStyle'] ?? body?.fields?.['image'];
    if (field) {
      return field;
    }
    return typeof body?.message === 'string' && body.message ? body.message : null;
  }
  return null;
}
