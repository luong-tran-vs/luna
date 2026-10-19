import {
  afterRenderEffect,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  input,
  output,
  viewChild,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spellings, spokenForms } from '../irregular-verbs.logic';
import { VERB_EXAMPLES, VERB_IPA, VERB_LESSONS, VERB_SAY_TIPS } from '../verb-lessons.data';
import { changedPart, exampleParts } from '../verb-study.logic';

/** Reading speed of the forms and the example. */
const RATE = 0.85;

/**
 * A verb's details, sliding up from the bottom when its row is tapped (design/tat-ca-de-xuat.png (mục dong-tu-gon-lai-v2)):
 * the three forms with the changed part marked and how each is said, a tip for forms often said
 * wrong, an example with the verb marked, listening, "Đánh dấu đã học" and the lesson it is in.
 * Escape or the backdrop closes it; Tab stays inside.
 */
@Component({
  selector: 'lu-verb-sheet',
  imports: [RouterLink, Icon],
  templateUrl: './verb-sheet.html',
  styleUrl: './verb-sheet.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown.escape)': 'close()' },
})
export class VerbSheet {
  private readonly speech = inject(SpeechService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  /** The verb shown; null: closed. */
  readonly verb = input<IrregularVerb | null>(null);
  readonly learned = input(false);
  /** Whether the "Đã học" button is there (not when the learner's verbs could not be loaded). */
  readonly canMark = input(false);
  readonly closed = output<void>();
  readonly toggleLearned = output<string>();

  private readonly closeButton = viewChild<ElementRef<HTMLButtonElement>>('closeButton');

  protected readonly canSpeak = this.speech.supported;
  protected readonly lesson = computed(() => {
    const v = this.verb();
    return v ? (VERB_LESSONS.find((l) => l.bases.includes(v.base)) ?? null) : null;
  });
  protected readonly forms = computed(() => {
    const v = this.verb();
    if (!v) {
      return [];
    }
    const ipa = VERB_IPA[v.base];
    return [v.base, v.past, v.participle].map((form, i) => ({
      label: ['V1', 'V2', 'V3'][i],
      parts: i === 0 ? ([form, ''] as [string, string]) : changedPart(v.base, spellings(form)[0]),
      alt: spellings(form).length > 1 ? form : '',
      ipa: ipa ? ipa[i].map((x) => `/${x}/`).join(', ') : '',
    }));
  });
  protected readonly tip = computed(() => {
    const v = this.verb();
    return v ? (VERB_SAY_TIPS[v.base] ?? v.noteVi ?? '') : '';
  });
  protected readonly example = computed(() => {
    const v = this.verb();
    const ex = v ? VERB_EXAMPLES[v.base] : undefined;
    return v && ex ? { parts: exampleParts(ex.en, v), en: ex.en, vi: ex.vi } : null;
  });

  constructor() {
    afterRenderEffect(() => {
      if (this.verb()) {
        this.closeButton()?.nativeElement.focus();
      }
    });
  }

  protected close(): void {
    if (this.verb()) {
      this.closed.emit();
    }
  }

  protected listen(text: string): void {
    this.speech.speak(text, RATE);
  }

  protected listenForms(v: IrregularVerb): void {
    this.speech.speak(spokenForms(v), RATE);
  }

  /** Keeps Tab focus inside the sheet while it is open. */
  protected trapTab(event: KeyboardEvent): void {
    if (event.key !== 'Tab') {
      return;
    }
    const focusable = Array.from(
      this.host.nativeElement.querySelectorAll<HTMLElement>('.sheet button, .sheet a[href]'),
    ).filter((e) => !e.hasAttribute('disabled'));
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  }
}
