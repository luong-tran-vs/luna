import { VerbProgress } from '../../../core/models/verbs';
import { IrregularVerb } from '../../irregular-verbs/irregular-verbs.data';
import {
  answerDuel,
  currentVerb,
  duelAnswer,
  duelOrder,
  duelOver,
  gradeDuel,
  nextDuel,
  startDuel,
} from './verb-duel.logic';

const verb = (base: string, past: string, participle: string, common = true): IrregularVerb => ({
  base,
  past,
  participle,
  meaningVi: 'nghĩa ' + base,
  common,
});

const learn = verb('learn', 'learnt/learned', 'learnt/learned');
const go = verb('go', 'went', 'gone');
const NOW = new Date('2026-11-16T10:00:00Z');

function progress(base: string, extra: Partial<VerbProgress> = {}): VerbProgress {
  return {
    base,
    learnedAt: '2026-11-01T00:00:00Z',
    level: 2,
    dueAt: '2026-11-30T00:00:00Z',
    right: 1,
    wrong: 0,
    ...extra,
  };
}

describe('verb duel grading', () => {
  it('accepts every spelling, any case, extra spaces ignored', () => {
    expect(gradeDuel(learn, 'learnt', 'learned')).toEqual(
      expect.objectContaining({ past: true, participle: true }),
    );
    expect(gradeDuel(learn, '  LEARNED ', 'Learnt')).toEqual(
      expect.objectContaining({ past: true, participle: true }),
    );
    expect(gradeDuel(go, 'went', 'went')).toEqual(
      expect.objectContaining({ past: true, participle: false }),
    );
    expect(gradeDuel(go, '', 'gone').past).toBe(false);
  });

  it('shows both forms as the answer', () => {
    expect(duelAnswer(go)).toBe('V2 went · V3 gone');
  });
});

describe('verb duel round', () => {
  it('adds to the score and the streak when both forms are right', () => {
    let s = startDuel([go, learn]);
    s = answerDuel(s, 'Went', 'gone');
    expect(s.score).toBe(1);
    expect(s.streak).toBe(1);
    expect(s.lives).toBe(3);
    expect(answerDuel(s, 'x', 'y')).toBe(s); // a second check is ignored
    s = answerDuel(nextDuel(s), 'learned', 'learnt');
    expect(s.score).toBe(2);
    expect(s.streak).toBe(2);
  });

  it('costs one life for a wrong form, one box or both, and ends the streak', () => {
    let s = answerDuel(startDuel([learn, go, go]), 'learned', 'learnt');
    s = answerDuel(nextDuel(s), 'went', 'goned');
    expect(s.lives).toBe(2);
    expect(s.streak).toBe(0);
    expect(s.misses).toEqual([{ base: 'go', form: 'participle', typed: 'goned' }]);
    s = answerDuel(nextDuel(s), ' goed ', 'goed');
    expect(s.lives).toBe(1);
    expect(s.misses.length).toBe(3);
    expect(s.missed.map((v) => v.base)).toEqual(['go']);
    expect(s.score).toBe(1);
  });

  it('ends when the three lives are gone', () => {
    let s = startDuel([go, go, go, go]);
    for (let i = 0; i < 3; i++) {
      expect(duelOver(s)).toBe(false);
      s = nextDuel(answerDuel(s, 'x', 'y'));
    }
    expect(s.lives).toBe(0);
    expect(duelOver(s)).toBe(true);
    expect(answerDuel(s, 'went', 'gone')).toBe(s);
  });

  it('adds a new order when the list runs out', () => {
    let s = startDuel([go]);
    s = nextDuel(answerDuel(s, 'went', 'gone'), [learn]);
    expect(currentVerb(s)?.base).toBe('learn');
  });
});

describe('verb duel order', () => {
  const verbs = [
    verb('be', 'was/were', 'been'),
    verb('go', 'went', 'gone'),
    verb('see', 'saw', 'seen'),
    verb('eat', 'ate', 'eaten'),
    verb('bid', 'bid', 'bid', false),
    verb('abide', 'abode', 'abode', false),
  ];

  it('puts due and "hay sai" verbs first, then learned, then common unlearned', () => {
    const p = new Map<string, VerbProgress>([
      ['bid', progress('bid')],
      ['eat', progress('eat', { dueAt: '2026-11-10T00:00:00Z' })],
      [
        'see',
        progress('see', {
          mistakes: [
            { form: 'past', typed: 'seed', at: '2026-11-14T00:00:00Z' },
            { form: 'past', typed: 'sawn', at: '2026-11-15T00:00:00Z' },
          ],
          cleanRun: 0,
        }),
      ],
    ]);
    const order = duelOrder(p, NOW, 7, verbs).map((v) => v.base);
    expect(order.slice(0, 2).sort()).toEqual(['eat', 'see']);
    expect(order[2]).toBe('bid');
    expect(order.slice(3).sort()).toEqual(['be', 'go']);
    expect(order).not.toContain('abide'); // neither learned nor common
  });

  it('fills a round of few learned verbs with common ones, without repeats', () => {
    const order = duelOrder(new Map(), NOW, 3, verbs).map((v) => v.base);
    expect(order.sort()).toEqual(['be', 'eat', 'go', 'see']);
    expect(new Set(order).size).toBe(order.length);
  });
});
