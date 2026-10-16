import { CHOICE_COUNT, makeChoices, seedOf, shuffled } from './choices';

describe('choices', () => {
  it('puts the answer among up to three others, none the same', () => {
    const c = makeChoices('đi', ['ăn', 'Đi', 'ngủ', 'ăn ', 'chạy', 'đọc'], 7);
    expect(c.options).toHaveLength(CHOICE_COUNT);
    expect(c.options[c.answer]).toBe('đi');
    expect(new Set(c.options.map((o) => o.toLowerCase().trim())).size).toBe(CHOICE_COUNT);
  });

  it('takes the preferred options first', () => {
    const c = makeChoices('đi', ['a', 'b', 'c', 'd', 'e'], 3, ['chạy', 'bơi', 'nhảy']);
    expect([...c.options].sort()).toEqual(['bơi', 'chạy', 'nhảy', 'đi'].sort());
  });

  it('gives fewer options with a small pool', () => {
    expect(makeChoices('đi', ['ăn'], 1).options).toHaveLength(2);
    expect(makeChoices('đi', [], 1)).toEqual({ options: ['đi'], answer: 0 });
  });

  it('shuffles the same way for the same seed', () => {
    expect(makeChoices('đi', ['a', 'b', 'c', 'd'], 5)).toEqual(
      makeChoices('đi', ['a', 'b', 'c', 'd'], 5),
    );
    expect(shuffled([1, 2, 3, 4], 9)).toEqual(shuffled([1, 2, 3, 4], 9));
    expect(seedOf('went')).toBe(seedOf('went'));
    expect(seedOf('went')).not.toBe(seedOf('gave up'));
  });
});
