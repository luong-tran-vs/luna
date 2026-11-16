import { GameWord } from '../../../core/models/games';
import {
  dealCards,
  elapsedLabel,
  flipCard,
  hideMismatch,
  isUp,
  MemoryCard,
  MemoryState,
  memoryFinished,
  memoryScore,
  pairsFound,
  startMemory,
} from './memory.logic';

const word = (lemma: string, imageUrl = `/api/words/${lemma}/image`): GameWord => ({
  lemma,
  imageUrl,
  meaningVi: 'nghĩa ' + lemma,
});

/** The cards in a known order: pair k is at 2k (word) and 2k + 1 (picture or meaning). */
function ordered(pairs: GameWord[]): MemoryCard[] {
  return dealCards(pairs, () => 0.999999).sort((a, b) => a.pair - b.pair || (a.face === 'word' ? -1 : 1));
}

const six = ['apple', 'car', 'dog', 'egg', 'fish', 'hat'].map((l) => word(l));

describe('memory match', () => {
  it('deals a word card and a picture or meaning card for each pair', () => {
    const cards = dealCards([word('apple'), word('car', '')]);
    expect(cards.length).toBe(4);
    const apple = cards.filter((c) => c.pair === 0).map((c) => c.face).sort();
    expect(apple).toEqual(['image', 'word']);
    const car = cards.find((c) => c.pair === 1 && c.face !== 'word')!;
    expect(car.face).toBe('meaning');
    expect(car.text).toBe('nghĩa car');
  });

  it('keeps two matching cards and counts the move', () => {
    let s = startMemory(ordered(six));
    s = flipCard(s, 0);
    expect(s.moves).toBe(0);
    s = flipCard(s, 1);
    expect(s.moves).toBe(1);
    expect(s.matched[0] && s.matched[1]).toBe(true);
    expect(s.flipped).toEqual([]);
    expect(s.justMatched).toBe(0);
    expect(pairsFound(s)).toBe(1);
    // A matched card cannot be flipped again.
    expect(flipCard(s, 0)).toBe(s);
  });

  it('turns a mismatch back down', () => {
    let s = startMemory(ordered(six));
    s = flipCard(flipCard(s, 0), 2);
    expect(s.mismatch).toBe(true);
    expect(isUp(s, 0) && isUp(s, 2)).toBe(true);
    s = hideMismatch(s);
    expect(s.flipped).toEqual([]);
    expect(isUp(s, 0)).toBe(false);
    expect(s.moves).toBe(1);
  });

  it('a third card turns the mismatch back at once and flips the new one', () => {
    let s = startMemory(ordered(six));
    s = flipCard(flipCard(s, 0), 2);
    s = flipCard(s, 5);
    expect(s.mismatch).toBe(false);
    expect(s.flipped).toEqual([5]);
    expect(isUp(s, 0)).toBe(false);
    expect(s.moves).toBe(1);
    // Tapping a card already up changes nothing.
    expect(flipCard(s, 5)).toBe(s);
  });

  it('a tap on one of the mismatched cards starts a new move with it', () => {
    let s = startMemory(ordered(six));
    s = flipCard(flipCard(s, 0), 2);
    s = flipCard(s, 0);
    expect(s.mismatch).toBe(false);
    expect(s.flipped).toEqual([0]);
    s = flipCard(s, 1);
    expect(pairsFound(s)).toBe(1);
    expect(s.moves).toBe(2);
  });

  it('ends once the six pairs are matched', () => {
    let s: MemoryState = startMemory(ordered(six));
    for (let k = 0; k < 6; k++) {
      expect(memoryFinished(s)).toBe(false);
      s = flipCard(flipCard(s, 2 * k), 2 * k + 1);
    }
    expect(memoryFinished(s)).toBe(true);
    expect(pairsFound(s)).toBe(6);
    expect(s.moves).toBe(6);
    expect(flipCard(s, 0)).toBe(s);
  });

  it('scores lower with more moves and more time', () => {
    expect(memoryScore(6, 0)).toBe(10_000);
    expect(memoryScore(6, 12_900)).toBe(10_000 - 20 * 12);
    expect(memoryScore(10, 12_000)).toBe(10_000 - 300 * 4 - 20 * 12);
    expect(memoryScore(10, 30_000)).toBeLessThan(memoryScore(8, 30_000));
    expect(memoryScore(10, 40_000)).toBeLessThan(memoryScore(10, 30_000));
    expect(memoryScore(60, 600_000)).toBe(0);
  });

  it('labels the elapsed time', () => {
    expect(elapsedLabel(12_400)).toBe('0:12');
    expect(elapsedLabel(65_000)).toBe('1:05');
  });
});
