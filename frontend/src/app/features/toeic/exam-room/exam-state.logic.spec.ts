import { PaperGroup, ToeicPart } from '../../../core/models/toeic';
import {
  byPart,
  cellState,
  changes,
  clear,
  ExamState,
  pagesOf,
  questionsOf,
  select,
  step,
  toggleFlag,
  unanswered,
} from './exam-state.logic';

function group(key: string, part: ToeicPart, n: number, start: number): PaperGroup {
  return {
    key,
    part,
    number: start,
    imageUrl: null,
    graphic: null,
    questions: Array.from({ length: n }, (_, i) => ({
      id: `${key}#${i + 1}`,
      no: i + 1,
      number: start + i,
      options: ['a', 'b', 'c', 'd'],
    })),
  };
}

const paper: PaperGroup[] = [
  ...Array.from({ length: 7 }, (_, i) => group(`p5-${i}`, 5, 1, 101 + i)),
  group('p6-a', 6, 4, 131),
  group('p7-a', 7, 3, 147),
];

describe('exam pages', () => {
  const pages = pagesOf(paper);
  const qs = questionsOf(pages, paper);

  it('puts five Part 5 questions on a page and a text with its questions on one', () => {
    expect(pages.map((p) => p.questionIds.length)).toEqual([5, 2, 4, 3]);
    expect(qs[6].page).toBe(1);
    expect(qs.at(-1)?.number).toBe(149);
  });

  it('groups the questions by part', () => {
    expect(byPart(qs).map((g) => [g.part, g.questions.length])).toEqual([
      [5, 7],
      [6, 4],
      [7, 3],
    ]);
  });

  it('steps from page to page and stops at the ends', () => {
    let s: ExamState = { answers: {}, flags: [], position: 'p5-0#1' };
    s = step(s, qs, pages, -1);
    expect(s.position).toBe('p5-0#1');
    s = step(s, qs, pages, 1);
    expect(s.position).toBe('p5-5#1');
    s = step(step(s, qs, pages, 1), qs, pages, 1);
    expect(s.position).toBe('p7-a#1');
    expect(step(s, qs, pages, 1).position).toBe('p7-a#1');
  });
});

describe('exam state', () => {
  const base: ExamState = { answers: { a: 1 }, flags: [], position: 'a' };

  it('selects, clears, flags and tells the cells apart', () => {
    let s = select(base, 'b', 2);
    expect(s.answers).toEqual({ a: 1, b: 2 });
    expect(s.position).toBe('b');
    s = toggleFlag(s, 'a');
    expect(cellState(s, 'a')).toBe('flagged');
    expect(cellState(s, 'b')).toBe('current');
    expect(cellState(s, 'c')).toBe('empty');
    s = toggleFlag(clear(s, 'a'), 'a');
    expect(s.flags).toEqual([]);
    expect(s.answers).toEqual({ b: 2 });
  });

  it('lists the unanswered questions', () => {
    const qs = [
      { id: 'a', number: 1, part: 5 as const, page: 0 },
      { id: 'b', number: 2, part: 5 as const, page: 0 },
    ];
    expect(unanswered(base, qs).map((q) => q.id)).toEqual(['b']);
  });

  it('sends only what changed, a cleared answer as -1', () => {
    expect(changes(base, base)).toEqual({});
    const now: ExamState = { answers: { b: 0 }, flags: ['b'], position: 'b' };
    expect(changes(base, now)).toEqual({ answers: { b: 0, a: -1 }, flags: ['b'], position: 'b' });
    expect(changes(base, select(base, 'a', 1))).toEqual({});
  });
});
