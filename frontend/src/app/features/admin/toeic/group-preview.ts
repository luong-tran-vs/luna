import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';

import { GroupFull, ToeicDoc } from '../../../core/models/toeic';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { ImageSlots } from './image-slots';
import { DocView } from '../../toeic/shared/doc-view';
import { QuestionView } from '../../toeic/shared/question-view';
import { PART_NAMES } from '../../toeic/shared/score.logic';

/**
 * The preview of a group as the learner sees it, with the answers, evidence, explanations and
 * translations (FR-042).
 */
@Component({
  selector: 'lu-toeic-group-preview',
  imports: [DocView, QuestionView, ImageSlots],
  template: `
    <div class="backdrop">
      <div
        class="dialog wide"
        role="dialog"
        aria-modal="true"
        aria-labelledby="toeic-preview-title"
      >
        <header class="head">
          <h2 class="title" id="toeic-preview-title">
            Xem trước <code>{{ key() }}</code>
          </h2>
          <button type="button" class="close" aria-label="Đóng" (click)="closed.emit()">×</button>
        </header>
        @if (error()) {
          <p class="form-alert" role="alert">{{ error() }}</p>
        } @else if (group(); as g) {
          <p class="muted">
            Part {{ g.part }} · {{ names[g.part] }} · mức {{ g.level }} · {{ g.status }} · bản
            {{ g.version }}
            @if (g.source) {
              · {{ g.source }}
            }
          </p>
          @if (g.part === 1 && g.scene) {
            <p class="scene">
              <b>Cảnh (vẽ ảnh):</b> <span lang="en">{{ g.scene }}</span>
            </p>
          }
          @if (g.part === 1 || g.docs?.length) {
            <section class="pictures" aria-label="Ảnh của cụm">
              <h3 class="sub">Ảnh</h3>
              <lu-toeic-image-slots [groupKey]="g.key" />
            </section>
          }
          @if (g.script?.length) {
            <ol class="script">
              @for (t of g.script; track $index) {
                <li>
                  <b>{{ t.speaker }} ({{ t.accent }}):</b> <span lang="en">{{ t.text }}</span>
                  @if (t.translateVi) {
                    <br /><span class="muted">{{ t.translateVi }}</span>
                  }
                </li>
              }
            </ol>
          }
          @for (d of docs(); track $index) {
            <lu-toeic-doc
              [doc]="d"
              [groupKey]="g.key"
              [docIndex]="$index"
              [blankNumbers]="blankNumbers()"
              [evidence]="evidence()"
              [showTranslation]="true"
            />
          }
          @for (q of questions(); track q.id) {
            <div class="q">
              <lu-toeic-question [question]="q" [answer]="q.answer" [readonly]="true" />
              <p class="small"><b>#</b> {{ q.tags.join(', ') }}</p>
              @if (q.translateVi) {
                <p class="small"><b>Dịch:</b> {{ q.translateVi }}</p>
              }
              <p class="small"><b>Giải thích:</b> {{ q.explainVi }}</p>
            </div>
          }
        } @else {
          <p class="muted" role="status">Đang tải…</p>
        }
        <div class="actions">
          <button type="button" class="btn" (click)="closed.emit()">Đóng</button>
        </div>
      </div>
    </div>
  `,
  styleUrls: [
    '../../../shared/components/confirm-dialog/confirm-dialog.css',
    './import-dialog.css',
  ],
  styles: `
    .scene,
    .small {
      font: var(--text-sm) var(--font-ui);
    }
    .pictures {
      display: grid;
      gap: var(--space-2);
      padding: var(--space-3);
      border: 1px solid var(--color-line);
      border-radius: var(--radius-md);
    }
    .sub {
      margin: 0;
      font: var(--text-base) var(--font-ui);
      font-weight: 700;
    }
    .script {
      display: grid;
      gap: var(--space-1);
      margin: 0;
      padding-left: var(--space-4);
      font: var(--text-sm) var(--font-ui);
    }
    .q {
      display: grid;
      gap: var(--space-1);
      padding-top: var(--space-2);
      border-top: 1px solid var(--color-line);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'closed.emit()' },
})
export class GroupPreview implements OnInit {
  readonly key = input.required<string>();
  readonly closed = output<void>();

  private readonly api = inject(ToeicApiService);
  protected readonly group = signal<GroupFull | null>(null);
  protected readonly error = signal('');
  protected readonly names = PART_NAMES;

  protected readonly docs = computed<ToeicDoc[]>(() =>
    (this.group()?.docs ?? []).map((d) => ({ ...d, meta: d.meta ?? [], imageUrl: null })),
  );
  protected readonly questions = computed(() => {
    const g = this.group();
    return (g?.questions ?? []).map((q) => ({ ...q, id: `${g?.key}#${q.no}`, number: q.no }));
  });
  protected readonly blankNumbers = computed<Record<number, number>>(() => {
    const out: Record<number, number> = {};
    for (const q of this.group()?.questions ?? []) {
      out[q.no] = q.no;
    }
    return out;
  });
  protected readonly evidence = computed(() =>
    (this.group()?.questions ?? []).flatMap((q) => q.evidence ?? []),
  );

  ngOnInit(): void {
    this.api.group(this.key()).subscribe({
      next: (g) => this.group.set(g),
      error: () => this.error.set('Không tải được cụm này.'),
    });
  }
}
