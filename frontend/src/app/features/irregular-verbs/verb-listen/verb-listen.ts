import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { SpeechService } from '../../../core/services/speech.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { newSeed } from '../../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from '../irregular-verbs.data';
import { VERB_IPA, VERB_LESSONS } from '../verb-lessons.data';
import { formOf, ListenQuestion, listenPool, listenRound } from '../verb-listen.logic';
import { VerbPictures } from '../verb-pictures.service';
import { VERB_SOUNDS } from '../verb-sounds.data';
import { lessonVerbs, spokenForm, threeForms } from '../verb-study.logic';

const RATE = 0.85;
const SLOW = 0.6;

/**
 * "Luyện nghe" (design/de-xuat-dong-tu.png part 8, added 2026-11-10). Nghe đoán từ: a V2 or V3 is
 * read aloud, the learner picks its verb, then sees its picture, how each form is said and hears
 * them again; ten questions over the learned verbs (or a lesson's, ?lesson=8). Đọc khác viết: forms
 * said otherwise than written, to hear one by one. Nothing is saved.
 */
@Component({
  selector: 'lu-verb-listen',
  imports: [RouterLink, Icon],
  templateUrl: './verb-listen.html',
  styleUrl: './verb-listen.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbListen {
  private readonly api = inject(VerbsApiService);
  private readonly speech = inject(SpeechService);
  private readonly injector = inject(Injector);
  private readonly heading = viewChild<ElementRef<HTMLElement>>('heading');

  protected readonly canSpeak = this.speech.supported;
  protected readonly pictures = inject(VerbPictures).urls;
  protected readonly tab = signal<'guess' | 'sounds'>('guess');
  /** The lesson the round is about (?lesson=8), or null for the learned verbs. */
  protected readonly lesson = computed(() => {
    const n = Number(this.route.snapshot?.queryParamMap.get('lesson'));
    return VERB_LESSONS.find((l) => l.number === n) ?? null;
  });
  private readonly route = inject(ActivatedRoute);
  private readonly progress = signal<ReadonlyMap<string, VerbProgress> | null>(null);

  protected readonly questions = signal<ListenQuestion[]>([]);
  protected readonly index = signal(0);
  protected readonly chosen = signal<string | null>(null);
  protected readonly score = signal(0);
  protected readonly finished = signal(false);
  protected readonly current = computed(() => this.questions()[this.index()] ?? null);
  protected readonly right = computed(() => this.chosen() === this.current()?.verb.base);

  protected readonly sounds = VERB_SOUNDS.map((s) => {
    const verb = IRREGULAR_VERBS.find((v) => v.base === s.base)!;
    const forms = threeForms(verb);
    return {
      ...s,
      verb,
      items: s.forms.map((i) => ({ index: i, word: forms[i], ipa: this.ipa(verb, i) })),
    };
  });

  constructor() {
    inject(VerbPictures).load();
    this.api.progress().subscribe({
      next: (r) => {
        this.progress.set(new Map(r.verbs.map((v) => [v.base, v])));
        this.newRound();
      },
      // Without the learned verbs the round is over the common ones.
      error: () => {
        this.progress.set(new Map());
        this.newRound();
      },
    });
  }

  protected newRound(): void {
    const l = this.lesson();
    const pool = listenPool(this.progress() ?? new Map(), l ? lessonVerbs(l, IRREGULAR_VERBS) : []);
    this.questions.set(listenRound(pool, newSeed()));
    this.index.set(0);
    this.chosen.set(null);
    this.score.set(0);
    this.finished.set(false);
  }

  /** How form i of a verb is said, "/bɔːt/"; '' when not written. */
  protected ipa(v: IrregularVerb, i: number): string {
    return VERB_IPA[v.base]?.[i].map((x) => `/${x}/`).join(', ') ?? '';
  }

  protected formIndex(q: ListenQuestion): number {
    return q.form === 'past' ? 1 : 2;
  }

  protected answer(q: ListenQuestion): string {
    return formOf(q);
  }

  protected forms(v: IrregularVerb): string[] {
    return threeForms(v);
  }

  protected play(slow = false): void {
    const q = this.current();
    if (q) {
      this.speech.speak(q.spoken, slow ? SLOW : RATE);
    }
  }

  protected say(v: IrregularVerb, i: number): void {
    this.speech.speak(spokenForm(v.base, threeForms(v)[i], i), RATE);
  }

  protected choose(base: string): void {
    if (this.chosen() !== null) {
      return;
    }
    this.chosen.set(base);
    if (base === this.current()?.verb.base) {
      this.score.update((s) => s + 1);
    }
  }

  protected next(): void {
    if (this.index() + 1 >= this.questions().length) {
      this.finished.set(true);
    } else {
      this.index.update((i) => i + 1);
      this.chosen.set(null);
      // The tap is the gesture the browser needs to speak.
      this.play();
    }
    afterNextRender(() => this.heading()?.nativeElement.focus(), { injector: this.injector });
  }
}
