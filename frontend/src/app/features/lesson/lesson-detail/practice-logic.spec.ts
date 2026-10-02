import { PracticeView } from '../../../core/models/practice';
import {
  checkFill,
  checkTranslation,
  clock,
  hasDialogue,
  hasExamples,
  hasFill,
  hasTranslations,
  levelLabel,
  nextEmptyBlank,
  shuffle,
  summary,
} from './practice-logic';

const empty: PracticeView = {
  status: 'none',
  lessonNumber: 0,
  objectiveVi: '',
  examples: [],
  dialogue: null,
  fill: null,
  grammarTipVi: '',
  translations: [],
};

describe('practice logic', () => {
  describe('shuffle', () => {
    const items = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];

    it('is deterministic for a seed', () => {
      expect(shuffle(items, 42)).toEqual(shuffle(items, 42));
    });

    it('returns a permutation and leaves the input alone', () => {
      const out = shuffle(items, 7);
      expect([...out].sort()).toEqual(items);
      expect(items).toEqual(['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']);
    });

    it('gives different orders for different seeds', () => {
      const orders = new Set([1, 2, 3, 4, 5].map((s) => shuffle(items, s).join('')));
      expect(orders.size).toBeGreaterThan(1);
    });

    it('handles empty and single lists', () => {
      expect(shuffle([], 1)).toEqual([]);
      expect(shuffle(['x'], 1)).toEqual(['x']);
    });
  });

  it.each([
    ['A1', 'Cơ bản'],
    ['A2', 'Cơ bản'],
    ['B1', 'Trung cấp'],
    ['B2', 'Trung cấp'],
    ['C1', 'Nâng cao'],
    ['C2', 'Nâng cao'],
  ] as const)('labels level %s as %s', (level, label) => {
    expect(levelLabel(level)).toBe(label);
  });

  it('tells which parts of the practice exist', () => {
    expect([hasExamples(null), hasDialogue(null), hasFill(null), hasTranslations(null)]).toEqual([
      false,
      false,
      false,
      false,
    ]);
    expect([
      hasExamples(empty),
      hasDialogue(empty),
      hasFill(empty),
      hasTranslations(empty),
    ]).toEqual([false, false, false, false]);
    const full: PracticeView = {
      ...empty,
      examples: [{ lemma: 'meet', sentence: 'Nice to meet you.' }],
      dialogue: {
        speakers: ['Minh', 'Anna'],
        turns: [{ speaker: 0, text: 'Hi.', meaningVi: 'Chào.' }],
      },
      fill: { turns: [], blanks: [{ answer: 'meet' }], wordBank: ['meet'] },
      translations: [{ vi: 'Chào.', answer: ['Hi.'], tiles: ['Hi.'] }],
    };
    expect([hasExamples(full), hasDialogue(full), hasFill(full), hasTranslations(full)]).toEqual([
      true,
      true,
      true,
      true,
    ]);
  });

  describe('checkFill', () => {
    const blanks = [{ answer: 'meet' }, { answer: 'Name' }, { answer: 'from' }];

    it('ignores case and surrounding spaces', () => {
      expect(checkFill(['Meet', ' name ', 'from'], blanks)).toEqual({
        results: [true, true, true],
        correct: 3,
        total: 3,
      });
    });

    it('counts wrong and empty blanks as wrong', () => {
      expect(checkFill(['meet', 'from', null], blanks)).toEqual({
        results: [true, false, false],
        correct: 1,
        total: 3,
      });
    });
  });

  describe('nextEmptyBlank', () => {
    it('finds the first empty blank', () => {
      expect(nextEmptyBlank([null, null])).toBe(0);
      expect(nextEmptyBlank(['a', null, null])).toBe(1);
    });

    it('looks after the given blank first, then wraps around', () => {
      expect(nextEmptyBlank([null, 'a', null], 0)).toBe(2);
      expect(nextEmptyBlank([null, 'a', 'b'], 1)).toBe(0);
    });

    it('returns -1 when every blank is filled', () => {
      expect(nextEmptyBlank(['a', 'b'], 0)).toBe(-1);
      expect(nextEmptyBlank([])).toBe(-1);
    });
  });

  describe('checkTranslation', () => {
    const answer = ['Nice', 'to', 'meet', 'you,', 'Anna.'];

    it('needs the same tiles in the same order', () => {
      expect(checkTranslation(['Nice', 'to', 'meet', 'you,', 'Anna.'], answer)).toBe(true);
      expect(checkTranslation(['to', 'Nice', 'meet', 'you,', 'Anna.'], answer)).toBe(false);
      expect(checkTranslation(['Nice', 'to', 'meet'], answer)).toBe(false);
    });

    it('compares the text of each tile exactly', () => {
      expect(checkTranslation(['nice', 'to', 'meet', 'you,', 'Anna.'], answer)).toBe(false);
      expect(checkTranslation(['Nice', 'to', 'meet', 'you', 'Anna.'], answer)).toBe(false);
    });

    it('lets tiles with the same text replace each other', () => {
      // Built from the second "the" tile: only the text matters.
      expect(
        checkTranslation(['the', 'cat', 'and', 'the', 'dog'], ['the', 'cat', 'and', 'the', 'dog']),
      ).toBe(true);
    });
  });

  describe('summary', () => {
    it('counts checked results', () => {
      expect(summary(5, { correct: 3, total: 5 }, [true, false, true])).toEqual({
        fill: { correct: 3, total: 5 },
        translate: { correct: 2, total: 3 },
      });
    });

    it('counts unchecked parts as wrong', () => {
      expect(summary(4, null, [true, null, null])).toEqual({
        fill: { correct: 0, total: 4 },
        translate: { correct: 1, total: 3 },
      });
    });

    it('leaves out parts the lesson does not have', () => {
      expect(summary(0, null, [])).toEqual({ fill: null, translate: null });
    });
  });

  it('formats elapsed time as mm:ss', () => {
    expect(clock(0)).toBe('00:00');
    expect(clock(7.9)).toBe('00:07');
    expect(clock(72)).toBe('01:12');
  });
});
