import { VerbProgress } from '../../../core/models/verbs';
import { IrregularVerb } from '../irregular-verbs.data';
import { gradeForm, inSet, isDue, pickRound, summarize } from './verb-practice.logic';

const verb = (base: string, past = base, participle = base, common = true): IrregularVerb => ({
  base,
  past,
  participle,
  meaningVi: base,
  common,
});
const rec = (base: string, over: Partial<VerbProgress> = {}): [string, VerbProgress] => [
  base,
  { base, level: 1, dueAt: null, right: 1, wrong: 0, ...over },
];

const now = new Date('2026-10-16T08:00:00Z');
const past = '2026-10-15T08:00:00Z';
const later = '2026-10-20T08:00:00Z';

describe('verb practice logic', () => {
  const verbs = [
    verb('cut'),
    verb('buy', 'bought', 'bought'),
    verb('come', 'came', 'come'),
    verb('go', 'went', 'gone'),
    verb('abide', 'abode/abided', 'abode/abided', false),
  ];

  it('takes the verbs of a set', () => {
    expect(inSet(verbs, 'common').map((v) => v.base)).toEqual(['cut', 'buy', 'come', 'go']);
    expect(inSet(verbs, 'all').length).toBe(5);
    expect(inSet(verbs, 'ABB').map((v) => v.base)).toEqual(['buy', 'abide']);
    expect(inSet(verbs, 'ABC').map((v) => v.base)).toEqual(['go']);
  });

  it('counts the known and the due verbs', () => {
    const progress = new Map([
      rec('cut', { dueAt: past }),
      rec('buy', { dueAt: later }),
      rec('go', { level: 0, wrong: 1 }),
    ]);
    expect(summarize(verbs, progress, now)).toEqual({ total: 5, known: 2, due: 1 });
    expect(isDue(progress.get('cut'), now)).toBe(true);
    expect(isDue(progress.get('buy'), now)).toBe(false);
    expect(isDue(progress.get('go'), now)).toBe(false);
  });

  it('asks the due verbs first, then the missed, then new ones, up to the size', () => {
    const progress = new Map([
      rec('cut', { dueAt: past }),
      rec('buy', { dueAt: later }),
      rec('go', { level: 0, dueAt: null, wrong: 2 }),
    ]);
    const round = pickRound(verbs, progress, now, 7, 3).map((v) => v.base);
    expect(round.length).toBe(3);
    expect(round).toContain('cut');
    expect(round).toContain('go');
    expect(round).not.toContain('buy');
    // Everything not known fits: the whole set but the known verb that is not due.
    expect(
      pickRound(verbs, progress, now, 7)
        .map((v) => v.base)
        .sort(),
    ).toEqual(['abide', 'come', 'cut', 'go']);
  });

  it('reviews the verbs due soonest when all are known and none is due', () => {
    const progress = new Map(verbs.map((v, i) => rec(v.base, { dueAt: `2026-10-2${i}T08:00:00Z` })));
    expect(
      pickRound(verbs, progress, now, 1, 2)
        .map((v) => v.base)
        .sort(),
    ).toEqual(['buy', 'cut']);
  });

  it('grades a form against any of its spellings', () => {
    expect(gradeForm('went', 'went')).toBe(true);
    expect(gradeForm('  Went ', 'went')).toBe(true);
    expect(gradeForm('abided', 'abode/abided')).toBe(true);
    expect(gradeForm('abode/abided', 'abode/abided')).toBe(true);
    expect(gradeForm('were', 'was/were')).toBe(true);
    expect(gradeForm('goed', 'went')).toBe(false);
    expect(gradeForm('', 'went')).toBe(false);
  });
});
