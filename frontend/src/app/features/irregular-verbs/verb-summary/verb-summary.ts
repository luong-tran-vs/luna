import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { changedPart, spokenForm, threeForms } from '../verb-study.logic';

/** Which column the learner hides to test themselves; 0 shows all. */
type Hide = 0 | 2 | 3;

/**
 * The end of "Làm quen" (design/de-xuat-dong-tu.png part 3, added 2026-11-10): every verb of the
 * lesson on one screen. The learner may hide V2 or V3 and tap a cell to check it after saying it,
 * and hear the whole table. Nothing is graded.
 */
@Component({
  selector: 'lu-verb-summary',
  imports: [Icon],
  template: `
    <p class="eyebrow">Tóm tắt · {{ verbs().length }} từ</p>
    <div class="hides" role="group" aria-label="Tự kiểm tra">
      @for (o of options; track o.value) {
        <button
          type="button"
          class="hide"
          [attr.aria-pressed]="hide() === o.value"
          (click)="setHide(o.value)"
        >
          {{ o.label }}
        </button>
      }
    </div>
    <table class="table" lang="en">
      <caption class="visually-hidden">
        Ba dạng của các động từ trong bài
      </caption>
      <thead>
        <tr>
          <th scope="col">V1</th>
          <th scope="col">V2</th>
          <th scope="col">V3</th>
        </tr>
      </thead>
      <tbody>
        @for (r of rows(); track r.verb.base) {
          <tr>
            @for (c of r.cells; track $index; let i = $index) {
              <td>
                @if (i === 0 && pictures().get(r.verb.base); as src) {
                  <img class="thumb" [src]="src" alt="" width="32" height="24" loading="lazy" />
                }
                @if (hidden(r.verb.base, i)) {
                  <button
                    type="button"
                    class="cover"
                    lang="vi"
                    [attr.aria-label]="'Xem V' + (i + 1) + ' của ' + r.verb.base"
                    (click)="reveal(r.verb.base)"
                  >
                    chạm
                  </button>
                } @else {
                  <span [class.checked]="hide() === i + 1"
                    ><span>{{ c[0] }}</span>
                    @if (c[1]) {
                      <mark [class]="'v' + (i + 1)">{{ c[1] }}</mark>
                    }
                  </span>
                }
              </td>
            }
          </tr>
        }
      </tbody>
    </table>
    @if (hide()) {
      <p class="hint">Nhẩm trước rồi chạm ô để kiểm tra.</p>
    }
    <div class="bottom-bar">
      @if (canSpeak) {
        <button type="button" class="btn" (click)="sayAll()">
          <lu-icon name="volume" /> Nghe cả bảng
        </button>
      }
      <button type="button" class="btn-primary" (click)="next.emit()">
        Sang bước Chọn <lu-icon name="arrow-right" />
      </button>
    </div>
  `,
  styleUrl: './verb-summary.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbSummary {
  private readonly speech = inject(SpeechService);

  readonly verbs = input.required<readonly IrregularVerb[]>();
  readonly pictures = input<ReadonlyMap<string, string>>(new Map());
  readonly next = output();

  protected readonly canSpeak = this.speech.supported;
  protected readonly options: readonly { value: Hide; label: string }[] = [
    { value: 0, label: 'Hiện hết' },
    { value: 2, label: 'Che V2' },
    { value: 3, label: 'Che V3' },
  ];
  protected readonly hide = signal<Hide>(0);
  private readonly shown = signal<ReadonlySet<string>>(new Set());

  protected readonly rows = computed(() =>
    this.verbs().map((v) => {
      const [base, past, participle] = threeForms(v);
      return {
        verb: v,
        cells: [changedPart(past, base), changedPart(base, past), changedPart(base, participle)],
      };
    }),
  );

  protected setHide(h: Hide): void {
    this.hide.set(h);
    this.shown.set(new Set());
  }

  protected hidden(base: string, i: number): boolean {
    return this.hide() === i + 1 && !this.shown().has(base);
  }

  protected reveal(base: string): void {
    this.shown.update((s) => new Set(s).add(base));
  }

  /** Reads every verb's three forms, one verb after another. */
  protected sayAll(): void {
    const text = this.verbs()
      .map((v) =>
        threeForms(v)
          .map((f, i) => spokenForm(v.base, f, i))
          .join(', '),
      )
      .join('. ');
    this.speech.speak(text, 0.85);
  }
}
