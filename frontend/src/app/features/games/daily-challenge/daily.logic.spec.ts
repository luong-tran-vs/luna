import { DailyQuestion, SpotQuestion } from '../../../core/models/games';
import { pauseClock, remaining, resumeClock, startClock, timeUp } from '../shared/game-clock.logic';
import {
  dailyAnswers,
  dayLabel,
  fixedSentence,
  hitsWrong,
  letterTiles,
  noErrorRight,
  QUESTION_MS,
  questionScore,
  recordAnswer,
  recordTimeout,
  resolveVerbs,
  rightTile,
  startDaily,
  weekCells,
} from './daily.logic';

const verb = (base = ''): DailyQuestion => ({ kind: 'verb', verb: { base } });
const pic: DailyQuestion = {
  kind: 'picture',
  picture: { lemma: 'apple', imageUrl: '', meaningVi: 'quả táo', options: ['apple', 'b', 'c', 'd'], answer: 0 },
};

describe('daily challenge', () => {
  it('gives each question 20 seconds; the time running out makes it wrong', () => {
    expect(QUESTION_MS).toBe(20_000);
    let c = startClock(QUESTION_MS, 0);
    c = pauseClock(c, 5_000);
    c = resumeClock(c, 60_000);
    expect(remaining(c, 60_000)).toBe(15_000);
    expect(timeUp(c, 75_000)).toBe(true);

    const s = recordTimeout(startDaily(), { word: 'apple' });
    expect(s.answers).toEqual([false]);
    expect(s.score).toBe(0);
    expect(s.wrongWords).toEqual(['apple']);
  });

  it('scores 100 + 10 per whole second left, at most 300, and nothing when wrong', () => {
    expect(questionScore(true, 20_000)).toBe(300);
    expect(questionScore(true, 12_900)).toBe(220);
    expect(questionScore(true, 0)).toBe(100);
    expect(questionScore(true, 99_000)).toBe(300);
    expect(questionScore(false, 15_000)).toBe(0);

    let s = startDaily();
    for (let i = 0; i < 12; i++) {
      s = recordAnswer(s, true, 20_000);
    }
    expect(s.answers.length).toBe(10);
    expect(s.score).toBe(3000);
  });

  it('keeps the missed words once and the verb forms typed wrong', () => {
    let s = startDaily();
    s = recordAnswer(s, false, 3_000, { word: 'apple' });
    s = recordAnswer(s, false, 3_000, { word: 'apple' });
    s = recordAnswer(s, false, 3_000, { verbs: [{ base: 'go', form: 'past', typed: 'goed' }] });
    s = recordAnswer(s, true, 3_000, { word: 'pear' });
    expect(s.wrongWords).toEqual(['apple']);
    expect(s.wrongVerbs).toEqual([{ base: 'go', form: 'past', typed: 'goed' }]);
    expect(dailyAnswers([pic, pic, verb('go'), pic], s.answers).map((a) => a.kind)).toEqual([
      'picture',
      'picture',
      'verb',
      'picture',
    ]);
  });

  it('picks a common verb from the seed for an empty base: same seed, same verb, none twice', () => {
    const qs = [verb(), pic, verb('eat'), verb(), verb()];
    const a = resolveVerbs(qs, 1873301);
    const b = resolveVerbs(qs, 1873301);
    expect(a.map((v) => v?.base)).toEqual(b.map((v) => v?.base));
    expect(a[1]).toBeNull();
    expect(a[2]?.base).toBe('eat');
    const bases = a.filter((v) => v !== null).map((v) => v.base);
    expect(new Set(bases).size).toBe(4);
    for (const v of a) {
      if (v) {
        expect(v.common || v.base === 'eat').toBe(true);
      }
    }
    // Another seed gives (almost surely) other verbs.
    const other = resolveVerbs(qs, 42).map((v) => v?.base);
    expect(other).not.toEqual(a.map((v) => v?.base));
  });

  it('shuffles the letters of "Ghép chữ" with 2–3 letters not in the word, the same for a seed', () => {
    const tiles = letterTiles('house', 7, 3);
    const extra = tiles.map((t) => t.ch).filter((c) => !'house'.includes(c));
    expect(tiles.length - 5).toBeGreaterThanOrEqual(2);
    expect(tiles.length - 5).toBeLessThanOrEqual(3);
    expect(extra.length).toBe(tiles.length - 5);
    expect([...tiles.map((t) => t.ch)].filter((c) => 'house'.includes(c)).sort()).toEqual(['e', 'h', 'o', 's', 'u']);
    expect(letterTiles('house', 7, 3)).toEqual(tiles);
    expect(rightTile('house', 0, { id: 0, ch: 'h' })).toBe(true);
    expect(rightTile('house', 1, { id: 0, ch: 'h' })).toBe(false);
  });

  it('grades the grammar sentence', () => {
    const q: SpotQuestion = {
      tokens: ['She', 'go', 'to', 'school', '.'],
      wrong: { from: 1, to: 2 },
      options: ['going', 'goes', 'gone'],
      answer: 1,
      ruleVi: 'Ngôi thứ ba số ít thêm -s.',
      topicVi: 'Hiện tại đơn',
      pointId: 'p',
      source: 'choice',
    };
    expect(hitsWrong(q, 1)).toBe(true);
    expect(hitsWrong(q, 0)).toBe(false);
    expect(noErrorRight(q)).toBe(false);
    expect(fixedSentence(q)).toBe('She goes to school.');
    expect(noErrorRight({ ...q, wrong: null, options: [] })).toBe(true);
  });

  it('labels the day and the week from Monday', () => {
    expect(dayLabel('2026-11-13')).toBe('Thứ Sáu, 13/11');
    const cells = weekCells([true, true, true, true, false, false, false], '2026-11-13');
    expect(cells.map((c) => c.label)).toEqual(['T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'CN']);
    expect(cells[4].today).toBe(true);
    expect(cells[4].played).toBe(false);
    expect(cells[0].played).toBe(true);
  });
});
