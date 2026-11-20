import { DOCUMENT, NgTemplateOutlet } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  input,
  output,
} from '@angular/core';

import { serverUrl } from '../../../core/api-url';
import { DocBlock, Ref, ToeicDoc } from '../../../core/models/toeic';
import { Mark, Segment, splitText } from './doc-text.logic';
import { blockId, Highlight, highlightsOf } from './highlight.logic';

interface ShownBlock {
  id: string;
  block: DocBlock;
  segments: Segment[];
  items: Segment[][];
}

/** A highlight to add or remove. */
export interface HighlightEdit {
  highlight: Highlight;
  remove: boolean;
}

/** The kinds drawn with a header of From / To / Subject lines. */
const LETTER_KINDS = ['email', 'letter', 'memo'];

/**
 * A text of Part 6/7 laid out like the real thing by its kind (FR-038): an email or memo with its
 * header lines, a web page with an address bar, a chat as bubbles, tables as tables. The blanks of
 * Part 6 show their question's number. In the exam room the learner can highlight words in yellow
 * (FR-009); after submitting, the evidence is marked.
 */
@Component({
  selector: 'lu-toeic-doc',
  imports: [NgTemplateOutlet],
  templateUrl: './doc-view.html',
  styleUrl: './doc-view.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocView {
  readonly doc = input.required<ToeicDoc>();
  readonly groupKey = input.required<string>();
  readonly docIndex = input(0);
  /** The shown number of each blank [[n]] of Part 6. */
  readonly blankNumbers = input<Partial<Record<number, number>>>({});
  /** The blank being answered (Part 6), drawn stronger. */
  readonly activeBlank = input<number | null>(null);
  /** The words to mark as evidence (answer detail). */
  readonly evidence = input<readonly Ref[]>([]);
  readonly highlights = input<readonly Highlight[]>([]);
  /** Whether the learner may highlight words. */
  readonly highlightable = input(false);
  readonly showTranslation = input(false);

  readonly highlightChange = output<HighlightEdit>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly document = inject(DOCUMENT);

  protected readonly letterLike = computed(() => LETTER_KINDS.includes(this.doc().kind));
  protected readonly url = computed(
    () => this.doc().meta.find((m) => m.k.toLowerCase() === 'url')?.v ?? null,
  );
  protected readonly metaLines = computed(() =>
    this.doc().meta.filter((m) => m.k.toLowerCase() !== 'url'),
  );
  protected readonly imageSrc = computed(() => {
    const u = this.doc().imageUrl;
    return u ? serverUrl(u) : null;
  });
  protected readonly kindLabel = computed(() => KIND_LABELS[this.doc().kind] ?? this.doc().kind);

  protected readonly blocks = computed<ShownBlock[]>(() => {
    const key = this.groupKey();
    const di = this.docIndex();
    const ev = this.evidence();
    const hl = this.highlights();
    return this.doc().body.map((block, bi) => {
      const id = blockId(key, di, bi);
      const marks: Mark[] = [
        ...ev.filter((r) => r.doc === di && r.block === bi).map((r) => ({ text: r.text, kind: 'evidence' as const })),
        ...highlightsOf(hl, id).map((text) => ({ text, kind: 'user' as const })),
      ];
      return {
        id,
        block,
        segments: block.text ? splitText(block.text, marks) : [],
        items: (block.items ?? []).map((it) => splitText(it, marks)),
      };
    });
  });

  /** Highlights (or clears) the words selected inside one block of this text. */
  protected mark(remove: boolean): void {
    const sel = this.document.getSelection();
    const text = sel?.toString() ?? '';
    const node = sel?.anchorNode ?? null;
    const el = node instanceof Element ? node : node?.parentElement;
    const blockEl = el?.closest<HTMLElement>('[data-block]');
    if (!text.trim() || !blockEl || !this.host.nativeElement.contains(blockEl)) {
      return;
    }
    this.highlightChange.emit({ highlight: { block: blockEl.dataset['block'] ?? '', text }, remove });
    sel?.removeAllRanges();
  }
}

const KIND_LABELS: Record<string, string> = {
  email: 'E-mail',
  letter: 'Thư',
  memo: 'Bản ghi nhớ',
  notice: 'Thông báo',
  ad: 'Quảng cáo',
  article: 'Bài báo',
  web: 'Trang web',
  invoice: 'Hoá đơn',
  form: 'Biểu mẫu',
  schedule: 'Lịch trình',
  chat: 'Tin nhắn',
  review: 'Đánh giá',
};
