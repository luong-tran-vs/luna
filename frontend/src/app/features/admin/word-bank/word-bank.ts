import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed, toObservable } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { catchError, debounceTime, distinctUntilChanged, EMPTY, firstValueFrom, switchMap, tap } from 'rxjs';

import {
  BankMissing,
  BankWord,
  MAX_BANK_IPA,
  MAX_BANK_LEMMA,
  MAX_BANK_MEANING,
} from '../../../core/models/word-bank';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { AdminApiService } from '../admin-api.service';
import { bankMessage } from './bank-message';
import { BankWordRow } from './bank-word-row/bank-word-row';

/** Wait after the last keystroke before searching. */
const SEARCH_DELAY_MS = 300;

const LOAD_FAILED = 'Không tải được kho từ vựng. Vui lòng thử lại.';

export const MISSING_OPTIONS: readonly { value: BankMissing; label: string }[] = [
  { value: '', label: 'Tất cả' },
  { value: 'image', label: 'Thiếu ảnh' },
  { value: 'ipa', label: 'Thiếu phiên âm' },
  { value: 'meaning', label: 'Thiếu nghĩa' },
];

/**
 * F24: the shared word bank. One entry per word with its meaning, IPA and picture, used by every
 * lesson: a lesson word without a picture of its own shows the bank's, and the bank's IPA comes
 * before the dictionary's. The admin searches, filters what is missing, adds words (IPA and
 * meaning filled from the dictionary when left empty) or imports every topic word.
 */
@Component({
  selector: 'lu-word-bank',
  imports: [ReactiveFormsModule, BankWordRow, ConfirmDialog],
  templateUrl: './word-bank.html',
  styleUrl: './word-bank.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WordBank {
  private readonly api = inject(AdminApiService);

  protected readonly missingOptions = MISSING_OPTIONS;
  protected readonly search = signal('');
  protected readonly missing = signal<BankMissing>('');
  private readonly reloads = signal(0);

  protected readonly words = signal<BankWord[] | null>(null);
  protected readonly total = signal(0);
  protected readonly hasMore = signal(false);
  private page = 1;
  protected readonly loadError = signal<string | null>(null);
  protected readonly loadingMore = signal(false);

  protected readonly adding = signal(false);
  protected readonly addError = signal<string | null>(null);
  protected readonly importing = signal(false);
  protected readonly note = signal<string | null>(null);
  protected readonly deleting = signal<BankWord | null>(null);
  protected readonly deleteMessage = computed(() => {
    const w = this.deleting();
    return w
      ? `Từ "${w.lemma}" sẽ bị xoá khỏi kho cùng ảnh của nó. Các bài học vẫn giữ nghĩa và ảnh riêng của bài.`
      : '';
  });

  protected readonly maxLemma = MAX_BANK_LEMMA;
  protected readonly form = new FormGroup({
    lemma: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required, Validators.maxLength(MAX_BANK_LEMMA)],
    }),
    meaningVi: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_BANK_MEANING)] }),
    ipa: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(MAX_BANK_IPA)] }),
  });

  constructor() {
    const query = computed(() => ({ q: this.search().trim(), missing: this.missing(), reload: this.reloads() }));
    toObservable(query)
      .pipe(
        debounceTime(SEARCH_DELAY_MS),
        distinctUntilChanged((a, b) => a.q === b.q && a.missing === b.missing && a.reload === b.reload),
        tap(() => this.loadError.set(null)),
        switchMap(({ q, missing }) =>
          this.api.bankWords(q, missing, 1).pipe(
            catchError(() => {
              this.loadError.set(LOAD_FAILED);
              return EMPTY;
            }),
          ),
        ),
        takeUntilDestroyed(),
      )
      .subscribe((p) => {
        this.page = 1;
        this.words.set(p.words);
        this.total.set(p.total);
        this.hasMore.set(p.hasMore);
      });
  }

  protected reload(): void {
    this.reloads.update((n) => n + 1);
  }

  protected setSearch(value: string): void {
    this.search.set(value);
  }

  protected setMissing(value: string): void {
    this.missing.set(value as BankMissing);
  }

  protected async loadMore(): Promise<void> {
    if (this.loadingMore()) {
      return;
    }
    this.loadingMore.set(true);
    try {
      const p = await firstValueFrom(this.api.bankWords(this.search().trim(), this.missing(), this.page + 1));
      this.page++;
      this.words.update((ws) => [...(ws ?? []), ...p.words]);
      this.total.set(p.total);
      this.hasMore.set(p.hasMore);
    } catch {
      this.loadError.set(LOAD_FAILED);
    } finally {
      this.loadingMore.set(false);
    }
  }

  protected async add(): Promise<void> {
    if (this.adding()) {
      return;
    }
    const lemma = this.form.controls.lemma;
    if (this.form.invalid || !lemma.value.trim()) {
      this.addError.set(
        lemma.value.trim() ? `Từ tối đa ${MAX_BANK_LEMMA} ký tự, nghĩa tối đa ${MAX_BANK_MEANING}, phiên âm tối đa ${MAX_BANK_IPA}.` : 'Vui lòng nhập từ',
      );
      return;
    }
    const v = this.form.getRawValue();
    this.adding.set(true);
    this.addError.set(null);
    this.note.set(null);
    try {
      const w = await firstValueFrom(
        this.api.addBankWord({ lemma: v.lemma.trim(), meaningVi: v.meaningVi.trim(), ipa: v.ipa.trim() }),
      );
      this.form.reset();
      this.note.set(`Đã thêm "${w.lemma}".`);
      this.reload();
    } catch (err) {
      this.addError.set(bankMessage(err) ?? 'Không thêm được, vui lòng thử lại.');
    } finally {
      this.adding.set(false);
    }
  }

  /** Adds every topic word that is not in the bank yet. */
  protected async importTopics(): Promise<void> {
    if (this.importing()) {
      return;
    }
    this.importing.set(true);
    this.note.set(null);
    try {
      const added = await firstValueFrom(this.api.importBankWords());
      this.note.set(added ? `Đã thêm ${added} từ từ các chủ đề.` : 'Mọi từ của các chủ đề đã có trong kho.');
      if (added) {
        this.reload();
      }
    } catch (err) {
      this.note.set(bankMessage(err) ?? 'Không nhập được, vui lòng thử lại.');
    } finally {
      this.importing.set(false);
    }
  }

  protected replace(w: BankWord): void {
    this.words.update((ws) => ws?.map((x) => (x.lemma === w.lemma ? w : x)) ?? null);
  }

  protected async confirmDelete(): Promise<void> {
    const w = this.deleting();
    this.deleting.set(null);
    if (!w) {
      return;
    }
    try {
      await firstValueFrom(this.api.deleteBankWord(w.lemma));
      this.words.update((ws) => ws?.filter((x) => x.lemma !== w.lemma) ?? null);
      this.total.update((n) => Math.max(0, n - 1));
      this.note.set(`Đã xoá "${w.lemma}".`);
    } catch (err) {
      this.note.set(bankMessage(err) ?? 'Không xoá được, vui lòng thử lại.');
    }
  }
}
