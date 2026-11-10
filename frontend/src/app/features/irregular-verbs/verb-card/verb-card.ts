import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';

import { SpeechService } from '../../../core/services/speech.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IrregularVerb } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';
import { VERB_EXAMPLES, VERB_IPA, VERB_SAY_TIPS } from '../verb-lessons.data';
import { changedPart, exampleParts, spokenForm, threeForms } from '../verb-study.logic';
import { VERB_USAGE } from '../verb-usage.data';

/** Reading speed of the forms and sentences. */
const RATE = 0.85;
/** Pause between two forms read in a row, in ms: a steady beat. */
const BEAT_GAP = 350;

interface Use {
  label: string;
  form: 'V1' | 'V2' | 'V3';
  en: string;
  vi: string;
}

/**
 * One verb of "Làm quen" (design/de-xuat-dong-tu.png part 2, added 2026-11-10): its picture when the
 * admin drew one, the three forms as beats to tap (or read in a row, the beat being read lit), the
 * mistake learners often make, and a second face "Dùng khi nào" with a past sentence (V2) and a
 * perfect one (have + V3).
 */
@Component({
  selector: 'lu-verb-card',
  imports: [Icon],
  templateUrl: './verb-card.html',
  styleUrl: './verb-card.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbCard {
  private readonly speech = inject(SpeechService);

  readonly verb = input.required<IrregularVerb>();
  readonly tag = input('');
  /** The verb's picture; '' hides the frame. */
  readonly picture = input('');

  protected readonly canSpeak = this.speech.supported;
  protected readonly face = signal<'word' | 'use'>('word');
  /** The beat being read (0: V1, 1: V2, 2: V3), or -1. */
  protected readonly beat = signal(-1);
  private run = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  protected readonly forms = computed(() => threeForms(this.verb()));
  /** Each form cut where it changes from the base, to colour the change. */
  protected readonly cuts = computed(() => {
    const [base, past, participle] = this.forms();
    return [changedPart(past, base), changedPart(base, past), changedPart(base, participle)];
  });
  protected readonly ipa = computed(() =>
    [0, 1, 2].map((i) => VERB_IPA[this.verb().base]?.[i].map((x) => `/${x}/`).join(', ') ?? ''),
  );
  /** The mistake learners make, else a tip on saying it, else the verb's note. */
  protected readonly mistake = computed(() => {
    const v = this.verb();
    return VERB_USAGE[v.base]?.mistakeVi ?? VERB_SAY_TIPS[v.base] ?? v.noteVi ?? '';
  });
  /** Other spellings of a form, "burnt/burned". */
  protected readonly alternatives = computed(() =>
    [this.verb().past, this.verb().participle].map((f) => (spellings(f).length > 1 ? f : '')),
  );
  protected readonly uses = computed<Use[]>(() => {
    const v = this.verb();
    const u = VERB_USAGE[v.base];
    if (u) {
      return [
        { label: 'Đã xảy ra (quá khứ)', form: 'V2', ...u.past },
        { label: 'Đã … rồi (have + V3)', form: 'V3', ...u.perfect },
      ];
    }
    const ex = VERB_EXAMPLES[v.base];
    return ex ? [{ label: 'Câu ví dụ', form: 'V2', ...ex }] : [];
  });

  constructor() {
    // A new verb starts on its word face, nothing being read.
    effect(() => {
      this.verb();
      untracked(() => {
        this.face.set('word');
        this.stopBeats();
      });
    });
    inject(DestroyRef).onDestroy(() => this.stopBeats());
  }

  protected parts(sentence: string): { text: string; verb: boolean }[] {
    return exampleParts(sentence, this.verb());
  }

  protected say(i: number): void {
    this.stopBeats();
    this.beat.set(i);
    const run = this.run;
    this.speech.speak(spokenForm(this.verb().base, this.forms()[i], i), RATE, {
      ended: () => {
        if (run === this.run) {
          this.beat.set(-1);
        }
      },
    });
  }

  /** Reads V1, V2, V3 one after another, lighting the beat being read. */
  protected sayAll(): void {
    this.stopBeats();
    const run = this.run;
    const next = (i: number) => {
      if (run !== this.run) {
        return;
      }
      if (i > 2) {
        this.beat.set(-1);
        return;
      }
      this.beat.set(i);
      this.speech.speak(spokenForm(this.verb().base, this.forms()[i], i), RATE, {
        ended: () => (this.timer = setTimeout(() => next(i + 1), BEAT_GAP)),
        failed: () => this.beat.set(-1),
      });
    };
    next(0);
  }

  protected listen(text: string): void {
    this.stopBeats();
    this.speech.speak(text, RATE);
  }

  private stopBeats(): void {
    this.run++;
    clearTimeout(this.timer);
    this.beat.set(-1);
  }
}
