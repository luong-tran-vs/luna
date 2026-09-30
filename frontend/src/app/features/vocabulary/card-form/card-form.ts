import { ChangeDetectionStrategy, Component, effect, inject, input, output, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Card, CardDetails } from '../../../core/models/vocab';
import { VocabApiService } from '../../../core/services/vocab-api.service';

type FieldName = 'text' | 'meaningVi' | 'ipa' | 'contextSentence';

const REQUIRED: Partial<Record<FieldName, string>> = { text: 'Vui lòng nhập từ', meaningVi: 'Vui lòng nhập nghĩa' };

/** Element ids of the inputs; error messages use "<id>-error". */
const INPUT_ID: Record<FieldName, string> = {
  text: 'card-text',
  meaningVi: 'card-meaning',
  ipa: 'card-ipa',
  contextSentence: 'card-example',
};

/** Server field names to form controls (the lemma is derived from the word). */
const SERVER_FIELD: Record<string, FieldName> = {
  text: 'text',
  lemma: 'text',
  meaningVi: 'meaningVi',
  ipa: 'ipa',
  contextSentence: 'contextSentence',
};

/** Adds a card by hand, or edits the meaning, IPA and example of an existing card. */
@Component({
  selector: 'lu-card-form',
  imports: [ReactiveFormsModule],
  templateUrl: './card-form.html',
  styleUrl: './card-form.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CardForm {
  private readonly api = inject(VocabApiService);

  /** The card to edit; null adds a new card. */
  readonly card = input<Card | null>(null);
  readonly saved = output<Card>();
  readonly cancelled = output<void>();

  protected readonly form = inject(NonNullableFormBuilder).group({
    text: ['', [Validators.required, Validators.maxLength(100)]],
    meaningVi: ['', [Validators.required, Validators.maxLength(200)]],
    ipa: ['', Validators.maxLength(100)],
    contextSentence: ['', Validators.maxLength(1000)],
  });
  protected readonly submitted = signal(false);
  protected readonly pending = signal(false);
  protected readonly serverError = signal<string | null>(null);
  protected readonly serverFields = signal<Partial<Record<FieldName, string>>>({});

  constructor() {
    effect(() => {
      const c = this.card();
      this.form.reset({
        text: c?.text ?? '',
        meaningVi: c?.meaningVi ?? '',
        ipa: c?.ipa ?? '',
        contextSentence: c?.contextSentence ?? '',
      });
      if (c) {
        this.form.controls.text.disable();
      } else {
        this.form.controls.text.enable();
      }
    });
  }

  protected error(name: FieldName): string | null {
    const c = this.form.controls[name];
    if (c.enabled && (c.touched || this.submitted())) {
      if (c.hasError('required')) {
        return REQUIRED[name] ?? null;
      }
      if (c.hasError('maxlength')) {
        return `Tối đa ${(c.getError('maxlength') as { requiredLength: number }).requiredLength} ký tự`;
      }
    }
    return this.serverFields()[name] ?? null;
  }

  protected describedBy(name: FieldName): string | null {
    return this.error(name) ? `${INPUT_ID[name]}-error` : null;
  }

  protected async submit(): Promise<void> {
    this.submitted.set(true);
    this.serverError.set(null);
    this.serverFields.set({});
    if (this.form.invalid || this.pending()) {
      this.form.markAllAsTouched();
      return;
    }
    const v = this.form.getRawValue();
    const values = {
      meaningVi: v.meaningVi.trim(),
      ipa: v.ipa.trim(),
      contextSentence: v.contextSentence.trim(),
    };
    const existing = this.card();

    let request;
    if (existing) {
      const changes: CardDetails = {};
      for (const key of ['meaningVi', 'ipa', 'contextSentence'] as const) {
        if (values[key] !== existing[key]) {
          changes[key] = values[key];
        }
      }
      if (Object.keys(changes).length === 0) {
        this.saved.emit(existing);
        return;
      }
      request = this.api.update(existing.id, changes);
    } else {
      const text = v.text.trim().replace(/\s+/g, ' ');
      request = this.api.saveCard({ text, lemma: text.toLowerCase(), ...values, source: 'manual' });
    }

    this.pending.set(true);
    try {
      this.saved.emit(await firstValueFrom(request));
    } catch (err) {
      this.showError(err);
    } finally {
      this.pending.set(false);
    }
  }

  private showError(err: unknown): void {
    const body = err instanceof ApiError ? (err.body as { message?: string; fields?: Record<string, string> } | null) : null;
    if (err instanceof ApiError && err.status === 409) {
      this.serverFields.set({ text: 'Từ này đã có trong sổ' });
    } else if (body?.fields) {
      const fields: Partial<Record<FieldName, string>> = {};
      for (const [key, message] of Object.entries(body.fields)) {
        const name = SERVER_FIELD[key];
        if (name) {
          fields[name] = message;
        }
      }
      this.serverFields.set(fields);
    } else if (err instanceof ApiError && err.kind === 'network') {
      this.serverError.set('Không kết nối được máy chủ. Vui lòng thử lại.');
    } else {
      this.serverError.set(body?.message ?? 'Có lỗi xảy ra, vui lòng thử lại.');
    }
  }
}
