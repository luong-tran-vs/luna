import { GroupAdmin, TestAdmin, TestCheck, ToeicPart } from '../../../core/models/toeic';
import {
  atLevel,
  bankSummary,
  byLevel,
  bySource,
  byTest,
  NO_TEST,
  reasonsOf,
  testCard,
} from './toeic-admin.logic';

function group(key: string, status: GroupAdmin['status'], tests: string[]): GroupAdmin {
  return {
    key,
    part: 5,
    level: 600,
    questions: 1,
    tags: [],
    status,
    source: 'claude',
    version: 1,
    needsPhoto: false,
    hasPhoto: false,
    pictures: 0,
    pictureSlots: 0,
    tests,
    title: key,
  };
}

const NEED = [6, 25, 39, 30, 30, 16, 54];

function check(have: number[], extra: Partial<TestCheck> = {}): TestCheck {
  return {
    parts: have.map((h, i) => ({ part: (i + 1) as ToeicPart, have: h, need: NEED[i] })),
    missingGroups: [],
    draftGroups: [],
    missingPhotos: [],
    graphics: 0,
    docs: 0,
    listeningMinutes: 0,
    complete: false,
    ready: false,
    ...extra,
  };
}

function test(status: TestAdmin['status'], c: TestCheck): TestAdmin {
  return {
    key: 'luna-test-01',
    title: 'Luna Test 01',
    kind: 'full',
    order: 1,
    groups: ['a', 'b', 'c'],
    status,
    updatedAt: '',
    check: c,
  };
}

describe('byTest', () => {
  const list = [
    group('a', 'published', ['t1']),
    group('b', 'draft', []),
    group('c', 'published', ['t1', 't2']),
  ];

  it('keeps every group without a filter', () => {
    expect(byTest(list, '').map((g) => g.key)).toEqual(['a', 'b', 'c']);
  });

  it('keeps the groups in no test', () => {
    expect(byTest(list, NO_TEST).map((g) => g.key)).toEqual(['b']);
  });

  it('keeps the groups of one test', () => {
    expect(byTest(list, 't2').map((g) => g.key)).toEqual(['c']);
  });
});

describe('bankSummary', () => {
  it('counts the groups by status and those in no test', () => {
    const list = [
      group('a', 'published', ['t1']),
      group('b', 'draft', []),
      group('c', 'hidden', []),
      group('d', 'published', ['t1']),
    ];
    expect(bankSummary(list)).toEqual({ total: 4, published: 2, draft: 1, hidden: 1, free: 2 });
  });
});

describe('testCard', () => {
  it('sums the questions and marks each part', () => {
    const card = testCard(test('draft', check([0, 0, 0, 0, 30, 8, 54])));
    expect(card.have).toBe(92);
    expect(card.need).toBe(200);
    expect(card.percent).toBe(46);
    expect(card.groups).toBe(3);
    expect(card.parts.map((p) => p.state)).toEqual([
      'none',
      'none',
      'none',
      'none',
      'ok',
      'some',
      'ok',
    ]);
    expect(card.parts[5].label).toBe('Part 6: 8/16 câu');
    expect(card.reasons).toEqual(['Part 6: có 8/16 câu']);
  });

  it('gives no reasons once published', () => {
    const card = testCard(test('published', check([0, 0, 0, 0, 30, 16, 54], { ready: true })));
    expect(card.reasons).toEqual([]);
    expect(card.percent).toBe(50);
  });
});

describe('reasonsOf', () => {
  it('lists draft groups and missing Part 1 photos', () => {
    const c = check([6, 0, 0, 0, 30, 16, 54], {
      draftGroups: ['x', 'y'],
      missingPhotos: [{ groupKey: 'p1', number: 3 }],
    });
    expect(reasonsOf(c)).toEqual(['2 cụm chưa đăng', 'Thiếu ảnh Part 1 ở câu 3']);
  });
});

describe('bySource', () => {
  const ai = { ...group('a', 'draft', []), source: 'ai' };
  const file = group('b', 'published', ['t1']);
  const old = { ...group('c', 'published', []), source: '' };

  it('keeps every group, the AI ones or the imported ones', () => {
    expect(bySource([ai, file, old], '').map((g) => g.key)).toEqual(['a', 'b', 'c']);
    expect(bySource([ai, file, old], 'ai').map((g) => g.key)).toEqual(['a']);
    expect(bySource([ai, file, old], 'file').map((g) => g.key)).toEqual(['b', 'c']);
  });
});

describe('byLevel', () => {
  const at = (key: string, level: number): GroupAdmin => ({ ...group(key, 'draft', []), level });
  const list = [at('a', 750), at('b', 450), at('c', 900), at('d', 450)];

  it('keeps the server order, or sorts by level keeping the order of equal levels', () => {
    expect(byLevel(list, '').map((g) => g.key)).toEqual(['a', 'b', 'c', 'd']);
    expect(byLevel(list, 'asc').map((g) => g.key)).toEqual(['b', 'd', 'a', 'c']);
    expect(byLevel(list, 'desc').map((g) => g.key)).toEqual(['c', 'a', 'b', 'd']);
    expect(list.map((g) => g.key)).toEqual(['a', 'b', 'c', 'd']);
  });
});

describe('atLevel', () => {
  const at = (key: string, level: number): GroupAdmin => ({ ...group(key, 'draft', []), level });
  const list = [at('a', 750), at('b', 450), at('c', 450)];

  it('keeps the groups of one level, or all of them', () => {
    expect(atLevel(list, 450).map((g) => g.key)).toEqual(['b', 'c']);
    expect(atLevel(list, 900)).toEqual([]);
    expect(atLevel(list, null).map((g) => g.key)).toEqual(['a', 'b', 'c']);
  });
});
