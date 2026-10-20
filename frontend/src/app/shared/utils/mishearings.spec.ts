import { compareDictation } from './dictation-compare';
import { mishearings } from './mishearings';

describe('mishearings', () => {
  it('names the kinds of slips, most frequent first, with the learner own examples', () => {
    const results = [
      compareDictation('I want to go home.', 'i want go home'), // "to" missed
      compareDictation('She likes the music.', 'she like music'), // "-s" and "the"
      compareDictation('We went there a lot.', 'we want there alot'), // vowel, joined
      compareDictation('It is beautiful.', 'it is beautifull'), // spelling
    ];
    const found = mishearings(results, 5);
    expect(found.map((m) => [m.kind, m.count])).toEqual([
      ['weak', 2],
      ['ending', 1],
      ['vowel', 1],
      ['linking', 1],
      ['spelling', 1],
    ]);
    expect(found[0].examples).toEqual(['to (bỏ sót)', 'the (bỏ sót)']);
    expect(found[1].examples).toEqual(['like → likes']);
    expect(found[2].examples).toEqual(['want → went']);
    expect(found[3].examples).toEqual(['alot → a lot']);
  });

  it('keeps the three most frequent kinds by default and says nothing of unclear slips', () => {
    expect(mishearings([compareDictation('I like apples.', 'i like oranges')])).toEqual([]);
    expect(mishearings([compareDictation('I like apples.', 'I like apples')])).toEqual([]);
  });

  it('takes a small word heard as another small word for a weak form, not an ending', () => {
    const found = mishearings([compareDictation('An apple and an orange.', 'an apple an an orange')]);
    expect(found.map((m) => m.kind)).toEqual(['weak']);
  });
});
