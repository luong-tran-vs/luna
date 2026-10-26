import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

import {
  CheckItem,
  CurriculumSlot,
  ImportPart,
  ImportReport,
  LessonChecks,
  SlotBrief,
} from '../../../../core/models/curriculum';
import { AdminApiService } from '../../admin-api.service';
import {
  downloadJson,
  errorBody,
  fieldLines,
  parseJson,
  trapTab,
} from '../../json-import/json-import';
import { JsonSource } from '../../json-import/json-source';
import { checkResult } from '../lesson-row/lesson-row';

type Result = 'ok' | 'warn' | 'bad';

const MARK: Record<Result, string> = { ok: '✓', warn: '!', bad: '✕' };

/** The six steps of a lesson and the parts of an import each shows. */
const STEPS: { name: string; keys: string[] }[] = [
  { name: 'Từ vựng', keys: ['annotate.annotations', 'practice.examples'] },
  { name: 'Nghe', keys: ['practice.dialogue.turns', 'practice.listeningQuestions'] },
  {
    name: 'Đọc',
    keys: ['practice.readingGuideVi', 'annotate.questions', 'annotate.sentenceTranslations'],
  },
  { name: 'Ngữ pháp', keys: ['practice.grammarStep', 'practice.grammarStep.exercises'] },
  { name: 'Nói', keys: ['practice.readAloud', 'practice.speakQuestions'] },
  { name: 'Viết', keys: ['annotate.writingPrompt', 'practice.writingHelp'] },
];

export interface StepPreview {
  name: string;
  result: Result;
  detail: string;
}

/** Each step of the lesson: bad when a part of it is empty, warn when the checks dropped some. */
export function stepPreviews(report: ImportReport): StepPreview[] {
  const byKey = new Map(report.parts.map((p) => [p.key, p]));
  return STEPS.map((s) => {
    const parts = s.keys.map((k) => byKey.get(k)).filter((p): p is ImportPart => !!p);
    let result: Result = 'ok';
    if (parts.some((p) => p.kept === 0)) {
      result = 'bad';
    } else if (parts.some((p) => p.kept < p.given)) {
      result = 'warn';
    }
    return {
      name: s.name,
      result,
      detail: parts.map((p) => `${p.labelVi}: ${p.kept}`).join(' · '),
    };
  });
}

/** A JSON file to fill in for the slot: every part with one empty item, the slot's words set. */
export function slotTemplate(b: SlotBrief): unknown {
  const grammar = b.grammar[0];
  const question = { promptVi: '', quote: '', answerIndex: 0, explanationVi: '' };
  return {
    content: '',
    targetWords: b.targetWords,
    annotate: {
      annotations: [{ text: '', lemma: '', meaningVi: '', pos: '', sentenceIndex: 0 }],
      questions: [
        { kind: 'gist', prompt: '', options: ['', '', '', ''], ...question },
        { kind: 'tfn', prompt: '', options: [], ...question },
      ],
      grammarNote: { title: grammar?.titleVi ?? '', bodyVi: '', examples: [] },
      writingPrompt: '',
      sentenceTranslations: [],
    },
    practice: {
      objectiveVi: '',
      examples: [{ lemma: '', sentence: '', meaningVi: '' }],
      dialogue: { speakers: ['', ''], turns: [{ speaker: 0, text: '', meaningVi: '' }] },
      grammarTipVi: '',
      translations: [{ vi: '', en: '', distractors: [] }],
      listeningQuestions: [
        {
          kind: 'gist',
          prompt: '',
          promptVi: '',
          options: ['', '', ''],
          answerIndex: 0,
          explanationVi: '',
          turnIndex: -1,
        },
      ],
      readingGuideVi: '',
      grammarStep: {
        titleVi: grammar?.titleVi ?? '',
        noticing: [{ sentence: '', highlights: [] }],
        questionVi: '',
        options: ['', '', ''],
        answerIndex: 0,
        explanationVi: '',
        whenVi: '',
        table: [{ subject: '', form: '', negative: '' }],
        mistakes: [{ wrong: '', right: '' }],
        noteVi: '',
        exercises: [
          { kind: 'fill', promptVi: '', text: '___', answers: [''], explanationVi: '' },
          {
            kind: 'choice',
            promptVi: '',
            text: '___',
            options: ['', '', '', ''],
            answerIndex: 0,
            explanationVi: '',
          },
          { kind: 'fix', promptVi: '', text: '', sentence: '', explanationVi: '' },
          { kind: 'reorder', promptVi: '', text: '', sentence: '', words: [], explanationVi: '' },
        ],
      },
      readAloud: [{ text: '', tipVi: '' }],
      speakQuestions: [{ question: '', questionVi: '', frame: '' }],
      writingHelp: { grammarLabelVi: '', anyOf: [], outlineVi: [], modelAnswer: '' },
    },
    glosses: [],
  };
}

/**
 * FR-030 (added 2026-10-26): imports a lesson written outside the app, by the admin or an agent, into
 * an empty slot. It shows what the lesson is written from (the slot's brief), takes the JSON pasted
 * or from a file, checks it on the server without saving (preview of the six steps, rule checks,
 * what was dropped and why), then saves it as a draft.
 */
@Component({
  selector: 'lu-slot-import-dialog',
  imports: [JsonSource],
  templateUrl: './import-dialog.html',
  styleUrls: [
    '../../../../shared/components/confirm-dialog/confirm-dialog.css',
    './import-dialog.css',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'onEscape()' },
})
export class SlotImportDialog {
  private readonly api = inject(AdminApiService);

  readonly open = input(false);
  readonly slotKey = input('');
  readonly slotTitle = input('');

  /** The slot with its new draft. */
  readonly saved = output<CurriculumSlot>();
  readonly closed = output<void>();

  protected readonly brief = signal<SlotBrief | null>(null);
  protected readonly briefError = signal(false);
  protected readonly text = signal('');
  protected readonly checking = signal(false);
  protected readonly saving = signal(false);
  protected readonly parseError = signal('');
  protected readonly failure = signal('');
  protected readonly fields = signal<Record<string, string> | null>(null);
  protected readonly report = signal<ImportReport | null>(null);
  protected readonly checks = signal<LessonChecks | null>(null);
  /** The text the last check ran on: saving needs it unchanged. */
  private readonly checkedText = signal<string | null>(null);

  protected readonly mark = MARK;
  protected readonly fieldList = computed(() => fieldLines(this.fields() ?? {}));
  protected readonly steps = computed(() => {
    const r = this.report();
    return r ? stepPreviews(r) : [];
  });
  protected readonly lost = computed(() =>
    (this.report()?.parts ?? []).filter((p) => p.kept < p.given),
  );
  protected readonly stale = computed(
    () => this.checkedText() !== null && this.checkedText() !== this.text(),
  );
  protected readonly canSave = computed(
    () =>
      this.checkedText() === this.text() &&
      this.fields() === null &&
      this.report() !== null &&
      !this.saving() &&
      !this.checking(),
  );
  protected readonly status = computed(() => {
    if (this.checking()) {
      return 'Đang kiểm tra…';
    }
    if (this.saving()) {
      return 'Đang lưu…';
    }
    if (this.stale()) {
      return 'Nội dung đã đổi sau lần kiểm tra, hãy kiểm tra lại.';
    }
    if (this.fields()) {
      return 'Bài chưa hợp lệ, chưa lưu được.';
    }
    if (this.report()) {
      return 'Đúng định dạng. Đã kiểm tra, chưa lưu.';
    }
    return '';
  });

  private readonly dialog = viewChild<ElementRef<HTMLElement>>('dialog');

  constructor() {
    effect(() => {
      if (this.open()) {
        const key = this.slotKey();
        untracked(() => this.reset(key));
      }
    });
    afterRenderEffect(() => {
      if (this.open()) {
        untracked(() => this.dialog()?.nativeElement.querySelector<HTMLElement>('h2')?.focus());
      }
    });
  }

  private reset(key: string): void {
    this.brief.set(null);
    this.briefError.set(false);
    this.text.set('');
    this.parseError.set('');
    this.failure.set('');
    this.fields.set(null);
    this.report.set(null);
    this.checks.set(null);
    this.checkedText.set(null);
    this.api.curriculumBrief(key).subscribe({
      next: (b) => this.brief.set(b),
      error: () => this.briefError.set(true),
    });
  }

  protected onEscape(): void {
    if (this.open()) {
      this.closed.emit();
    }
  }

  protected onKeydown(event: KeyboardEvent): void {
    trapTab(event, this.dialog()?.nativeElement);
  }

  protected result(item: CheckItem): Result {
    return checkResult(item);
  }

  protected downloadTemplate(): void {
    const b = this.brief();
    if (b) {
      downloadJson(`${b.key}.json`, slotTemplate(b));
    }
  }

  /** Checks the lesson on the server without saving it. */
  protected async check(): Promise<void> {
    const parsed = this.parse();
    if (!parsed) {
      return;
    }
    const text = this.text();
    this.checking.set(true);
    try {
      const res = await firstValueFrom(
        this.api.importCurriculumSlot(this.slotKey(), parsed.value, true),
      );
      this.fields.set(null);
      this.report.set(res.report);
      this.checks.set(res.checks);
      this.checkedText.set(text);
    } catch (err) {
      this.showError(err, text);
    } finally {
      this.checking.set(false);
    }
  }

  protected async save(): Promise<void> {
    if (!this.canSave()) {
      return;
    }
    const parsed = this.parse();
    if (!parsed) {
      return;
    }
    this.saving.set(true);
    try {
      const res = await firstValueFrom(
        this.api.importCurriculumSlot(this.slotKey(), parsed.value, false),
      );
      this.saved.emit(res.slot);
    } catch (err) {
      this.showError(err, this.text());
    } finally {
      this.saving.set(false);
    }
  }

  private parse(): { value: unknown } | null {
    this.failure.set('');
    const parsed = parseJson(this.text());
    if ('error' in parsed) {
      this.parseError.set(parsed.error);
      return null;
    }
    this.parseError.set('');
    return parsed;
  }

  /** An invalid lesson shows its field errors and report; any other failure its message. */
  private showError(err: unknown, text: string): void {
    const e = errorBody(err);
    if (e.fields) {
      this.fields.set(e.fields);
      this.report.set((e.body?.['report'] as ImportReport | undefined) ?? null);
      this.checks.set(null);
      this.checkedText.set(text);
      return;
    }
    this.failure.set(e.message ?? 'Không gửi được bài, vui lòng thử lại.');
  }
}
