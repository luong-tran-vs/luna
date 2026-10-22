import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { GrammarExercise } from '../../../../core/models/grammar-study';
import { GrammarNote } from '../../../../core/models/lesson';
import { GrammarStepContent } from '../../../../core/models/practice';
import { SpeechService } from '../../../../core/services/speech.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { SpeakButton } from '../../../../shared/directives/speak-button';
import { GrammarExerciseView } from '../../../grammar/grammar-exercise/grammar-exercise';

/** A piece of a noticing sentence; `mark` pieces are the words to notice. */
export interface MarkedPart {
  text: string;
  mark: boolean;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Splits a sentence into plain and marked pieces: every occurrence of a highlight, ignoring case,
 * as a whole word when it starts or ends with a letter (so "has" does not mark "hash").
 * Longer highlights win over shorter ones that overlap them.
 */
export function markHighlights(sentence: string, highlights: readonly string[]): MarkedPart[] {
  const words = [...new Set(highlights.map((h) => h.trim()).filter((h) => h !== ''))].sort(
    (a, b) => b.length - a.length,
  );
  if (words.length === 0 || sentence === '') {
    return sentence ? [{ text: sentence, mark: false }] : [];
  }
  const alternatives = words.map((w) => {
    const start = /^\w/.test(w) ? '\\b' : '';
    const end = /\w$/.test(w) ? '\\b' : '';
    return start + escapeRegExp(w) + end;
  });
  const re = new RegExp(alternatives.join('|'), 'gi');
  const parts: MarkedPart[] = [];
  let last = 0;
  for (const m of sentence.matchAll(re)) {
    const at = m.index ?? 0;
    if (at > last) {
      parts.push({ text: sentence.slice(last, at), mark: false });
    }
    parts.push({ text: m[0], mark: true });
    last = at + m[0].length;
  }
  if (last < sentence.length) {
    parts.push({ text: sentence.slice(last), mark: false });
  }
  return parts;
}

type Phase = 'notice' | 'rule' | 'practice';

/**
 * The Grammar step of the six-step lesson (2026-10-22, design section 4): notice the pattern in the
 * lesson's sentences and answer why (Nhận ra), the rule in one card (Quy tắc), then the exercises one
 * at a time (Luyện tập) and a link to the full grammar point. A lesson made before the six steps has
 * no content: its old grammar note shows instead.
 */
@Component({
  selector: 'lu-grammar-step',
  imports: [GrammarExerciseView, Icon, RouterLink, SpeakButton],
  templateUrl: './grammar-step.html',
  styleUrls: ['../practice.css', './grammar-step.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GrammarStep {
  private readonly speech = inject(SpeechService);
  protected readonly canSpeak = this.speech.supported;

  /** null for a lesson whose practice was made before the six steps. */
  readonly content = input<GrammarStepContent | null>(null);
  /** The lesson's old grammar note, shown when there is no content. */
  readonly note = input<GrammarNote | null>(null);
  /** The syllabus point of the lesson, '' when none. */
  readonly grammarPointId = input('');
  readonly grammarTitleVi = input('');
  readonly number = input(0);
  readonly finishLabel = input('Tiếp theo');
  readonly canBack = input(false);
  readonly back = output<void>();
  /** The step is finished; the page records it and moves on. */
  readonly done = output<void>();

  /** New content starts the step afresh. */
  protected readonly phase = linkedSignal<GrammarStepContent | null, Phase>({
    source: this.content,
    computation: () => 'notice',
  });
  /** The why-question's chosen option, null until tapped. */
  protected readonly chosen = linkedSignal<GrammarStepContent | null, number | null>({
    source: this.content,
    computation: () => null,
  });
  protected readonly index = signal(0);
  /** Whether the exercise on screen has been checked. */
  protected readonly answered = signal(false);
  protected readonly correct = signal(0);

  private readonly exerciseView = viewChild(GrammarExerciseView);

  protected readonly title = computed(() => (this.number() ? this.number() + '. ' : '') + 'Ngữ pháp');
  protected readonly noticing = computed(() =>
    (this.content()?.noticing ?? []).map((n) => ({
      sentence: n.sentence,
      parts: markHighlights(n.sentence, n.highlights ?? []),
    })),
  );
  protected readonly hasQuestion = computed(() => {
    const c = this.content();
    return !!c && c.questionVi.trim() !== '' && (c.options ?? []).length > 0;
  });
  protected readonly exercises = computed(() => this.content()?.exercises ?? []);
  protected readonly exercise = computed<GrammarExercise | null>(() => this.exercises()[this.index()] ?? null);
  protected readonly isLast = computed(() => this.index() >= this.exercises().length - 1);
  protected readonly progress = computed(() => {
    const n = this.exercises().length;
    return n ? ((this.index() + (this.answered() ? 1 : 0)) / n) * 100 : 0;
  });
  /** The link to the full grammar point shows at the end: after the last exercise, or with the rule when there is none. */
  protected readonly showLink = computed(() => {
    if (!this.grammarPointId()) {
      return false;
    }
    if (this.phase() === 'practice') {
      return this.isLast() && this.answered();
    }
    return this.phase() === 'rule' && this.exercises().length === 0;
  });
  /** The Kiểm tra button of the bar, for a fill, reorder or fix not checked yet. */
  protected readonly canCheck = computed(() => this.exerciseView()?.canCheck() ?? false);

  protected listen(text: string): void {
    this.speech.speak(text, 1);
  }

  protected choose(i: number): void {
    if (this.chosen() === null) {
      this.chosen.set(i);
    }
  }

  protected toRule(): void {
    this.phase.set('rule');
  }

  protected startPractice(): void {
    if (this.exercises().length === 0) {
      this.done.emit();
      return;
    }
    this.index.set(0);
    this.answered.set(false);
    this.correct.set(0);
    this.phase.set('practice');
  }

  protected onAnswered(ok: boolean): void {
    this.answered.set(true);
    if (ok) {
      this.correct.update((c) => c + 1);
    }
  }

  protected check(): void {
    this.exerciseView()?.check();
  }

  protected next(): void {
    if (this.isLast()) {
      this.done.emit();
      return;
    }
    this.index.update((i) => i + 1);
    this.answered.set(false);
  }

  /** ← goes back a phase first, then to the step before. */
  protected goBack(): void {
    switch (this.phase()) {
      case 'practice':
        this.phase.set('rule');
        break;
      case 'rule':
        this.phase.set('notice');
        break;
      default:
        this.back.emit();
    }
  }

  /** Whether the bar shows ←. */
  protected readonly showBack = computed(() => this.canBack() || (!!this.content() && this.phase() !== 'notice'));
}
