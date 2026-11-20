import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { PART_NAMES } from '../shared/score.logic';
import { ListeningPlayer, RATES } from './listening-player';
import { DIRECTIONS, scriptSeconds } from './listening-script.logic';

/**
 * The audio bar of the listening parts (FR-018, FR-019): what is playing and how far, the directions
 * of a part while they are read, and the controls. The audio starts on a tap (browsers block speech
 * before one). While practising: pause / go on, hear the group again, the speed, skip the
 * directions; in a test it only starts and goes on.
 */
@Component({
  selector: 'lu-toeic-listening-bar',
  template: `
    @let pl = player();
    <section class="bar" aria-label="Âm thanh phần Nghe">
      <div class="row">
        @if (pl.playing()) {
          @if (practice()) {
            <button type="button" class="btn-primary ctl" (click)="pl.pause()">⏸ Tạm dừng</button>
          } @else {
            <span class="live" aria-hidden="true">●</span>
          }
        } @else if (pl.finished()) {
          <span class="done">✓ Hết phần Nghe</span>
          @if (practice()) {
            <button type="button" class="btn ctl" (click)="pl.play(0)">⟲ Nghe lại từ đầu</button>
          }
        } @else {
          <button type="button" class="btn-primary ctl" (click)="pl.index() < 0 ? pl.play(0) : pl.resume()">
            {{ pl.index() < 0 ? '▶ Bắt đầu phần Nghe' : '▶ Phát tiếp' }}
          </button>
        }
        <p class="status" aria-live="polite">{{ status() }}</p>
        @if (practice() && pl.index() >= 0 && !pl.finished()) {
          <button type="button" class="btn ctl" (click)="pl.replayGroup()">⟲ Nghe lại cụm</button>
          @if (pl.segment()?.kind === 'directions') {
            <button type="button" class="btn ctl" (click)="pl.skipDirections()">Bỏ qua hướng dẫn</button>
          }
        }
        @if (practice()) {
          <label class="rate">
            Tốc độ
            <select (change)="setRate($event)">
              @for (r of rates; track r) {
                <option [value]="r" [selected]="r === pl.rate()">{{ r }}×</option>
              }
            </select>
          </label>
        }
      </div>
      <div
        class="track"
        role="progressbar"
        aria-label="Tiến độ phần Nghe"
        aria-valuemin="0"
        [attr.aria-valuemax]="pl.script.length"
        [attr.aria-valuenow]="Math.max(0, pl.index() + 1)"
      >
        <span class="fill" [style.width.%]="percent()"></span>
      </div>
      @if (directions(); as d) {
        <div class="directions">
          <p lang="en">{{ d.en }}</p>
          <p class="vi">{{ d.vi }}</p>
        </div>
      }
      @if (pl.error() === 'unsupported') {
        <p class="form-alert" role="alert">Trình duyệt này không đọc được tiếng Anh. Hãy dùng Chrome, Edge hoặc Safari.</p>
      } @else if (pl.error()) {
        <p class="form-alert" role="alert">Trình duyệt chặn âm thanh: bấm “Phát tiếp” để nghe.</p>
      }
    </section>
  `,
  styles: `
    .bar {
      display: grid;
      gap: var(--space-2);
      padding: var(--space-3);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-lg);
    }
    .row {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-2);
      align-items: center;
    }
    .ctl {
      min-height: 40px;
    }
    .status {
      flex: 1 1 180px;
      margin: 0;
      font: var(--text-sm) var(--font-ui);
      font-weight: 600;
    }
    .live {
      color: var(--color-bad);
    }
    .done {
      font: var(--text-sm) var(--font-ui);
      font-weight: 700;
      color: var(--color-ok);
    }
    .rate {
      display: flex;
      gap: var(--space-1);
      align-items: center;
      font: var(--text-sm) var(--font-ui);
    }
    select {
      min-height: 40px;
      padding: 0 var(--space-2);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }
    .track {
      height: 6px;
      overflow: hidden;
      background: var(--color-track);
      border-radius: 999px;
    }
    .fill {
      display: block;
      height: 100%;
      background: var(--color-primary);
      transition: width 0.3s;
    }
    .directions {
      display: grid;
      gap: var(--space-1);
      padding: var(--space-2) var(--space-3);
      font: var(--text-sm) var(--font-ui);
      background: var(--color-bg);
      border-radius: var(--radius-md);
    }
    .directions p {
      margin: 0;
    }
    .vi {
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ListeningBar {
  readonly player = input.required<ListeningPlayer>();
  readonly practice = input(true);
  /** The numbers of the questions of each group key, to say "Câu 41–43". */
  readonly numbers = input<Record<string, string>>({});

  protected readonly rates = RATES;
  protected readonly Math = Math;

  protected readonly percent = computed(() => {
    const pl = this.player();
    if (pl.finished()) {
      return 100;
    }
    return pl.script.length ? (Math.max(0, pl.index()) / pl.script.length) * 100 : 0;
  });

  protected readonly directions = computed(() => {
    const s = this.player().segment();
    return s?.kind === 'directions' ? DIRECTIONS[s.part as 1 | 2 | 3 | 4] : null;
  });

  protected readonly status = computed(() => {
    const pl = this.player();
    const s = pl.segment();
    if (pl.finished()) {
      return '';
    }
    if (!s) {
      return `Phần Nghe · khoảng ${Math.round(pl.script.length ? this.minutes() : 0)} phút`;
    }
    const where = `Part ${s.part} · ${PART_NAMES[s.part]}`;
    if (s.kind === 'directions') {
      return `${where} · hướng dẫn`;
    }
    const nums = s.group ? this.numbers()[s.group] : '';
    return `${where}${nums ? ' · câu ' + nums : ''}${pl.playing() ? '' : ' · đã dừng'}`;
  });

  private minutes(): number {
    return scriptSeconds(this.player().script, this.player().rate()) / 60;
  }

  protected setRate(event: Event): void {
    this.player().setRate(Number((event.target as HTMLSelectElement).value));
  }
}
