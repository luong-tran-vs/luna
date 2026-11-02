import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import {
  autoMark,
  breakAnswer,
  breakAnswerText,
  PatternPart,
  sentenceParts,
  SentencePart,
  splitLines,
} from '../grammar-logic';

/** One line of an example: the English in runs coloured like the pattern, and its meaning. */
export interface ExampleLine {
  sentence: SentencePart[];
  vi: string;
}

/**
 * An example as shown: one line, or a question then its answer ("Are you twenty? – No, I'm not.")
 * each with its own meaning. `en` is the whole sentence, read aloud.
 */
export interface ShownExample {
  en: string;
  lines: ExampleLine[];
}

/**
 * Coloured from its marks, or else from the words its structure's pattern names; without a pattern
 * the sentence stays plain.
 */
export function shownExample(
  en: string,
  vi: string,
  marked?: string,
  parts: readonly PatternPart[] = [],
): ShownExample {
  const runs = marked ? sentenceParts(en, marked, parts) : autoMark(en, parts);
  const english = splitLines(breakAnswer(runs));
  const meanings = breakAnswerText(vi).split('\n');
  // The meanings go with their lines when both split alike, else all under the last line.
  const paired = meanings.length === english.length;
  return {
    en,
    lines: english.map((sentence, i) => ({
      sentence,
      vi: paired ? meanings[i] : i === english.length - 1 ? meanings.join(' ') : '',
    })),
  };
}

/**
 * An example sentence, or a question then its answer, each line in English with its coloured words
 * and its Vietnamese meaning under it; the answer set in under the question.
 */
@Component({
  selector: 'lu-example-lines',
  template: `
    @for (line of lines(); track $index; let first = $first) {
      <div class="line" [class.answer]="!first">
        <p lang="en" class="en">
          @for (r of line.sentence; track $index) {
            <span [class]="r.tone === null ? '' : 'hl tone-' + r.tone">{{ r.text }}</span>
          }
        </p>
        @if (line.vi) {
          <p class="vi">{{ line.vi }}</p>
        }
      </div>
    }
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-2);
    }

    .answer {
      padding-left: var(--space-3);
      border-left: 3px solid var(--color-line);
    }

    .en {
      font: var(--text-base) var(--font-ui);
      font-weight: 500;
    }

    .vi {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }

    /* The words coloured like the piece of the pattern they stand for; the first tone (the subject) is
       only coloured, so a sentence carries at most a few soft patches. */
    .hl {
      padding: 0 2px;
      border-radius: var(--radius-sm);
      font-weight: 600;
      color: var(--color-tone-0);
    }

    .hl.tone-1 {
      color: var(--color-tone-1);
      background: var(--color-tone-1-soft);
    }

    .hl.tone-2 {
      color: var(--color-tone-2);
      background: var(--color-tone-2-soft);
    }

    .hl.tone-3 {
      color: var(--color-tone-3);
      background: var(--color-tone-3-soft);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ExampleLines {
  readonly lines = input.required<readonly ExampleLine[]>();
}
