import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { MISTAKE_LABELS, MistakeType } from '../../../core/models/writing';

/** Colour family of each mistake type's tag (word-order orange, plural blue, to-be pink, -s yellow…). */
const TONES: Record<MistakeType, string> = {
  'word-order': 'var(--color-skill-write)',
  plural: 'var(--color-skill-listen)',
  'to-be': 'var(--color-skill-speak)',
  'verb-s': 'var(--color-warn)',
  tense: 'var(--color-primary)',
  'verb-form': 'var(--color-primary)',
  article: 'var(--color-accent)',
  preposition: 'var(--color-accent)',
  'word-choice': 'var(--color-skill-read)',
  spelling: 'var(--color-skill-read)',
  other: 'var(--color-text-muted)',
};

/** The tag of a mistake type: its Vietnamese label on a tint of the type's colour. */
@Component({
  selector: 'lu-mistake-tag',
  template: `{{ label() }}`,
  host: { '[style.--tone]': 'tone()' },
  styles: `
    /* A tint behind ordinary text, so the label keeps its contrast in both themes. */
    :host {
      display: inline-block;
      padding: 1px var(--space-2);
      border-radius: var(--radius-sm);
      font: var(--text-xs) var(--font-ui);
      font-weight: 700;
      color: var(--color-text);
      background: color-mix(in srgb, var(--tone) 22%, var(--color-surface));
      white-space: nowrap;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MistakeTag {
  readonly type = input.required<MistakeType>();

  protected readonly label = computed(() => MISTAKE_LABELS[this.type()] ?? MISTAKE_LABELS.other);
  protected readonly tone = computed(() => TONES[this.type()] ?? TONES.other);
}
