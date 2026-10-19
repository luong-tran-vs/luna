import { VerbProgress } from '../../core/models/verbs';
import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { spellings } from './irregular-verbs.logic';
import { VERB_EXAMPLES, VERB_IPA, VERB_LESSONS } from './verb-lessons.data';
import {
  changedPart,
  choiceQuestion,
  dueLabel,
  exampleParts,
  gradeForm,
  isDue,
  learnedIn,
  learnedVerbs,
  lessonVerbs,
  modelForms,
  nextLesson,
  regularPast,
  reviewRound,
  reviewSummary,
} from './verb-study.logic';

const now = new Date('2026-10-16T08:00:00');
const rec = (base: string, over: Partial<VerbProgress> = {}): [string, VerbProgress] => [
  base,
  { base, learnedAt: '2026-10-10T08:00:00', level: 1, dueAt: null, right: 0, wrong: 0, ...over },
];
const verb = (base: string): IrregularVerb => IRREGULAR_VERBS.find((v) => v.base === base)!;

describe('verb lessons data', () => {
  it('puts every verb in exactly one lesson, the common ones first, numbered in order', () => {
    const bases = VERB_LESSONS.flatMap((l) => l.bases);
    expect([...bases].sort()).toEqual(IRREGULAR_VERBS.map((v) => v.base).sort());
    expect(new Set(bases).size).toBe(bases.length);
    expect(VERB_LESSONS.map((l) => l.number)).toEqual(VERB_LESSONS.map((_, i) => i + 1));
    const common = new Set(IRREGULAR_VERBS.filter((v) => v.common).map((v) => v.base));
    for (const l of VERB_LESSONS) {
      // Lessons 1–18: only common verbs; "Nâng cao": none.
      expect(
        l.bases.every((b) => common.has(b) === !l.advanced),
        `lesson ${l.number}`,
      ).toBe(true);
    }
    expect(VERB_LESSONS.filter((l) => !l.advanced).length).toBe(18);
  });

  it('gives every verb an example with one of its forms, and how each spelling of each form is said', () => {
    for (const v of IRREGULAR_VERBS) {
      const ex = VERB_EXAMPLES[v.base];
      expect(ex?.vi, v.base).toBeTruthy();
      expect(
        exampleParts(ex.en, v).some((p) => p.verb),
        `${v.base}: ${ex.en}`,
      ).toBe(true);
      const ipa = VERB_IPA[v.base];
      expect(ipa, v.base).toBeTruthy();
      [v.base, v.past, v.participle].forEach((form, i) =>
        expect(ipa[i].length, `${v.base} form ${i + 1}`).toBe(spellings(form).length),
      );
    }
  });
});

describe('verb study logic', () => {
  const lesson = VERB_LESSONS[7]; // -ought: buy, bring, think, fight, catch, teach

  it('takes the verbs of a lesson and counts the learned ones', () => {
    expect(lessonVerbs(lesson).map((v) => v.base)).toEqual(['buy', 'bring', 'think', 'fight', 'catch', 'teach']);
    const progress = new Map([rec('buy'), rec('think'), rec('fight', { learnedAt: null, level: 0 })]);
    expect(learnedIn(lesson, progress)).toBe(2);
  });

  it('suggests the first lesson with a verb not learned', () => {
    expect(nextLesson(VERB_LESSONS, new Map())?.number).toBe(1);
    const first = new Map(VERB_LESSONS[0].bases.map((b) => rec(b)));
    expect(nextLesson(VERB_LESSONS, first)?.number).toBe(2);
    const all = new Map(VERB_LESSONS.flatMap((l) => l.bases).map((b) => rec(b)));
    expect(nextLesson(VERB_LESSONS, all)).toBeNull();
  });

  it('sums up the learned verbs and the due ones', () => {
    const progress = new Map([
      rec('go', { dueAt: '2026-10-15T08:00:00' }),
      rec('see', { dueAt: '2026-10-16T20:00:00' }),
      rec('cut', { dueAt: '2026-10-25T08:00:00' }),
      rec('put', { learnedAt: null, level: 0, wrong: 2 }),
    ]);
    expect(reviewSummary(progress, now)).toEqual({ learned: 3, dueToday: 1, dueTomorrow: 1 });
    expect(isDue(progress.get('go'), now)).toBe(true);
    expect(isDue(progress.get('put'), now)).toBe(false);
    expect(learnedVerbs(progress, now).map((l) => [l.verb.base, l.due])).toEqual([
      ['go', true],
      ['see', false],
      ['cut', false],
    ]);
  });

  it('reviews the due verbs, or the soonest ones when none is due', () => {
    const due = new Map([rec('go', { dueAt: '2026-10-15T08:00:00' }), rec('cut', { dueAt: '2026-10-25T08:00:00' })]);
    expect(reviewRound(due, now, 1).map((v) => v.base)).toEqual(['go']);
    const none = new Map([rec('go', { dueAt: '2026-10-20T08:00:00' }), rec('cut', { dueAt: '2026-10-25T08:00:00' })]);
    expect(reviewRound(none, now, 1, 1).map((v) => v.base)).toEqual(['go']);
  });

  it('says when a verb comes back', () => {
    expect(dueLabel('2026-10-14T23:00:00', now)).toBe('quá 2 ngày');
    expect(dueLabel('2026-10-16T22:00:00', now)).toBe('hôm nay');
    expect(dueLabel('2026-10-17T01:00:00', now)).toBe('ngày mai');
    expect(dueLabel('2026-10-22T08:00:00', now)).toBe('sau 6 ngày');
    expect(dueLabel(null, now)).toBe('');
  });

  it('grades a typed form against any of its spellings', () => {
    expect(gradeForm(' Went ', 'went')).toBe(true);
    expect(gradeForm('burned', 'burnt/burned')).toBe(true);
    expect(gradeForm('were', 'was/were')).toBe(true);
    expect(gradeForm('goed', 'went')).toBe(false);
    expect(gradeForm('', 'went')).toBe(false);
  });

  it('builds a choice with the regular guess and the other verbs of the lesson as wrong options', () => {
    expect(regularPast('think')).toBe('thinked');
    expect(regularPast('make')).toBe('maked');
    expect(regularPast('fly')).toBe('flied');
    const q = choiceQuestion(verb('think'), 'past', lessonVerbs(lesson), 3);
    expect(q.answer).toBe('thought');
    expect(q.options.length).toBe(4);
    expect(q.options).toContain('thought');
    expect(q.options).toContain('thinked');
    expect(new Set(q.options).size).toBe(4);
    // A verb with the same three forms never offers its own form as a wrong option.
    const cut = choiceQuestion(verb('cut'), 'participle', lessonVerbs(VERB_LESSONS[0]), 1);
    expect(cut.options.filter((o) => o === 'cut').length).toBe(1);
  });

  it('marks the part of a form that differs from the base', () => {
    expect(changedPart('buy', 'bought')).toEqual(['b', 'ought']);
    expect(changedPart('cut', 'cut')).toEqual(['cut', '']);
    expect(changedPart('go', 'went')).toEqual(['', 'went']);
  });
});

describe('verb study display helpers', () => {
  it('shows a lesson pattern on its first verb', () => {
    expect(modelForms(VERB_LESSONS[3])).toBe('send → sent → sent');
    expect(modelForms(VERB_LESSONS[11])).toBe('be → was → been');
  });

  it('marks the forms of the verb in an example', () => {
    const buy = IRREGULAR_VERBS.find((v) => v.base === 'buy')!;
    expect(exampleParts('I bought a new bike.', buy)).toEqual([
      { text: 'I ', verb: false },
      { text: 'bought', verb: true },
      { text: ' a new bike.', verb: false },
    ]);
    const be = IRREGULAR_VERBS.find((v) => v.base === 'be')!;
    expect(
      exampleParts('We were at home, been there.', be)
        .filter((p) => p.verb)
        .map((p) => p.text),
    ).toEqual(['were', 'been']);
    // Inside another word it is not the verb.
    expect(exampleParts('Abuyer', buy)).toEqual([{ text: 'Abuyer', verb: false }]);
  });
});
