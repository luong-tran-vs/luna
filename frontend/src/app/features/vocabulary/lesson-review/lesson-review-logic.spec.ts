import { PracticeFill } from '../../../core/models/practice';
import { DueCard, VocabItem } from '../../../core/models/vocab';
import {
  Answer,
  badgeOf,
  buildBank,
  checkTiles,
  FillQuestion,
  LessonRow,
  mastery,
  missedWords,
  partCount,
  Question,
  resultAfter,
  scoreText,
  seenLine,
  roundOf,
  scores,
  suggested,
  VocabQuestion,
} from './lesson-review-logic';

const item = (lemma: string, meaningVi: string, pos = 'noun'): VocabItem => ({
  lemma,
  text: lemma,
  meaningVi,
  ipa: '',
  sentenceIndex: 0,
  sentence: `The ${lemma} is here.`,
  pos,
});

const ITEMS = [
  item('desk', 'bàn làm việc'),
  item('boss', 'sếp'),
  item('meeting', 'cuộc họp'),
  item('deadline', 'hạn chót'),
  item('work', 'làm việc', 'verb'),
];

const due: DueCard = {
  id: 'c1',
  text: 'meeting',
  lemma: 'meeting',
  ipa: '',
  meaningVi: 'cuộc họp',
  contextSentence: 'The meeting starts.',
  contextSentenceVi: 'Cuộc họp bắt đầu.',
  lessonId: 'l1',
  source: 'ai',
  createdAt: '',
  due: '',
  reps: 2,
  state: 'review',
  intervals: { again: 60, hard: 300, good: 600, easy: 86400 },
};

const FILL: PracticeFill = {
  turns: [
    {
      speaker: 0,
      turnIndex: 0,
      meaningVi: 'Sếp tôi có cuộc họp.',
      parts: [{ text: 'My ' }, { blank: 0 }, { text: ' has a ' }, { blank: 1 }, { text: '.' }],
    },
  ],
  blanks: [{ answer: 'boss' }, { answer: 'meeting' }],
  wordBank: ['boss', 'meeting', 'desk'],
};

describe('lesson review logic', () => {
  it('asks the due cards first, then the other words, each once', () => {
    const bank = buildBank(ITEMS, [due], null, [], 1);
    const vocab = bank.vocab as VocabQuestion[];
    expect(vocab[0].lemma).toBe('meeting');
    expect(vocab[0].card).toBe(due);
    expect(vocab[0].sentenceVi).toBe('Cuộc họp bắt đầu.');
    expect(vocab.map((q) => q.lemma).sort()).toEqual([
      'boss',
      'deadline',
      'desk',
      'meeting',
      'work',
    ]);
    expect(vocab.slice(1).every((q) => q.card === null)).toBe(true);
    for (const q of vocab) {
      expect(q.choices.options).toHaveLength(4);
      expect(q.choices.options[q.choices.answer]).toBe(q.meaningVi);
    }
  });

  it('makes one fill-in question per blank, the other blanks filled in', () => {
    const fill = buildBank(ITEMS, [], FILL, [], 1).fill as FillQuestion[];
    const byAnswer = Object.fromEntries(fill.map((q) => [q.answer, q]));
    expect(byAnswer['boss'].before).toBe('My ');
    expect(byAnswer['boss'].after).toBe(' has a meeting.');
    expect(byAnswer['meeting'].before).toBe('My boss has a ');
    expect(byAnswer['meeting'].vi).toBe('Sếp tôi có cuộc họp.');
    expect(byAnswer['meeting'].choices.options[byAnswer['meeting'].choices.answer]).toBe('meeting');
  });

  it('keeps up to 8, 4 and 3 questions of the parts chosen, in order', () => {
    const many = Array.from({ length: 12 }, (_, i) => item(`w${i}`, `nghĩa ${i}`));
    const translations = Array.from({ length: 5 }, (_, i) => ({
      vi: `Câu ${i}`,
      answer: ['I', 'go.'],
      tiles: ['go.', 'I', 'we'],
    }));
    const bank = buildBank(many, [], FILL, translations, 2);
    expect(roundOf(bank, ['vocab', 'fill', 'translate']).map((q) => q.kind)).toEqual([
      ...Array(8).fill('vocab'),
      'fill',
      'fill',
      ...Array(3).fill('translate'),
    ]);
    expect(roundOf(bank, ['translate']).map((q) => q.kind)).toEqual([
      'translate',
      'translate',
      'translate',
    ]);
  });

  it('always asks the words due, even with Từ vựng left out or past 8', () => {
    const many = Array.from({ length: 12 }, (_, i) => item(`w${i}`, `nghĩa ${i}`));
    const cards = Array.from({ length: 10 }, (_, i) => ({
      ...due,
      id: `c${i}`,
      lemma: `w${i}`,
      text: `w${i}`,
    }));
    const bank = buildBank(many, cards.slice(0, 1), FILL, [], 2);
    const round = roundOf(bank, ['fill']);
    expect(round.map((q) => q.kind)).toEqual(['vocab', 'fill', 'fill']);
    expect(round[0].kind === 'vocab' && round[0].card?.id).toBe('c0');

    const all = buildBank(many, cards, null, [], 2);
    expect(partCount(all, 'vocab')).toBe(10);
    expect(roundOf(all, ['vocab']).filter((q) => q.kind === 'vocab' && q.card)).toHaveLength(10);
  });

  it('checks tiles word for word', () => {
    expect(checkTiles(['I', 'go.'], ['I', 'go.'])).toBe(true);
    expect(checkTiles(['go.', 'I'], ['I', 'go.'])).toBe(false);
    expect(checkTiles(['I'], ['I', 'go.'])).toBe(false);
  });

  it('scores each part and brings back the words got wrong', () => {
    const bank = buildBank(ITEMS, [due], FILL, [], 1);
    const [dueQ, otherQ] = bank.vocab;
    const [fillQ] = bank.fill;
    const answers: Answer[] = [
      { question: dueQ, correct: false, given: null },
      { question: otherQ, correct: false, given: 'x' },
      { question: bank.vocab[2], correct: true, given: 'y' },
      { question: fillQ, correct: false, given: 'desk' },
    ];
    expect(scores(answers)).toEqual([
      { part: 'vocab', correct: 1, total: 3 },
      { part: 'fill', correct: 0, total: 1 },
    ]);
    // The due card is rated instead.
    expect(missedWords(answers)).toEqual([
      (otherQ as VocabQuestion).lemma,
      (fillQ as FillQuestion).answer,
    ]);
  });

  describe('the list of lessons', () => {
    const now = Date.parse('2026-10-16T05:00:00Z');
    const row = (
      id: string,
      stats: LessonRow['stats'],
      completedAt = '2026-10-01T00:00:00Z',
    ): LessonRow => ({
      id,
      title: id,
      topicName: '',
      completedAt,
      stats,
    });
    const stats = (due: number, rest: Partial<NonNullable<LessonRow['stats']>> = {}) => ({
      lessonId: 'x',
      cards: 10,
      due,
      remembered: 4,
      lastReview: null,
      nextDue: null,
      result: null,
      ...rest,
    });

    it('labels a lesson by its words', () => {
      expect(badgeOf(row('a', stats(3)), now)).toEqual({ tone: 'due', text: 'Cần ôn · 3' });
      expect(
        badgeOf(
          row(
            'a',
            stats(0, { lastReview: '2026-10-16T01:00:00Z', nextDue: '2026-10-20T00:00:00Z' }),
          ),
          now,
        ),
      ).toEqual({
        tone: 'done',
        text: 'Đã ôn',
      });
      expect(
        badgeOf(
          row(
            'a',
            stats(0, { lastReview: '2026-10-10T01:00:00Z', nextDue: '2026-10-18T05:00:00Z' }),
          ),
          now,
        ),
      ).toEqual({
        tone: 'later',
        text: 'Còn 2 ngày',
      });
      expect(badgeOf(row('a', null), now)).toEqual({ tone: 'new', text: 'Chưa ôn' });
      // A lesson review round today counts as reviewed too.
      const round = {
        correct: 3,
        total: 5,
        wrong: ['v:a', 'f:0'],
        reviewedAt: '2026-10-16T02:00:00Z',
      };
      const reviewed = row('a', stats(0, { lastReview: '2026-10-10T01:00:00Z', result: round }));
      expect(badgeOf(reviewed, now)).toEqual({ tone: 'done', text: 'Đã ôn' });
      expect(seenLine(reviewed, now)).toBe('Ôn lần cuối hôm nay');
      expect(mastery(reviewed)).toBe(3);
      expect(scoreText(reviewed)).toBe('3/5 câu đúng');
      expect(mastery(row('a', stats(0)))).toBeNull();
      expect(scoreText(row('a', null))).toBe('');
    });

    it('suggests the most words due, then the longest unseen', () => {
      expect(suggested([row('a', stats(1)), row('b', stats(4)), row('c', null)])?.id).toBe('b');
      expect(
        suggested([row('a', null, '2026-10-10T00:00:00Z'), row('b', null, '2026-10-02T00:00:00Z')])
          ?.id,
      ).toBe('b');
      expect(suggested([])).toBeNull();
    });

    it('leaves out a lesson reviewed today, and suggests none once all are', () => {
      const at = new Date(now - 60 * 1000).toISOString();
      const round = { correct: 4, total: 4, wrong: [], reviewedAt: at };
      const old = { ...round, reviewedAt: new Date(now - 3 * 24 * 3600 * 1000).toISOString() };
      const done = row('a', stats(1, { result: round }));
      expect(suggested([done, row('b', stats(0, { result: old }))], now)?.id).toBe('b');
      expect(suggested([done], now)).toBeNull();
    });
  });
});

describe('resultAfter', () => {
  const q = (id: string) => ({ id }) as Question;
  const answers = (...xs: [string, boolean][]): Answer[] =>
    xs.map(([id, correct]) => ({ question: q(id), correct, given: null }));

  it('scores a full round on its own', () => {
    expect(
      resultAfter('full', answers(['v:a', true], ['f:0', false]), {
        correct: 9,
        total: 9,
        wrong: [],
      }),
    ).toEqual({
      total: 2,
      correct: 1,
      wrong: ['f:0'],
    });
  });

  it('mends the previous score with a round of the wrong ones', () => {
    const previous = { correct: 2, total: 5, wrong: ['v:a', 'f:0', 't:1'] };
    expect(
      resultAfter('wrong', answers(['v:a', true], ['f:0', false], ['t:1', true]), previous),
    ).toEqual({
      total: 5,
      correct: 4,
      wrong: ['f:0'],
    });
  });
});
