import { NgOptimizedImage } from '@angular/common';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  Injector,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';

import { GrammarMistake } from '../../../core/models/grammar-study';
import { Icon } from '../../../shared/components/icon/icon';

/** One of the two sentences offered for a mistake. */
interface Choice {
  text: string;
  right: boolean;
}

/**
 * The common mistakes of a point as "Câu nào đúng?", one per screen: the wrong and the right
 * sentence in a random order, the note shown once one is picked. Choosing makes the learner notice
 * the mistake better than reading a Sai / Đúng table; nothing is scored or recorded. The lesson's
 * bottom bar moves on with `next()` once `answered()`.
 */
@Component({
  selector: 'lu-mistake-check',
  imports: [Icon, NgOptimizedImage],
  templateUrl: './mistake-check.html',
  styleUrl: './mistake-check.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MistakeCheck {
  private readonly injector = inject(Injector);
  private readonly question = viewChild<ElementRef<HTMLElement>>('question');

  readonly mistakes = input.required<readonly GrammarMistake[]>();

  /** Index of the mistake on screen. */
  protected readonly index = signal(0);
  /** The choice picked for the mistake on screen, null before picking. */
  protected readonly picked = signal<Choice | null>(null);

  /** For each mistake, whether the right sentence comes first; drawn again when the list changes. */
  private readonly rightFirst = computed(() => this.mistakes().map(() => Math.random() < 0.5));

  protected readonly current = computed(() => this.mistakes()[this.index()] ?? null);
  protected readonly choices = computed<Choice[]>(() => {
    const m = this.current();
    if (!m) {
      return [];
    }
    const pair: Choice[] = [
      { text: m.wrong, right: false },
      { text: m.right, right: true },
    ];
    return this.rightFirst()[this.index()] ? pair.reverse() : pair;
  });
  /** A sentence of the mistake on screen has been picked. */
  readonly answered = computed(() => this.picked() !== null);
  readonly isLast = computed(() => this.index() >= this.mistakes().length - 1);

  protected pick(choice: Choice): void {
    if (!this.picked()) {
      this.picked.set(choice);
    }
  }

  /** Shows the next mistake, its question focused. */
  next(): void {
    if (this.isLast()) {
      return;
    }
    this.picked.set(null);
    this.index.update((i) => i + 1);
    afterNextRender(() => this.question()?.nativeElement.focus(), { injector: this.injector });
  }
}
