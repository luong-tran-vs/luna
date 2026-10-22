import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { MistakeStat } from '../../../core/models/writing';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { MistakeTag } from '../corrections/mistake-tag';

/**
 * "Lỗi hay gặp của bạn": the learner's 3 most frequent mistake types over their graded writings,
 * each with one example and how many times. Shows nothing when there are none or on error.
 */
@Component({
  selector: 'lu-frequent-mistakes',
  imports: [MistakeTag],
  template: `
    @if (stats().length) {
      <section class="card frequent" aria-labelledby="frequent-heading">
        <h3 class="title" id="frequent-heading">Lỗi hay gặp của bạn</h3>
        <ul class="list">
          @for (s of stats(); track s.type) {
            <li>
              <lu-mistake-tag [type]="s.type" />
              @if (s.examples[0]; as e) {
                <span class="example"><span class="visually-hidden">: </span><span lang="en">{{ e.original || '…' }} → {{ e.corrected }}</span></span>
              }
              <strong class="times"><span class="visually-hidden">, </span>{{ s.count }} lần</strong>
            </li>
          }
        </ul>
        <p class="note">Cộng dồn qua các bài viết đã chấm của bạn.</p>
      </section>
    }
  `,
  styles: `
    :host {
      display: contents;
    }

    .frequent {
      display: grid;
      gap: var(--space-2);
    }

    .title {
      font: var(--text-xs) var(--font-ui);
      font-weight: 700;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: var(--color-text-muted);
    }

    .list {
      display: grid;
      gap: var(--space-2);
      margin: 0;
      padding: 0;
      list-style: none;
      font: var(--text-sm) var(--font-ui);
    }

    .list li {
      display: flex;
      flex-wrap: wrap;
      align-items: baseline;
      gap: var(--space-1) var(--space-2);
    }

    .times {
      margin-left: auto;
      font-variant-numeric: tabular-nums;
    }

    .note {
      font: var(--text-xs) var(--font-ui);
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FrequentMistakes implements OnInit {
  private readonly api = inject(WritingApiService);

  protected readonly stats = signal<MistakeStat[]>([]);

  ngOnInit(): void {
    void this.load();
  }

  private async load(): Promise<void> {
    try {
      const stats = await firstValueFrom(this.api.mistakes());
      this.stats.set(stats.filter((s) => s.count > 0).slice(0, 3));
    } catch {
      this.stats.set([]);
    }
  }
}
