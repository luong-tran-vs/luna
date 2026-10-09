import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { firstValueFrom, Observable } from 'rxjs';

import { serverUrl } from '../../../../core/api-url';
import { MAX_UPLOAD_BYTES, UPLOAD_TYPES } from '../../../../core/models/generate';
import { BankWord, MAX_BANK_IPA, MAX_BANK_MEANING } from '../../../../core/models/word-bank';
import { AdminApiService } from '../../admin-api.service';
import { bankMessage } from '../bank-message';

/** What the row is doing; only one action at a time. */
type Busy = 'save' | 'image' | 'draw' | null;

/**
 * F24: one word of the bank. The admin edits its meaning and IPA, and uploads, links, draws with
 * the AI (one paid request) or removes its picture. Changes go to the server at once; the parent
 * gets the updated word.
 */
@Component({
  selector: 'lu-bank-word-row',
  imports: [ReactiveFormsModule],
  templateUrl: './bank-word-row.html',
  styleUrl: './bank-word-row.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'bank-word', '[attr.aria-busy]': 'busy() !== null' },
})
export class BankWordRow {
  private readonly api = inject(AdminApiService);

  readonly word = input.required<BankWord>();
  /** The word as saved by the server. */
  readonly changed = output<BankWord>();
  /** The admin asked to delete the word; the parent confirms. */
  readonly removeRequested = output<BankWord>();

  protected readonly imageSrc = computed(() => serverUrl(this.word().imageUrl));
  protected readonly accept = UPLOAD_TYPES.join(',');
  protected readonly busy = signal<Busy>(null);
  protected readonly editing = signal(false);
  protected readonly linkOpen = signal(false);
  protected readonly note = signal<{ text: string; error: boolean } | null>(null);

  protected readonly form = new FormGroup({
    meaningVi: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_BANK_MEANING)] }),
    ipa: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_BANK_IPA)] }),
  });

  protected id(part: string): string {
    return `bank-${part}-${this.word().lemma.replace(/[^a-z0-9]+/gi, '-')}`;
  }

  protected startEdit(): void {
    const w = this.word();
    this.form.reset({ meaningVi: w.meaningVi, ipa: w.ipa });
    this.note.set(null);
    this.editing.set(true);
  }

  protected cancelEdit(): void {
    this.editing.set(false);
  }

  protected async save(): Promise<void> {
    if (this.busy() !== null) {
      return;
    }
    if (this.form.invalid) {
      this.note.set({
        text: `Nghĩa tối đa ${MAX_BANK_MEANING} ký tự, phiên âm tối đa ${MAX_BANK_IPA} ký tự.`,
        error: true,
      });
      return;
    }
    const v = this.form.getRawValue();
    const ok = await this.run('save', 'Đã lưu.', () =>
      this.api.updateBankWord(this.word().lemma, { meaningVi: v.meaningVi.trim(), ipa: v.ipa.trim() }),
    );
    if (ok) {
      this.editing.set(false);
    }
  }

  /** A file was chosen: checked here, then scaled down and stored by the server. */
  protected async upload(event: Event): Promise<void> {
    const el = event.target as HTMLInputElement;
    const file = el.files?.[0];
    el.value = ''; // choosing the same file again fires change again
    if (!file || this.busy() !== null) {
      return;
    }
    if (!UPLOAD_TYPES.includes(file.type)) {
      this.note.set({ text: 'Chỉ nhận ảnh JPEG, PNG hoặc GIF.', error: true });
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      this.note.set({ text: 'Ảnh tối đa 5 MB.', error: true });
      return;
    }
    await this.run('image', 'Đã tải ảnh lên.', () => this.api.uploadBankImage(this.word().lemma, file));
  }

  protected toggleLink(): void {
    this.linkOpen.update((o) => !o);
    this.note.set(null);
  }

  /** Lấy ảnh: the server downloads the picture behind the link. */
  protected async importLink(el: HTMLInputElement): Promise<void> {
    const url = el.value.trim();
    if (this.busy() !== null) {
      return;
    }
    if (!/^https?:\/\/\S+$/i.test(url)) {
      this.note.set({ text: 'Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://', error: true });
      return;
    }
    if (await this.run('image', 'Đã lấy ảnh từ link.', () => this.api.importBankImage(this.word().lemma, url))) {
      this.linkOpen.set(false);
    }
  }

  protected async draw(): Promise<void> {
    if (this.busy() === null) {
      await this.run('draw', 'Đã sinh ảnh bằng AI.', () => this.api.generateBankImage(this.word().lemma));
    }
  }

  protected async removeImage(): Promise<void> {
    if (this.busy() === null) {
      await this.run('image', 'Đã xoá ảnh.', () => this.api.deleteBankImage(this.word().lemma));
    }
  }

  private async run(kind: Busy, done: string, call: () => Observable<BankWord>): Promise<boolean> {
    this.busy.set(kind);
    this.note.set(null);
    try {
      this.changed.emit(await firstValueFrom(call()));
      this.note.set({ text: done, error: false });
      return true;
    } catch (err) {
      this.note.set({ text: bankMessage(err) ?? 'Không lưu được, vui lòng thử lại.', error: true });
      return false;
    } finally {
      this.busy.set(null);
    }
  }
}
