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
import { SpeakButton } from '../../directives/speak-button';

export type PopupState =
  | { kind: 'loading' }
  | { kind: 'result'; result: LookupResult }
  | { kind: 'not-found' };

/** Parts of speech of the offline dictionary (english-dictionary.db) and of AI answers. */
const POS_NAMES: Record<string, string> = {
  N: 'danh từ',
  V: 'động từ',
  A: 'tính từ',
  ADJ: 'tính từ',
  ADV: 'trạng từ',
  R: 'trạng từ',
  D: 'trạng từ',
  P: 'đại từ',
  PREP: 'giới từ',
  E: 'giới từ',
  C: 'liên từ',
  I: 'thán từ',
  O: 'thán từ',
  M: 'số từ',
};

const SOURCE_LABELS = { ai: 'AI · theo ngữ cảnh', dictionary: 'Từ điển' } as const;

export const MAX_MEANING = 200;

/** How many meanings a lookup shows (the server sends them spread over the parts of speech). */
const MAX_MEANINGS = 4;

/** "Chuyến đi" → "chuyến đi" for a meaning joined after another; "TV" stays as it is. */
function lowerFirst(s: string): string {
  const [first, second] = s;
  return second && second === second.toLowerCase() ? first.toLowerCase() + s.slice(1) : s;
}

let nextId = 0;

/** Popup content for a looked-up word or phrase. Positioning is done by the reading page. */
@Component({
  selector: 'lu-word-popup',
  imports: [SpeakButton],
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
  /** F9: AI asks left in the lesson for this selection; null when unlimited or already asked. */
  readonly askLeft = input<number | null>(null);
  readonly askLimit = input(0);
  /** An "Hỏi AI" request is running (F9). */
  readonly asking = input(false);
  readonly askError = input<string | null>(null);

  readonly closed = output<void>();
  /** Save with the shown meaning, or with a meaning typed by the learner. */
  readonly save = output<string | undefined>();
  /** The learner wants to hear the word (named listen: "play" is a DOM media event). */
  readonly listen = output<void>();
  /** The learner asks the AI what the selection means in its sentence (F9). */
  readonly ask = output<void>();

  protected readonly titleId = `word-popup-title-${nextId++}`;
  protected readonly manualMeaning = signal('');
  protected readonly manualError = signal<string | null>(null);

  protected readonly result = computed(() => {
    const s = this.state();
    return s.kind === 'result' ? s.result : null;
  });
  /** What the play button reads: the dictionary form when found, else the selected text. */
  protected readonly spoken = computed(() => this.result()?.lemma ?? this.text());
  protected readonly sourceLabel = computed(() => {
    const r = this.result();
    return r ? SOURCE_LABELS[r.source] : '';
  });
  /** Offered when nothing was found or the meaning does not come from the AI. */
  protected readonly canAsk = computed(() => {
    const s = this.state();
    return s.kind === 'not-found' || (s.kind === 'result' && s.result.source !== 'ai');
  });
  /**
   * The meanings shown, one line per part of speech in order of first appearance: "Sự chạy." and
   * "Chuyến đi." (both nouns) give "danh từ Sự chạy; chuyến đi".
   */
  protected readonly meaningLines = computed(() => {
    const lines: { pos: string; parts: string[] }[] = [];
    for (const m of this.result()?.meanings.slice(0, MAX_MEANINGS) ?? []) {
      const text = m.text.trim().replace(/[.;]+$/, '');
      if (!text) {
        continue;
      }
      const pos = this.posName(m.pos);
      const line = lines.find((l) => l.pos === pos);
      if (line) {
        line.parts.push(lowerFirst(text));
      } else {
        lines.push({ pos, parts: [text] });
      }
    }
    return lines.map((l) => ({ pos: l.pos, text: l.parts.join('; ') }));
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

  /** The Vietnamese name of a part of speech; '' (nothing shown) for none or an unknown code. */
  protected posName(code: string): string {
    return POS_NAMES[code.toUpperCase()] ?? '';
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
