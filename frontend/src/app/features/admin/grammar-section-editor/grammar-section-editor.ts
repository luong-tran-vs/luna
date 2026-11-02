import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormArray, FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';

import { GrammarContent, GrammarExample } from '../../../core/models/grammar-admin';
import { PatternRow, patternParts, patternRows } from '../../grammar/grammar-logic';
import { Pattern } from '../../grammar/pattern/pattern';

/** The parts of a grammar lesson the admin can edit one at a time (the exercises have their own form). */
export type GrammarSection = 'theory' | 'structures' | 'examples' | 'mistakes';

export const SECTION_LABELS: Record<GrammarSection, string> = {
  theory: 'Mục tiêu, giải thích và cách dùng',
  structures: 'Cấu trúc',
  examples: 'Ví dụ',
  mistakes: 'Lỗi thường gặp',
};

const text = (value = '') => new FormControl(value, { nonNullable: true });

function structureRow(label = '', pattern = '', example = '', origin: number | null = null) {
  return new FormGroup({
    label: text(label),
    pattern: text(pattern),
    example: text(example),
    /** The structure's index in the saved lesson, null for one added here. */
    origin: new FormControl<number | null>(origin),
  });
}

function exampleRow(e: Partial<GrammarExample> = {}) {
  return new FormGroup({
    en: text(e.en),
    vi: text(e.vi),
    structure: new FormControl(e.structure ?? 0, { nonNullable: true }),
    marked: text(e.marked),
  });
}

function mistakeRow(wrong = '', right = '', noteVi = '') {
  return new FormGroup({ wrong: text(wrong), right: text(right), noteVi: text(noteVi) });
}

/** Lines of a textarea, trimmed, blank ones dropped. */
const lines = (value: string) =>
  value
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l !== '');

/** Paragraphs of a textarea: separated by a blank line. */
const paragraphs = (value: string) =>
  value
    .split(/\n\s*\n/)
    .map((p) => p.replace(/\s*\n\s*/g, ' ').trim())
    .filter((p) => p !== '');

/** A structure whose pattern now has another number of pieces than the marked examples count on. */
export interface MarkWarning {
  row: number;
  before: number;
  after: number;
  sentences: string[];
}

/**
 * Edits one section of a grammar lesson and hands back the whole content with that section replaced,
 * so a fix in the theory does not mean editing the lesson's JSON. The exercises are never touched,
 * so saving keeps the AI check. The server validates as for any save.
 */
@Component({
  selector: 'lu-grammar-section-editor',
  imports: [ReactiveFormsModule, Pattern],
  templateUrl: './grammar-section-editor.html',
  styleUrl: './grammar-section-editor.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarSectionEditor implements OnInit {
  private readonly destroyRef = inject(DestroyRef);

  readonly section = input.required<GrammarSection>();
  readonly content = input.required<GrammarContent>();
  readonly busy = input(false);
  /** The server's answer to the last save: a message and "đường dẫn: thông báo" lines. */
  readonly error = input<string | null>(null);
  readonly fields = input<string[]>([]);

  readonly saved = output<GrammarContent>();
  readonly cancelled = output<void>();

  protected readonly label = computed(() => SECTION_LABELS[this.section()]);

  protected readonly objective = text();
  protected readonly explanation = text();
  protected readonly usage = text();
  protected readonly structures = new FormArray<ReturnType<typeof structureRow>>([]);
  protected readonly examples = new FormArray<ReturnType<typeof exampleRow>>([]);
  protected readonly mistakes = new FormArray<ReturnType<typeof mistakeRow>>([]);
  /** Drop the marks of the examples a changed pattern no longer fits (they are then coloured by guess). */
  protected readonly unmark = new FormControl(false, { nonNullable: true });
  protected readonly form = new FormGroup({
    objective: this.objective,
    explanation: this.explanation,
    usage: this.usage,
    structures: this.structures,
    examples: this.examples,
    mistakes: this.mistakes,
    unmark: this.unmark,
  });

  /** The structures as typed, kept in a signal so the previews and warnings follow the typing. */
  private readonly typed = signal<{ pattern: string; origin: number | null }[]>([]);

  protected readonly previews = computed<PatternRow[][]>(() =>
    patternRows(this.typed().map((s) => s.pattern)),
  );

  protected readonly warnings = computed<MarkWarning[]>(() => {
    const saved = this.content();
    const out: MarkWarning[] = [];
    this.typed().forEach((s, row) => {
      if (s.origin === null) {
        return;
      }
      const marked = saved.examples.filter((e) => e.structure === s.origin! + 1 && e.marked);
      const before = patternParts([saved.structures[s.origin].pattern])[0].length;
      const after = patternParts([s.pattern])[0].length;
      if (marked.length > 0 && before !== after) {
        out.push({ row, before, after, sentences: marked.map((e) => e.en) });
      }
    });
    return out;
  });

  ngOnInit(): void {
    const c = this.content();
    this.objective.setValue(c.objective);
    this.explanation.setValue(c.explanation.join('\n\n'));
    this.usage.setValue(c.usage.join('\n'));
    c.structures.forEach((s, i) =>
      this.structures.push(structureRow(s.label, s.pattern, s.example, i)),
    );
    c.examples.forEach((e) => this.examples.push(exampleRow(e)));
    c.mistakes.forEach((m) => this.mistakes.push(mistakeRow(m.wrong, m.right, m.noteVi)));
    this.syncTyped();
    this.structures.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.syncTyped());
  }

  private syncTyped(): void {
    this.typed.set(
      this.structures.getRawValue().map((s) => ({ pattern: s.pattern, origin: s.origin })),
    );
  }

  /** How many examples show the structure on this row (only saved structures have examples). */
  protected examplesOf(row: number): number {
    const origin = this.structures.at(row).controls.origin.value;
    return origin === null
      ? 0
      : this.content().examples.filter((e) => e.structure === origin + 1).length;
  }

  protected addStructure(): void {
    this.structures.push(structureRow());
  }

  protected removeStructure(row: number): void {
    this.structures.removeAt(row);
  }

  protected addExample(): void {
    this.examples.push(exampleRow());
  }

  protected removeExample(row: number): void {
    this.examples.removeAt(row);
  }

  protected addMistake(): void {
    this.mistakes.push(mistakeRow());
  }

  protected removeMistake(row: number): void {
    this.mistakes.removeAt(row);
  }

  /** The numbers an example can name: 0 (none) and every structure of the lesson. */
  protected structureNumbers(): number[] {
    return this.content().structures.map((_, i) => i + 1);
  }

  protected submit(): void {
    if (!this.busy()) {
      this.saved.emit(this.result());
    }
  }

  /** The lesson's content with this section replaced; the exercises are passed on untouched. */
  result(): GrammarContent {
    const c = this.content();
    switch (this.section()) {
      case 'theory':
        return {
          ...c,
          objective: this.objective.value.trim(),
          explanation: paragraphs(this.explanation.value),
          usage: lines(this.usage.value),
        };
      case 'structures':
        return this.withStructures(c);
      case 'examples':
        return {
          ...c,
          examples: this.examples.getRawValue().map((e) => {
            const out: GrammarExample = { en: e.en.trim(), vi: e.vi.trim() };
            if (e.structure > 0) {
              out.structure = Number(e.structure);
              if (e.marked.trim()) {
                out.marked = e.marked.trim();
              }
            }
            return out;
          }),
        };
      case 'mistakes':
        return {
          ...c,
          mistakes: this.mistakes.getRawValue().map((m) => ({
            wrong: m.wrong.trim(),
            right: m.right.trim(),
            noteVi: m.noteVi.trim(),
          })),
        };
    }
  }

  /**
   * The structures as edited. The examples follow their structure to its new number; those of a removed
   * structure lose their number and marks, and so do the marks a changed pattern no longer fits when
   * the admin asked for it.
   */
  private withStructures(c: GrammarContent): GrammarContent {
    const rows = this.structures.getRawValue();
    const newNumber = new Map<number, number>();
    rows.forEach((r, i) => {
      if (r.origin !== null) {
        newNumber.set(r.origin + 1, i + 1);
      }
    });
    const unmarkRows = new Set(this.unmark.value ? this.warnings().map((w) => w.row + 1) : []);
    const examples = c.examples.map((e) => {
      if (!e.structure) {
        return e;
      }
      const moved = newNumber.get(e.structure);
      if (moved === undefined) {
        return { en: e.en, vi: e.vi };
      }
      const out: GrammarExample = { ...e, structure: moved };
      if (unmarkRows.has(moved)) {
        delete out.marked;
      }
      return out;
    });
    return {
      ...c,
      structures: rows.map((r) => ({
        label: r.label.trim(),
        pattern: r.pattern.trim(),
        example: r.example.trim(),
      })),
      examples,
    };
  }
}
