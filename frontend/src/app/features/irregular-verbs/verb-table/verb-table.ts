import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';

type Form = 'past' | 'participle';

/**
 * The verbs in three aligned columns V1 | V2 | V3, like a printed verb table (design/tat-ca-de-xuat.png (mục dong-tu-gon-lai-v2)):
 * the column titles stay on top while scrolling, the meaning goes under the forms on phones. A tick
 * marks the verbs learned; tapping a row asks for its details (`opened`).
 */
@Component({
  selector: 'lu-verb-table',
  imports: [Icon],
  templateUrl: './verb-table.html',
  styleUrl: './verb-table.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbTable {
  /** Verbs by letter; a group without a letter has no letter row (search results). */
  readonly groups = input.required<{ letter: string; verbs: IrregularVerb[] }[]>();
  /** Base forms of the verbs the learner confirmed as learned. */
  readonly learned = input<ReadonlySet<string>>(new Set());
  readonly opened = output<IrregularVerb>();

  protected readonly forms: readonly Form[] = ['past', 'participle'];
  protected readonly spellings = spellings;

  protected formOf(v: IrregularVerb, form: Form): string {
    return form === 'past' ? v.past : v.participle;
  }
}
