import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { Mistake } from '../../../core/models/writing';
import { wordDiff } from '../../../shared/utils/word-diff';
import { MistakeTag } from './mistake-tag';

/** One run of the diff, its trailing space apart so the marks stop at the word. */
export interface MarkedPart {
  kind: 'same' | 'add' | 'del';
  word: string;
  gap: string;
  /** The number of the mistake this change is (from 1), on the last run of the change; null for none. */
  mark: number | null;
}

const norm = (s: string) =>
  ` ${s
    .toLowerCase()
    .replace(/[^\p{L}\p{N}'\s]/gu, ' ')
    .replace(/\s+/g, ' ')
    .trim()} `;

/** Whether one run of words holds the other (" really " in " really like "). */
const overlaps = (a: string, b: string) => a.trim() !== '' && b.trim() !== '' && (a.includes(b) || b.includes(a));

/**
 * The word diff of text → corrected with each change numbered after the mistake it is: a change
 * takes the first mistake not yet used whose words it shares, else one already used (the same
 * mistake made twice keeps its number). Changes matching no mistake stay unnumbered.
 */
export function markCorrections(text: string, corrected: string, mistakes: Mistake[]): MarkedPart[] {
  const parts: MarkedPart[] = wordDiff(text, corrected).map((p) => {
    const word = p.text.trimEnd();
    return { kind: p.kind, word, gap: p.text.slice(word.length), mark: null };
  });
  if (mistakes.length === 0) {
    return parts;
  }
  const keys = mistakes.map((m) => ({ original: norm(m.original), corrected: norm(m.corrected) }));
  const used = new Set<number>();
  let i = 0;
  while (i < parts.length) {
    if (parts[i].kind === 'same') {
      i++;
      continue;
    }
    let j = i;
    let del = '';
    let add = '';
    for (; j < parts.length && parts[j].kind !== 'same'; j++) {
      if (parts[j].kind === 'del') {
        del += ` ${parts[j].word}`;
      } else {
        add += ` ${parts[j].word}`;
      }
    }
    const d = norm(del);
    const a = norm(add);
    const fits = (k: number) => overlaps(a, keys[k].corrected) || overlaps(d, keys[k].original);
    const all = keys.map((_, k) => k);
    const k = all.find((n) => !used.has(n) && fits(n)) ?? all.find((n) => fits(n));
    if (k !== undefined) {
      used.add(k);
      parts[j - 1].mark = k + 1;
    }
    i = j;
  }
  return parts;
}

/**
 * The learner's text with the corrections marked in it: struck in red what was wrong, in green
 * the right words. With tagged mistakes (grades since 2026-10-22) each change carries its number
 * and the mistakes are listed below with a tag of their type and the note.
 */
@Component({
  selector: 'lu-corrections',
  imports: [MistakeTag],
  template: `
    <p class="text diff" lang="en">
      @for (part of parts(); track $index) {
        @switch (part.kind) {
          @case ('add') {
            <ins><span class="visually-hidden">{{ addLabel() }} </span>{{ part.word }}</ins>
          }
          @case ('del') {
            <del><span class="visually-hidden">{{ delLabel() }} </span>{{ part.word }}</del>
          }
          @default {
            <span>{{ part.word }}</span>
          }
        }
        @if (part.mark) {
          <sup class="mark"><span class="visually-hidden">(lỗi </span>{{ part.mark }}<span class="visually-hidden">)</span></sup>
        }
        <span>{{ part.gap }}</span>
      }
    </p>
    @if (mistakes().length) {
      <ol class="mistakes" aria-label="Các lỗi đã sửa">
        @for (m of mistakes(); track $index) {
          <li>
            <lu-mistake-tag [type]="m.type" />
            <sup class="mark"><span class="visually-hidden">, lỗi </span>{{ $index + 1 }}</sup>
            <span class="note">
              @if (m.noteVi) {
                {{ m.noteVi }}
              } @else {
                <span lang="en">{{ m.original || '…' }} → {{ m.corrected }}</span>
              }
            </span>
          </li>
        }
      </ol>
    }
  `,
  styleUrl: './corrections.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Corrections {
  readonly text = input.required<string>();
  readonly corrected = input.required<string>();
  readonly mistakes = input<Mistake[]>([]);
  /** Read before an added / removed word by screen readers. */
  readonly addLabel = input('sửa thành:');
  readonly delLabel = input('bỏ:');

  protected readonly parts = computed(() => markCorrections(this.text(), this.corrected(), this.mistakes()));
}
