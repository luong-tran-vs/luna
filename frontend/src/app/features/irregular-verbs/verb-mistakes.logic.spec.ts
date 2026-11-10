import { VerbProgress } from '../../core/models/verbs';
import { IRREGULAR_VERBS } from './irregular-verbs.data';
import { todayTasks, verbGroups } from './verb-groups.logic';
import { VERB_LESSONS } from './verb-lessons.data';
import {
  isStruggling,
  letterDiff,
  strugglingVerbs,
  typedResult,
  typoDiff,
} from './verb-mistakes.logic';
import { pairOf, VERB_PAIRS } from './verb-pairs.data';

const now = new Date('2026-11-10T10:00:00Z');
const daysAgo = (n: number) => new Date(now.getTime() - n * 24 * 3600 * 1000).toISOString();
const verb = (base: string) => IRREGULAR_VERBS.find((v) => v.base === base)!;
const progress = (base: string, over: Partial<VerbProgress> = {}): VerbProgress => ({
  base,
  learnedAt: null,
  level: 0,
  dueAt: null,
  right: 0,
  wrong: 0,
  ...over,
});

describe('Từ hay sai (added 2026-11-10)', () => {
  it('counts a verb typed wrong twice in 14 days until it is typed right twice in a row', () => {
    const twice = progress('begin', {
      mistakes: [
        { form: 'past', typed: 'begun', at: daysAgo(3) },
        { form: 'past', typed: 'begined', at: daysAgo(1) },
      ],
    });
    expect(isStruggling(twice, now)).toBe(true);
    expect(isStruggling({ ...twice, cleanRun: 1 }, now)).toBe(true);
    expect(isStruggling({ ...twice, cleanRun: 2 }, now)).toBe(false);
    // An old mistake no longer counts.
    const old = {
      ...twice,
      mistakes: [{ form: 'past' as const, typed: 'x', at: daysAgo(20) }, twice.mistakes![1]],
    };
    expect(isStruggling(old, now)).toBe(false);
  });

  it('lists the verbs most often typed wrong first, newest mistake first', () => {
    const m = (typed: string, d: number) => ({
      form: 'participle' as const,
      typed,
      at: daysAgo(d),
    });
    const list = strugglingVerbs(
      new Map([
        ['go', progress('go', { mistakes: [m('goed', 3), m('went', 1)] })],
        [
          'choose',
          progress('choose', { mistakes: [m('choosen', 4), m('chose', 3), m('choosed', 2)] }),
        ],
        ['see', progress('see', { mistakes: [m('saw', 1)] })],
      ]),
      now,
    );
    expect(list.map((s) => s.verb.base)).toEqual(['choose', 'go']);
    expect(list[0].mistakes[0].typed).toBe('choosed');
  });

  it('sends the forms typed wrong with what was typed', () => {
    const r = typedResult(
      { verb: verb('go'), past: 'went', participle: 'went', right: false },
      true,
    );
    expect(r).toEqual({
      base: 'go',
      right: false,
      wrong: [{ form: 'participle', typed: 'went' }],
      practice: true,
    });
    expect(
      typedResult({ verb: verb('go'), past: 'went', participle: 'gone', right: true }),
    ).toEqual({
      base: 'go',
      right: true,
    });
  });

  it('shows letter by letter where a typed word differs', () => {
    expect(letterDiff('choosen', 'chosen')).toEqual([
      { text: 'cho', kind: 'same' },
      { text: 'o', kind: 'extra' },
      { text: 'sen', kind: 'same' },
    ]);
    expect(letterDiff('bit', 'bitten')).toEqual([
      { text: 'bit', kind: 'same' },
      { text: 'ten', kind: 'missing' },
    ]);
  });

  it('pairs verbs often mixed up, each pair naming an irregular verb of the list', () => {
    expect(pairOf('lie')?.verbs.map((v) => v.base)).toEqual(['lie', 'lay']);
    expect(pairOf('go')).toBeNull();
    for (const p of VERB_PAIRS) {
      expect(p.verbs.some((v) => IRREGULAR_VERBS.some((x) => x.base === v.base))).toBe(true);
    }
  });
});

describe('Nhóm and Hôm nay (added 2026-11-10)', () => {
  it('puts every lesson in exactly one group, the like ones together', () => {
    const groups = verbGroups();
    expect(groups.flatMap((g) => g.lessons).length).toBe(VERB_LESSONS.length);
    const same = groups.find((g) => g.key === 'khong-doi')!;
    expect(same.lessons.map((l) => l.number)).toEqual([1, 2, 19, 20, 21]);
    expect(new Set(groups.map((g) => g.key)).size).toBe(groups.length);
  });

  it('lists only the tasks that apply, ticked once done', () => {
    expect(todayTasks(new Map(), now).map((t) => t.key)).toEqual(['lesson']);

    const learnedToday = progress('cut', {
      learnedAt: now.toISOString(),
      level: 1,
      dueAt: daysAgo(-1),
    });
    const due = progress('put', { learnedAt: daysAgo(5), level: 1, dueAt: daysAgo(1) });
    const struggling = progress('let', {
      mistakes: [
        { form: 'past', typed: 'letted', at: daysAgo(2) },
        { form: 'past', typed: 'lat', at: daysAgo(1) },
      ],
    });
    const tasks = todayTasks(
      new Map([
        ['cut', learnedToday],
        ['put', due],
        ['let', struggling],
      ]),
      now,
    );
    expect(tasks).toEqual([
      { key: 'review', label: 'Ôn 1 từ đến hạn', done: false },
      { key: 'lesson', label: 'Đã học một bài hôm nay', done: true },
      { key: 'mistakes', label: 'Luyện 1 từ hay sai', done: false },
    ]);
  });
});

describe('typoDiff (added 2026-11-10)', () => {
  it('shows the letters only for a near miss', () => {
    expect(typoDiff('choosen', 'chosen').length).toBeGreaterThan(0);
    expect(typoDiff('writen', 'written').length).toBeGreaterThan(0);
    expect(typoDiff('thinked', 'brought')).toEqual([]);
    expect(typoDiff('buyed', 'bought')).toEqual([]);
    expect(typoDiff('', 'bought')).toEqual([]);
  });
});
