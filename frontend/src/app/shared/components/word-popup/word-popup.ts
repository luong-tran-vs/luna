import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  input,
  output,
  signal,
} from '@angular/core';

import { LookupResult } from '../../../core/models/reading';

export type PopupState =
  | { kind: 'loading' }
  | { kind: 'result'; result: LookupResult }
  | { kind: 'not-found' };

const POS_NAMES: Record<string, string> = {
  N: 'danh từ',
  V: 'động từ',
  A: 'tính từ',
  ADJ: 'tính từ',
  ADV: 'trạng từ',
  R: 'trạng từ',
  P: 'đại từ',
  PREP: 'giới từ',
  C: 'liên từ',
  I: 'thán từ',
};

const SOURCE_LABELS = { ai: 'AI · theo ngữ cảnh', dictionary: 'Từ điển' } as const;

export const MAX_MEANING = 200;

let nextId = 0;

/** Popup content for a looked-up word or phrase. Positioning is done by the reading page. */
@Component({
  selector: 'lu-word-popup',
  templateUrl: './word-popup.html',
  styleUrl: './word-popup.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WordPopup {
  readonly text = input.required<string>();
  readonly state = input.required<PopupState>();
  readonly saved = input(false);
  readonly saving = input(false);
  readonly error = input<string | null>(null);

  readonly closed = output<void>();
  /** Save with the shown meaning, or with a meaning typed by the learner. */
  readonly save = output<string | undefined>();
  /** The learner wants to hear the word (named listen: "play" is a DOM media event). */
  readonly listen = output<void>();

  protected readonly titleId = `word-popup-title-${nextId++}`;
  protected readonly manualMeaning = signal('');
  protected readonly manualError = signal<string | null>(null);

  protected readonly result = computed(() => {
    const s = this.state();
    return s.kind === 'result' ? s.result : null;
  });
  protected readonly sourceLabel = computed(() => {
    const r = this.result();
    return r ? SOURCE_LABELS[r.source] : '';
  });
  protected readonly showLemma = computed(() => {
    const r = this.result();
    return !!r && r.lemma.toLowerCase() !== this.text().trim().toLowerCase();
  });

  constructor() {
    // Move focus into the popup when it opens so keyboard and screen-reader users land on it.
    const host = inject<ElementRef<HTMLElement>>(ElementRef);
    afterNextRender(() => host.nativeElement.querySelector<HTMLElement>('.popup')?.focus());
  }

  protected posName(code: string): string {
    return POS_NAMES[code.toUpperCase()] ?? code;
  }

  protected saveManual(): void {
    const meaning = this.manualMeaning().trim();
    if (!meaning) {
      this.manualError.set('Vui lòng nhập nghĩa');
      return;
    }
    if (meaning.length > MAX_MEANING) {
      this.manualError.set(`Nghĩa tối đa ${MAX_MEANING} ký tự`);
      return;
    }
    this.manualError.set(null);
    this.save.emit(meaning);
  }
}
