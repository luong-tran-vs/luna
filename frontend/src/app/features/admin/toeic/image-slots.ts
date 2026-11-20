import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';

import { serverUrl } from '../../../core/api-url';
import { ImageSlot } from '../../../core/models/toeic';
import { ImageEngine } from '../../../core/models/word-bank';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { EngineDialog } from '../../../shared/components/engine-dialog/engine-dialog';
import { errorBody } from '../json-import/json-import';

/**
 * The pictures of one group (FR-036…FR-038): the Part 1 photo or a picture for each Part 6/7 text.
 * Each one can be drawn by AI (Gemini or Codex, chosen every time), uploaded, or removed.
 */
@Component({
  selector: 'lu-toeic-image-slots',
  imports: [EngineDialog],
  template: `
    @if (slots(); as list) {
      @if (list.length === 0) {
        <p class="muted">Cụm này không có ô ảnh.</p>
      }
      <ul class="slots">
        @for (s of list; track s.slot) {
          <li class="slot">
            <div class="thumb">
              @if (s.has) {
                <img crossorigin="use-credentials" [src]="src(s)" [alt]="s.label" />
              } @else {
                <span class="none">Chưa có ảnh</span>
              }
            </div>
            <div class="info">
              <b>{{ s.label }}</b>
              <div class="ops">
                <button
                  type="button"
                  class="btn small"
                  [disabled]="busy() !== null"
                  (click)="asking.set(s.slot)"
                >
                  {{ busy() === s.slot ? 'Đang vẽ…' : s.has ? 'Vẽ lại bằng AI' : 'Vẽ bằng AI' }}
                </button>
                <label class="btn small upload" [class.disabled]="busy() !== null">
                  Tải ảnh lên
                  <input
                    type="file"
                    accept="image/png,image/jpeg,image/gif,image/webp"
                    class="sr-only"
                    [disabled]="busy() !== null"
                    (change)="upload(s.slot, $event)"
                  />
                </label>
                @if (s.has) {
                  <button
                    type="button"
                    class="btn small"
                    [disabled]="busy() !== null"
                    (click)="remove(s.slot)"
                  >
                    Xoá ảnh
                  </button>
                }
              </div>
            </div>
          </li>
        }
      </ul>
    } @else if (!error()) {
      <p class="muted" role="status">Đang tải ảnh…</p>
    }
    @if (error()) {
      <p class="form-alert" role="alert">{{ error() }}</p>
    }
    <lu-engine-dialog
      [open]="asking() !== null"
      heading="Vẽ ảnh bằng AI"
      [message]="message()"
      [codexAvailable]="codex()"
      (confirmed)="draw($event)"
      (cancelled)="asking.set(null)"
    />
  `,
  styleUrls: ['./toeic-admin.css'],
  styles: `
    .slots {
      display: grid;
      gap: var(--space-3);
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .slot {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-3);
      align-items: center;
    }
    .thumb {
      display: grid;
      place-items: center;
      width: 160px;
      aspect-ratio: 4 / 3;
      overflow: hidden;
      background: var(--color-bg);
      border: 1px dashed var(--color-line);
      border-radius: var(--radius-md);
    }
    .thumb img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
    .none {
      font: var(--text-xs) var(--font-ui);
      color: var(--color-text-muted);
    }
    .info {
      display: grid;
      flex: 1 1 200px;
      gap: var(--space-2);
      font: var(--text-sm) var(--font-ui);
    }
    .ops {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-1);
    }
    .upload {
      cursor: pointer;
    }
    .upload:focus-within {
      outline: 2px solid var(--color-primary);
      outline-offset: 2px;
    }
    .upload.disabled {
      pointer-events: none;
      opacity: 0.6;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ImageSlots implements OnInit {
  readonly groupKey = input.required<string>();
  /** Emitted after a picture was drawn, uploaded or removed. */
  readonly changed = output<void>();

  private readonly api = inject(ToeicApiService);
  protected readonly slots = signal<ImageSlot[] | null>(null);
  protected readonly codex = signal(false);
  protected readonly asking = signal<string | null>(null);
  protected readonly busy = signal<string | null>(null);
  protected readonly error = signal('');
  /** Changes after each update so the browser reloads the pictures. */
  private readonly version = signal(Date.now());

  ngOnInit(): void {
    this.api.groupImages(this.groupKey()).subscribe({
      next: (r) => this.apply(r.slots, r.codex),
      error: () => this.error.set('Không tải được ảnh của cụm.'),
    });
  }

  protected src(s: ImageSlot): string {
    return `${serverUrl(s.url)}?v=${this.version()}`;
  }

  protected message(): string {
    const slot = this.slots()?.find((s) => s.slot === this.asking());
    return slot?.slot === 'photo'
      ? 'Vẽ ảnh chụp theo mô tả cảnh và câu mô tả đúng (không gửi các câu sai). Ảnh cũ sẽ được thay.'
      : `Vẽ ảnh minh hoạ cho ${slot?.label ?? 'văn bản'}: không có chữ, không lộ đáp án. Ảnh cũ sẽ được thay.`;
  }

  protected draw(engine: ImageEngine): void {
    const slot = this.asking();
    this.asking.set(null);
    if (!slot) {
      return;
    }
    this.run(slot, this.api.drawImage(this.groupKey(), slot, engine));
  }

  protected upload(slot: string, event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) {
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      this.error.set('Ảnh tối đa 5 MB.');
      return;
    }
    this.run(slot, this.api.uploadImage(this.groupKey(), slot, file));
  }

  protected remove(slot: string): void {
    this.run(slot, this.api.deleteImage(this.groupKey(), slot));
  }

  private run(slot: string, req: ReturnType<ToeicApiService['groupImages']>): void {
    this.busy.set(slot);
    this.error.set('');
    req.subscribe({
      next: (r) => {
        this.busy.set(null);
        this.apply(r.slots, r.codex);
        this.changed.emit();
      },
      error: (err: unknown) => {
        this.busy.set(null);
        this.error.set(errorBody(err).message ?? 'Chưa làm được, vui lòng thử lại.');
      },
    });
  }

  private apply(slots: ImageSlot[], codex: boolean): void {
    this.slots.set(slots);
    this.codex.set(codex);
    this.version.set(Date.now());
  }
}
