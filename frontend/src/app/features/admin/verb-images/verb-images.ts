import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  signal,
} from '@angular/core';

import { Observable } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { VerbDrawItem, VerbImageList } from '../../../core/models/verb-images';
import { engineLabel, ImageEngine, MAX_IMAGE_BATCH } from '../../../core/models/word-bank';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { Loading } from '../../../shared/components/loading/loading';
import { IRREGULAR_VERBS, IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';
import { VERB_EXAMPLES, VERB_LESSONS } from '../../irregular-verbs/verb-lessons.data';
import { VerbImageBatchService } from './verb-image-batch.service';
import { VerbImagesApiService } from './verb-images-api.service';

/** Which verbs the page lists. */
type Filter = 'all' | 'missing' | 'done';

const VERBS = new Map(IRREGULAR_VERBS.map((v) => [v.base, v]));
const BASES = IRREGULAR_VERBS.map((v) => v.base);

/** The server's Vietnamese message for a failed request, when it has one. */
function serverMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const field = Object.values(body?.fields ?? {})[0];
    if (field) {
      return field;
    }
    if (typeof body?.message === 'string' && body.message) {
      return body.message;
    }
  }
  return fallback;
}

/** What the server is asked to draw for a verb: its meaning and example keep it on the verb sense. */
function drawItem(v: IrregularVerb): VerbDrawItem {
  return { base: v.base, meaningVi: v.meaningVi, sentence: VERB_EXAMPLES[v.base]?.en ?? '' };
}

/**
 * "Ảnh động từ" (added 2026-11-10): one picture per irregular verb, of its sense as a verb, shown on
 * the cards of the learners' "Động từ" tab. The pictures are kept apart from the word bank's (a noun
 * spelt alike may have one there, which can be reused when it shows the verb). The verbs are listed
 * by lesson; the admin draws the missing ones in one batch, or one again, choosing each time between
 * Gemini and the Codex CLI (no default, asked 2026-11-10); a picture can also be uploaded.
 */
@Component({
  selector: 'lu-verb-images',
  imports: [Loading, EngineDialog, ConfirmDialog],
  templateUrl: './verb-images.html',
  styleUrl: './verb-images.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbImages {
  private readonly api = inject(VerbImagesApiService);
  protected readonly batches = inject(VerbImageBatchService);

  protected readonly data = signal<VerbImageList | null>(null);
  protected readonly loadError = signal(false);
  protected readonly message = signal<string | null>(null);
  protected readonly filter = signal<Filter>('all');
  protected readonly query = signal('');
  /** Verbs whose upload, copy or removal is under way. */
  protected readonly busy = signal<ReadonlySet<string>>(new Set());

  /** The verbs the engine dialog is about to draw; null while it is closed. */
  protected readonly toDraw = signal<IrregularVerb[] | null>(null);
  /** The verb whose picture the admin asked to remove. */
  protected readonly toRemove = signal<string | null>(null);

  protected readonly total = BASES.length;
  protected readonly images = computed(
    () => new Map((this.data()?.images ?? []).map((i) => [i.base, i.imageUrl])),
  );
  protected readonly bank = computed(
    () => new Map((this.data()?.bank ?? []).map((b) => [b.base, b])),
  );
  protected readonly withImage = computed(() => BASES.filter((b) => this.images().has(b)).length);
  protected readonly missing = computed(() =>
    IRREGULAR_VERBS.filter((v) => !this.images().has(v.base)),
  );
  protected readonly drawing = computed(() => new Set(this.batches.status()?.drawing ?? []));
  protected readonly running = this.batches.running;
  protected readonly status = this.batches.status;
  /** The batch the page shows: running, or finished and not hidden yet. */
  protected readonly shown = computed(() => {
    const b = this.status();
    return b?.running ? b : this.batches.result();
  });
  protected readonly engineName = computed(() => {
    const e = this.status()?.engine;
    return e ? engineLabel(e) : '';
  });
  protected readonly percent = computed(() => {
    const b = this.status();
    return b && b.total ? Math.round((b.done / b.total) * 100) : 0;
  });

  /** The lessons with the verbs the filter and the search keep. */
  protected readonly lessons = computed(() => {
    const f = this.filter();
    const q = this.query().trim().toLowerCase();
    const images = this.images();
    return VERB_LESSONS.map((l) => ({
      lesson: l,
      verbs: l.bases
        .map((b) => VERBS.get(b))
        .filter((v): v is IrregularVerb => !!v)
        .filter((v) =>
          f === 'missing' ? !images.has(v.base) : f === 'done' ? images.has(v.base) : true,
        )
        .filter(
          (v) =>
            !q ||
            v.base.includes(q) ||
            v.past.includes(q) ||
            v.participle.includes(q) ||
            v.meaningVi.toLowerCase().includes(q),
        ),
    })).filter((g) => g.verbs.length > 0);
  });

  protected readonly drawHeading = computed(() => {
    const list = this.toDraw() ?? [];
    return list.length === 1 ? `Vẽ ảnh cho "${list[0].base}"` : `Vẽ ${list.length} ảnh còn thiếu`;
  });
  protected readonly drawMessage = computed(() => {
    const list = this.toDraw() ?? [];
    if (list.length === 1) {
      const has = this.images().has(list[0].base);
      return `${has ? 'Ảnh đang có sẽ được thay bằng ảnh mới. ' : ''}AI vẽ theo nghĩa động từ "${list[0].meaningVi}" và câu ví dụ của từ.`;
    }
    const rest = this.missing().length - list.length;
    return (
      `Máy chủ vẽ lần lượt ${list.length} động từ chưa có ảnh, chạy ngầm, xong ảnh nào lưu ảnh đó. Bạn có thể rời trang.` +
      (rest > 0 ? ` Mỗi lượt tối đa ${MAX_IMAGE_BATCH} ảnh: ${rest} động từ còn lại vẽ ở lượt sau.` : '')
    );
  });

  constructor() {
    this.load();
    inject(DestroyRef).onDestroy(this.batches.watch());
    // A picture saved by the batch shows at once: reload when the batch saves one more.
    let saved = -1;
    effect(() => {
      const n = this.status()?.saved.length ?? 0;
      if (saved >= 0 && n !== saved) {
        this.load();
      }
      saved = n;
    });
  }

  protected load(): void {
    this.api.list(BASES).subscribe({
      next: (d) => {
        this.data.set(d);
        this.loadError.set(false);
      },
      error: () => this.loadError.set(true),
    });
  }

  protected setFilter(f: Filter): void {
    this.filter.set(f);
  }

  protected onSearch(event: Event): void {
    this.query.set((event.target as HTMLInputElement).value);
  }

  protected askDrawMissing(): void {
    this.message.set(null);
    // One batch draws at most MAX_IMAGE_BATCH verbs, in lesson-list order; the next click draws the rest.
    this.toDraw.set(this.missing().slice(0, MAX_IMAGE_BATCH));
  }

  protected askDraw(v: IrregularVerb): void {
    this.message.set(null);
    this.toDraw.set([v]);
  }

  /** Starts the batch with the engine the admin picked in the dialog. */
  protected draw(engine: ImageEngine): void {
    const list = this.toDraw() ?? [];
    this.toDraw.set(null);
    if (list.length === 0) {
      return;
    }
    this.api.startBatch({ items: list.map(drawItem), engine }).subscribe({
      next: (b) => this.batches.set(b),
      error: (err) =>
        this.message.set(serverMessage(err, 'Không bắt đầu tạo ảnh được, vui lòng thử lại.')),
    });
  }

  protected stop(): void {
    this.api.stopBatch().subscribe({
      next: (b) => this.batches.set(b),
      error: () => this.message.set('Không dừng được, vui lòng thử lại.'),
    });
  }

  protected upload(base: string, event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) {
      return;
    }
    this.run(base, this.api.upload(base, file), 'Không tải ảnh lên được.');
  }

  protected useBank(base: string): void {
    this.run(base, this.api.fromBank(base), 'Không chép được ảnh từ Kho từ vựng.');
  }

  protected remove(): void {
    const base = this.toRemove();
    this.toRemove.set(null);
    if (base) {
      this.run(base, this.api.remove(base), 'Không xoá được ảnh.');
    }
  }

  private run(base: string, req: Observable<unknown>, fail: string): void {
    this.message.set(null);
    this.busy.update((s) => new Set(s).add(base));
    const done = () =>
      this.busy.update((s) => {
        const next = new Set(s);
        next.delete(base);
        return next;
      });
    req.subscribe({
      next: () => {
        done();
        this.load();
      },
      error: (err: unknown) => {
        done();
        this.message.set(serverMessage(err, fail));
      },
    });
  }
}
