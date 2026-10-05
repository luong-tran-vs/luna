import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { GrammarNote as GrammarNoteData } from '../../../../core/models/lesson';

/** "Ngữ pháp" section of the Reading step (F15): a native <details>, open by default. */
@Component({
  imports: [RouterLink],
  selector: 'lu-grammar-note',
  templateUrl: './grammar-note.html',
  styleUrl: './grammar-note.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarNote {
  readonly note = input.required<GrammarNoteData>();
  /** The grammar point this lesson teaches (F20); '' for none. Adds a link to its page. */
  readonly pointId = input('');

  /** Body paragraphs, split on blank lines. */
  protected readonly paragraphs = computed(() =>
    this.note()
      .bodyVi.split(/\n\s*\n/)
      .map((p) => p.trim())
      .filter(Boolean),
  );
}
