import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { Router } from '@angular/router';

import { ComposePlan, TestKind } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { kindLabel, PART_NAMES } from '../../toeic/shared/score.logic';
import { errorBody } from '../json-import/json-import';

const KINDS: TestKind[] = ['full', 'mini100', 'mini50'];

/**
 * "Ghép đề tự động": the app picks published groups so each part has the questions the kind needs
 * (unused groups first), shows the plan, and makes a draft test from it.
 */
@Component({
  selector: 'lu-toeic-compose-dialog',
  template: `
    <div class="backdrop">
      <div
        #dialog
        class="dialog wide"
        role="dialog"
        aria-modal="true"
        aria-labelledby="compose-title"
        tabindex="-1"
      >
        <h2 class="title" id="compose-title">Ghép đề tự động</h2>
        <p class="muted">
          Lấy các cụm đã đăng trong kho cho đủ số câu từng Part, ưu tiên cụm chưa thuộc đề nào. Đề
          tạo ra ở dạng nháp.
        </p>
        <div class="fields">
          <label>
            Loại đề
            <select (change)="setKind($event)">
              @for (k of kinds; track k) {
                <option [value]="k" [selected]="k === kind()">{{ label(k) }}</option>
              }
            </select>
          </label>
          <label class="grow">
            Tên đề (bỏ trống để tự đặt)
            <input
              type="text"
              maxlength="80"
              [value]="title()"
              (input)="setTitle($event)"
              [placeholder]="plan()?.title ?? ''"
            />
          </label>
        </div>

        @if (plan(); as p) {
          <table class="plan">
            <caption class="sr-only">
              Số câu lấy cho từng Part
            </caption>
            <thead>
              <tr>
                <th scope="col">Part</th>
                <th scope="col">Cần</th>
                <th scope="col">Lấy được</th>
                <th scope="col">Cụm (dùng lại)</th>
                <th scope="col">Kho có</th>
              </tr>
            </thead>
            <tbody>
              @for (row of p.parts; track row.part) {
                <tr [attr.data-ok]="row.filled">
                  <th scope="row">
                    Part {{ row.part }} <span class="muted">{{ names[row.part] }}</span>
                  </th>
                  <td>{{ row.need }}</td>
                  <td>{{ row.filled ? row.have + ' ✓' : '✕ chưa đủ' }}</td>
                  <td>
                    {{
                      row.filled ? row.groups + (row.reused ? ' (' + row.reused + ')' : '') : '—'
                    }}
                  </td>
                  <td>{{ row.available }} câu</td>
                </tr>
              }
            </tbody>
          </table>
          @if (!p.complete) {
            <label class="check">
              <input type="checkbox" [checked]="partial()" (change)="partial.set(!partial())" />
              Vẫn tạo đề, bỏ trống các Part chưa đủ (luyện tập được các Part đã có)
            </label>
          }
          <p class="muted">
            Mã đề: <code>{{ p.key }}</code>
          </p>
        } @else if (!error()) {
          <p class="muted" role="status">Đang xem kho…</p>
        }
        @if (error()) {
          <p class="form-alert" role="alert">{{ error() }}</p>
        }
        <div class="actions">
          <button type="button" class="btn" (click)="closed.emit()">Huỷ</button>
          <button
            type="button"
            class="btn-primary"
            [disabled]="!canCreate() || busy()"
            (click)="create()"
          >
            {{ busy() ? 'Đang tạo…' : 'Tạo đề nháp' }}
          </button>
        </div>
      </div>
    </div>
  `,
  styleUrls: ['../../../shared/components/confirm-dialog/confirm-dialog.css', './toeic-admin.css'],
  styles: `
    .fields {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-3);
    }
    .fields label {
      display: grid;
      gap: var(--space-1);
      font: var(--text-sm) var(--font-ui);
    }
    .fields .grow {
      flex: 1 1 220px;
    }
    input[type='text'] {
      min-height: 40px;
      padding: 0 var(--space-2);
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text);
      background: var(--color-surface);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }
    .plan tbody th {
      font-weight: 600;
    }
    .plan tr[data-ok='false'] td:nth-child(3) {
      color: var(--color-bad);
    }
    .check {
      display: flex;
      gap: var(--space-2);
      align-items: flex-start;
      font: var(--text-sm) var(--font-ui);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'closed.emit()' },
})
export class ComposeDialog {
  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);

  readonly closed = output<void>();

  protected readonly kinds = KINDS;
  protected readonly names = PART_NAMES;
  protected readonly kind = signal<TestKind>('full');
  protected readonly title = signal('');
  protected readonly partial = signal(false);
  protected readonly plan = signal<ComposePlan | null>(null);
  protected readonly error = signal('');
  protected readonly busy = signal(false);
  protected readonly canCreate = computed(() => {
    const p = this.plan();
    return !!p && p.groups.length > 0 && (p.complete || this.partial());
  });

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    this.preview();
    afterNextRender(() => this.dialog()?.nativeElement.querySelector('select')?.focus());
  }

  protected label(k: TestKind): string {
    return kindLabel(k);
  }

  protected setKind(event: Event): void {
    this.kind.set((event.target as HTMLSelectElement).value as TestKind);
    this.preview();
  }

  protected setTitle(event: Event): void {
    this.title.set((event.target as HTMLInputElement).value);
  }

  private preview(): void {
    this.plan.set(null);
    this.error.set('');
    this.partial.set(false);
    this.api.composeTest({ kind: this.kind(), dryRun: true }).subscribe({
      next: (r) => this.plan.set(r.plan),
      error: (err: unknown) => {
        const e = errorBody(err);
        const plan = e.body?.['plan'] as ComposePlan | undefined;
        if (plan) {
          this.plan.set(plan);
        } else {
          this.error.set(
            e.fields?.['groups']
              ? 'Kho chưa có cụm đã đăng nào.'
              : (e.message ?? 'Không xem được kho.'),
          );
        }
      },
    });
  }

  protected create(): void {
    this.busy.set(true);
    this.error.set('');
    this.api
      .composeTest({
        kind: this.kind(),
        title: this.title().trim() || undefined,
        partial: this.partial(),
      })
      .subscribe({
        next: (r) => {
          this.busy.set(false);
          if (r.test) {
            void this.router.navigate(['/admin/toeic/tests', r.test.key]);
          }
        },
        error: (err: unknown) => {
          this.busy.set(false);
          const e = errorBody(err);
          this.error.set(
            e.error === 'exists'
              ? 'Mã đề đã có, hãy đặt tên khác.'
              : (e.message ?? 'Chưa tạo được đề.'),
          );
        },
      });
  }
}
