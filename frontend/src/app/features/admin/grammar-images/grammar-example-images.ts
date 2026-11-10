import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  input,
  output,
  signal,
  untracked,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { serverUrl } from '../../../core/api-url';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import { GrammarImageBatch, GrammarLessonAdmin } from '../../../core/models/grammar-admin';
import { engineLabel, ImageEngine } from '../../../core/models/word-bank';
import { EnginePicker } from '../../../shared/components/engine-picker/engine-picker';
import { AdminApiService } from '../admin-api.service';
import { GrammarImageBatchService, pointBatchKey } from './grammar-image-batch.service';

/** An example of the lesson as the picture list shows it. */
interface ExampleRow {
  index: number;
  en: string;
  vi: string;
  /** The admin URL of its picture; '' for none. */
  image: string;
  /** The exercise id for the picture of a picture exercise; '' for an example. */
  exercise: string;
  /** The key of its picture, from the URL; '' for none. */
  key: string;
}

/**
 * The pictures of a grammar lesson's examples (added 2026-11-06), in its admin page: one per
 * example sentence, shown beside it in "Cấu trúc & ví dụ". The server draws them in the background,
 * with Gemini or the Codex CLI (as the word bank and the lesson scenes); the admin may draw one
 * again or remove it.
 */
@Component({
  selector: 'lu-grammar-example-images',
  imports: [EnginePicker],
  templateUrl: './grammar-example-images.html',
  styleUrl: './grammar-example-images.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarExampleImages {
  private readonly api = inject(AdminApiService);
  private readonly batches = inject(GrammarImageBatchService);

  readonly lesson = input.required<GrammarLessonAdmin>();
  /** The point's title, named in the banner of its batch. */
  readonly title = input('');
  /** The point can be shown in pictures; otherwise no picture is drawn (added 2026-11-06). */
  readonly drawable = input(true);
  /** The lesson after a picture was removed. */
  readonly changed = output<GrammarLessonAdmin>();
  /** The lesson's batch ended: the page reads the lesson again for the new pictures. */
  readonly reload = output<void>();

  protected readonly busy = signal<string | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly note = signal<string | null>(null);
  /** Changes when pictures are drawn, so the browser asks for them again under the same URL. */
  private readonly version = signal(Date.now());

  /** No engine is chosen at first: the admin picks one each time (asked 2026-11-10). */
  private readonly imageEngine = signal<ImageEngine | ''>('');
  protected readonly codex = computed(
    () => !!this.lesson().codexImages || !!this.batches.status()?.codexImages,
  );
  protected readonly engine = computed<ImageEngine | ''>(() => {
    const e = this.imageEngine();
    return e === 'codex' && !this.codex() ? '' : e;
  });

  /** The examples, then the picture exercises (added 2026-11-06), each with its picture. */
  protected readonly rows = computed<ExampleRow[]>(() => {
    const l = this.lesson();
    const images = l.exampleImages ?? [];
    const examples = l.content.examples.map((e, i) => {
      const image = images[i] ?? '';
      return { index: i, en: e.en, vi: e.vi, image, key: image ? keyOf(image) : '', exercise: '' };
    });
    const pictures = [...l.content.practice, ...l.content.mastery]
      .filter((e) => e.kind === 'picture')
      .map((e, k) => {
        const image = e.imageUrl ?? '';
        return {
          index: examples.length + k,
          en: e.text ?? '',
          vi: e.picture ?? '',
          image,
          key: image ? keyOf(image) : '',
          exercise: e.id,
        };
      });
    return [...examples, ...pictures];
  });
  protected readonly missing = computed(() => this.rows().filter((r) => !r.image).length);

  private readonly batchKey = computed(() => pointBatchKey(this.lesson().pointId));
  /** This lesson's pictures are being drawn in the background. */
  protected readonly drawingNow = computed(() => {
    const b = this.batches.status();
    return !!b?.running && b.key === this.batchKey();
  });
  /** The example being drawn now, by its sentence. */
  protected readonly drawingEn = computed(() =>
    this.drawingNow() ? (this.batches.status()?.drawing?.en ?? '') : '',
  );
  protected readonly batch = computed(() => (this.drawingNow() ? this.batches.status() : null));
  /** Another picture batch runs (one runs at a time). */
  protected readonly otherBatch = computed(() => {
    const b = this.batches.status();
    return b?.running && b.key !== this.batchKey() ? b : null;
  });
  protected readonly locked = computed(
    () => this.busy() !== null || this.drawingNow() || !!this.otherBatch(),
  );

  constructor() {
    inject(DestroyRef).onDestroy(this.batches.watch());
    // When this lesson's batch ends, the page reads the lesson again, and the result is shown.
    let wasDrawing = false;
    effect(() => {
      const drawing = this.drawingNow();
      if (wasDrawing && !drawing) {
        const b = untracked(() => this.batches.status());
        untracked(() => this.finished(b));
      }
      wasDrawing = drawing;
    });
  }

  protected thumb(r: ExampleRow): string {
    return `${serverUrl(r.image)}?v=${this.version()}`;
  }

  protected setEngine(engine: ImageEngine): void {
    this.imageEngine.set(engine);
  }

  /** Tạo ảnh cho các câu còn thiếu: every example without a picture, one after another. */
  protected drawMissing(): Promise<void> {
    return this.start('all', {});
  }

  /** Vẽ lại one example, replacing its picture. */
  protected redraw(r: ExampleRow): Promise<void> {
    return this.start(`row-${r.index}`, { keys: [r.key], replace: true });
  }

  private async start(busy: string, extra: { keys?: string[]; replace?: boolean }): Promise<void> {
    if (this.locked()) {
      return;
    }
    const engine = this.engine();
    if (engine === '') {
      this.error.set('Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.');
      return;
    }
    this.busy.set(busy);
    this.error.set(null);
    this.note.set(null);
    try {
      const b = await firstValueFrom(
        this.api.startGrammarImageBatch({
          pointIds: [this.lesson().pointId],
          engine,
          key: this.batchKey(),
          label: `bài “${this.title() || this.lesson().pointId}”`,
          ...extra,
        }),
      );
      this.batches.set(b);
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không bắt đầu tạo ảnh được, vui lòng thử lại.');
    } finally {
      this.busy.set(null);
    }
  }

  protected async remove(r: ExampleRow): Promise<void> {
    if (this.locked() || !r.key) {
      return;
    }
    this.busy.set(`row-${r.index}`);
    this.error.set(null);
    this.note.set(null);
    try {
      this.changed.emit(
        await firstValueFrom(this.api.deleteGrammarImage(this.lesson().pointId, r.key)),
      );
      this.note.set('Đã xoá ảnh.');
    } catch (err) {
      this.error.set(messageOf(err) ?? 'Không xoá được ảnh, vui lòng thử lại.');
    } finally {
      this.busy.set(null);
    }
  }

  private finished(b: GrammarImageBatch | null): void {
    this.version.set(Date.now());
    this.reload.emit();
    if (!b) {
      return;
    }
    const engine = engineLabel(b.engine === 'codex' ? 'codex' : 'gemini');
    const failed = b.failures.length ? `, ${b.failures.length} câu lỗi` : '';
    if (b.saved.length > 0) {
      this.note.set(`Đã tạo ${b.saved.length}/${b.total} ảnh bằng ${engine}${failed}.`);
    }
    const reason = b.stopMessage ?? b.failures[0]?.message;
    if (b.saved.length === 0 || b.stopMessage) {
      this.error.set(
        reason ?? (b.stopped ? 'Đã dừng tạo ảnh.' : 'Sinh ảnh thất bại, vui lòng thử lại.'),
      );
    }
  }

  protected async stop(): Promise<void> {
    try {
      this.batches.set(await firstValueFrom(this.api.stopGrammarImageBatch()));
    } catch {
      // The next status read shows where it is.
    }
  }
}

/** The key of a picture, the last part of its URL. */
function keyOf(url: string): string {
  return decodeURIComponent(url.slice(url.lastIndexOf('/') + 1));
}

/** The server's Vietnamese message, when the error has one. */
function messageOf(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field = body?.fields ? Object.values(body.fields)[0] : undefined;
    if (field) {
      return field;
    }
    return typeof body?.message === 'string' && body.message ? body.message : null;
  }
  return null;
}
