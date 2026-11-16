import { SpotQuestion } from '../../../core/models/games';
import {
  callClean,
  chooseFix,
  correctedSentence,
  currentSpot,
  joinTokens,
  mistakeText,
  nextSpot,
  SPOT_MAX_SCORE,
  spotFinished,
  spotPoints,
  startSpot,
  tapToken,
  timeOut,
} from './grammar-spot.logic';

/** "She go to school every day." with "go" wrong (goes / going / went). */
const withMistake = (): SpotQuestion => ({
  tokens: ['She', 'go', 'to', 'school', 'every', 'day', '.'],
  wrong: { from: 1, to: 2 },
  options: ['going', 'goes', 'went'],
  answer: 1,
  ruleVi: 'Chủ ngữ số ít: động từ thêm -s.',
  topicVi: 'Hiện tại đơn',
  pointId: 'a1-present-simple',
  source: 'fix',
});

/** "They are playing football now." */
const clean = (): SpotQuestion => ({
  tokens: ['They', 'are', 'playing', 'football', 'now', '.'],
  wrong: null,
  options: [],
  answer: 0,
  ruleVi: 'Now: hiện tại tiếp diễn.',
  topicVi: 'Hiện tại tiếp diễn',
  pointId: 'a1-present-continuous',
  source: 'fix',
});

/** "Hoa says to they." with a two-word span to test any token of the span. */
const longSpan = (): SpotQuestion => ({
  tokens: ['Yes', ',', 'she', 'aren', 'not', 'here', '.'],
  wrong: { from: 2, to: 5 },
  options: ['she is not', 'she are not', 'she am not'],
  answer: 0,
  ruleVi: 'She đi với is.',
  topicVi: 'Động từ to be',
  pointId: 'a1-to-be',
  source: 'choice',
});

describe('grammar spot', () => {
  it('opens the fixes when any token of the mistake is tapped', () => {
    const s = startSpot([withMistake()]);
    const after = tapToken(s, 1, 15_000);
    expect(after.stage).toBe('fix');
    expect(after.tapped).toBe(1);
    expect(after.lives).toBe(3);
    for (const i of [2, 3, 4]) {
      expect(tapToken(startSpot([longSpan()]), i, 15_000).stage).toBe('fix');
    }
  });

  it('scores the right fix with the time left and keeps the lives', () => {
    let s = tapToken(startSpot([withMistake(), clean()]), 1, 12_000);
    s = chooseFix(s, 1, 12_000);
    expect(s.stage).toBe('shown');
    expect(s.outcome).toBe('fixed');
    expect(s.gained).toBe(spotPoints(12_000));
    expect(s.gained).toBe(220);
    expect(s.score).toBe(220);
    expect(s.lives).toBe(3);
    expect(s.right).toBe(1);
  });

  it('loses a life on a wrong fix', () => {
    let s = tapToken(startSpot([withMistake()]), 1, 12_000);
    s = chooseFix(s, 2, 12_000);
    expect(s.outcome).toBe('bad-fix');
    expect(s.lives).toBe(2);
    expect(s.gained).toBe(0);
    expect(s.missed.length).toBe(1);
  });

  it('loses a life and reveals the mistake when a right word is tapped', () => {
    const s = tapToken(startSpot([withMistake()]), 3, 12_000);
    expect(s.stage).toBe('shown');
    expect(s.outcome).toBe('bad-token');
    expect(s.lives).toBe(2);
    expect(s.wrong).toBe(1);
    expect(mistakeText(withMistake())).toBe('go');
  });

  it('loses a life on "Không có lỗi" when the sentence has a mistake', () => {
    const s = callClean(startSpot([withMistake()]), 12_000);
    expect(s.outcome).toBe('missed');
    expect(s.lives).toBe(2);
    expect(s.score).toBe(0);
  });

  it('counts "Không có lỗi" on a right sentence as right', () => {
    const s = callClean(startSpot([clean()]), 5_000);
    expect(s.outcome).toBe('clean');
    expect(s.lives).toBe(3);
    expect(s.score).toBe(150);
  });

  it('counts a tap in a right sentence as wrong', () => {
    const s = tapToken(startSpot([clean()]), 1, 5_000);
    expect(s.outcome).toBe('false-alarm');
    expect(s.lives).toBe(2);
  });

  it('counts a sentence whose time ran out as wrong, also with the fixes open', () => {
    let s = timeOut(startSpot([clean()]));
    expect(s.outcome).toBe('timeout');
    expect(s.lives).toBe(2);
    s = timeOut(tapToken(startSpot([withMistake()]), 1, 1000));
    expect(s.outcome).toBe('timeout');
    expect(s.lives).toBe(2);
    // An answered sentence cannot time out any more.
    const done = callClean(startSpot([clean()]), 1000);
    expect(timeOut(done)).toBe(done);
  });

  it('ignores a second answer', () => {
    const s = callClean(startSpot([clean()]), 1000);
    expect(callClean(s, 1000)).toBe(s);
    expect(tapToken(s, 0, 1000)).toBe(s);
    expect(chooseFix(s, 0, 1000)).toBe(s);
  });

  it('ends after the last sentence or the third lost life', () => {
    let s = startSpot([withMistake(), withMistake(), withMistake(), clean()]);
    for (let i = 0; i < 3; i++) {
      s = nextSpot(callClean(s, 1000));
    }
    expect(s.lives).toBe(0);
    expect(spotFinished(s)).toBe(true);
    expect(currentSpot(s)).toBeNull();

    let t = startSpot([clean()]);
    t = nextSpot(callClean(t, 1000));
    expect(spotFinished(t)).toBe(true);
  });

  it('never scores above the server maximum', () => {
    const many = Array.from({ length: 30 }, () => clean());
    let s = startSpot(many);
    while (!spotFinished(s)) {
      s = nextSpot(callClean(s, 20_000));
    }
    expect(s.score).toBe(SPOT_MAX_SCORE);
  });

  it('joins tokens without spaces before marks and gives the corrected sentence', () => {
    expect(joinTokens(['Hello', ',', '"', 'Tom', '"', '(', 'he', ')', 'said', '.'])).toBe(
      'Hello, "Tom" (he) said.',
    );
    expect(correctedSentence(withMistake())).toBe('She goes to school every day.');
    expect(correctedSentence(longSpan())).toBe('Yes, she is not here.');
    expect(correctedSentence(clean())).toBe('They are playing football now.');
  });
});
