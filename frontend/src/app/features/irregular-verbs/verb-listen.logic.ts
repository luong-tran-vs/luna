import { VerbProgress } from '../../core/models/verbs';
import { shuffle } from '../lesson/lesson-detail/practice-logic';
import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { spellings } from './irregular-verbs.logic';
import { Form, isLearned, spokenForm } from './verb-study.logic';

/**
 * "Luyện nghe" (design/de-xuat-dong-tu.png part 8, added 2026-11-10): the learner hears a V2 or V3
 * read by the browser voice and picks the verb it comes from.
 */

/** Questions in one round. */
export const LISTEN_SIZE = 10;
/** Choices of a question. */
const CHOICES = 4;

/**
 * Forms the browser voice reads as another word, left out of the questions: "wound" (of wind) is
 * read /wuːnd/, "bound"… are fine. A form is also left out when it is spelt like the base (cut).
 */
const MISREAD: ReadonlySet<string> = new Set(['wound']);

export interface ListenQuestion {
  verb: IrregularVerb;
  form: Form;
  /** The form heard, as written. */
  word: string;
  /** What the voice is given for it ("red" for read). */
  spoken: string;
  /** V2 and V3 are the same word. */
  both: boolean;
  options: IrregularVerb[];
}

/** Every form of a verb, all spellings. */
function allForms(v: IrregularVerb): Set<string> {
  return new Set([v.base, ...spellings(v.past), ...spellings(v.participle)]);
}

/** The forms of a verb worth hearing: V2 or V3 not spelt like the base, not misread. */
function hearable(v: IrregularVerb): { form: Form; word: string }[] {
  const out: { form: Form; word: string }[] = [];
  const past = spellings(v.past)[0];
  const participle = spellings(v.participle)[0];
  if (past !== v.base && !MISREAD.has(past)) {
    out.push({ form: 'past', word: past });
  }
  if (participle !== v.base && participle !== past && !MISREAD.has(participle)) {
    out.push({ form: 'participle', word: participle });
  }
  return out;
}

/**
 * The verbs a round asks: those given (a lesson's), else the learned ones once there are enough,
 * else the common ones. Only verbs with a form worth hearing.
 */
export function listenPool(
  progress: ReadonlyMap<string, VerbProgress>,
  only: readonly IrregularVerb[] = [],
  verbs: readonly IrregularVerb[] = IRREGULAR_VERBS,
): IrregularVerb[] {
  const ok = (v: IrregularVerb) => hearable(v).length > 0;
  if (only.length) {
    return only.filter(ok);
  }
  const learned = verbs.filter((v) => isLearned(progress.get(v.base)) && ok(v));
  return learned.length >= LISTEN_SIZE ? learned : verbs.filter((v) => v.common && ok(v));
}

/**
 * A round of questions over the pool, each verb once (up to LISTEN_SIZE). The wrong choices are
 * other verbs none of whose forms is the word heard ("lay" is a form of lie: lay itself is never
 * offered beside it).
 */
export function listenRound(
  pool: readonly IrregularVerb[],
  seed: number,
  size = LISTEN_SIZE,
  verbs: readonly IrregularVerb[] = IRREGULAR_VERBS,
): ListenQuestion[] {
  const common = verbs.filter((v) => v.common);
  return shuffle(
    pool.filter((v) => hearable(v).length > 0),
    seed,
  )
    .slice(0, size)
    .map((v, i) => {
      const forms = hearable(v);
      const pick = forms[(seed + i) % forms.length];
      const others = shuffle(
        [...pool, ...common].filter(
          (o, k, arr) => o.base !== v.base && arr.indexOf(o) === k && !allForms(o).has(pick.word),
        ),
        seed + i + 1,
      ).slice(0, CHOICES - 1);
      const past = spellings(v.past)[0];
      return {
        verb: v,
        form: pick.form,
        word: pick.word,
        spoken: spokenForm(v.base, pick.word, pick.form === 'past' ? 1 : 2),
        both: past === spellings(v.participle)[0],
        options: shuffle([v, ...others], seed + i + 2),
      };
    });
}

/** What the answer says the word is: "V2 và V3 của teach", "V3 của go". */
export function formOf(q: ListenQuestion): string {
  if (q.both) {
    return `V2 và V3 của ${q.verb.base}`;
  }
  return `${q.form === 'past' ? 'V2' : 'V3'} của ${q.verb.base}`;
}
