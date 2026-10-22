import { containsAny, containsWord, pickWords, segments, wordForms } from './checklist';

describe('writing checklist', () => {
  it('knows the simple inflections of a word', () => {
    expect(wordForms('dance')).toEqual(expect.arrayContaining(['dances', 'danced', 'dancing']));
    expect(wordForms('study')).toEqual(expect.arrayContaining(['studies', 'studied', 'studying']));
    expect(wordForms('stop')).toEqual(expect.arrayContaining(['stopped', 'stopping']));
    expect(wordForms('watch')).toEqual(expect.arrayContaining(['watches', 'watched']));
  });

  it('matches a lesson word as a whole word, any case, with its inflections', () => {
    expect(containsWord('The Sound is nice.', 'sound')).toBe(true);
    expect(containsWord('I like these sounds', 'sound')).toBe(true);
    expect(containsWord('We were dancing all night', 'dance')).toBe(true);
    expect(containsWord('It sounded loud', 'sound')).toBe(true);
    expect(containsWord('Soundproof walls', 'sound')).toBe(false);
    expect(containsWord('This is it', 'is')).toBe(true);
    expect(containsWord('This one', 'is')).toBe(false);
    expect(containsWord('anything', '')).toBe(false);
  });

  it('matches a phrase, inflecting its first word', () => {
    expect(containsWord('He turned  on the radio', 'turn on')).toBe(true);
    expect(containsWord('Turn it on', 'turn on')).toBe(false);
  });

  it('checks the grammar words exactly', () => {
    expect(containsAny('My mother likes songs', ['he', 'she', 'likes'])).toBe(true);
    expect(containsAny('She’s happy', ['she'])).toBe(true);
    expect(containsAny('The shelf is old', ['she', 'he'])).toBe(false);
    expect(containsAny('He DOESN’T like it', ["doesn't"])).toBe(true);
    expect(containsAny('anything', [])).toBe(false);
  });

  it('picks up to 3 distinct lesson words, base forms and single words first', () => {
    expect(pickWords(['songs', 'song', 'turn on', 'Loud', 'loud', 'sound', 'dance'])).toEqual(['song', 'Loud', 'sound']);
    expect(pickWords(['turn on', 'track'])).toEqual(['track', 'turn on']);
    expect(pickWords([])).toEqual([]);
  });

  it('cuts a text into lesson words, grammar words and the rest', () => {
    expect(segments('He likes the loud sounds.', ['sound', 'loud'], ['likes'])).toEqual([
      { text: 'He ', kind: 'plain' },
      { text: 'likes', kind: 'grammar' },
      { text: ' the ', kind: 'plain' },
      { text: 'loud', kind: 'word' },
      { text: ' ', kind: 'plain' },
      { text: 'sounds', kind: 'word' },
      { text: '.', kind: 'plain' },
    ]);
    expect(segments('Nothing here', ['sound'])).toEqual([{ text: 'Nothing here', kind: 'plain' }]);
    expect(segments('Plain', [])).toEqual([{ text: 'Plain', kind: 'plain' }]);
    expect(segments('', ['x'])).toEqual([]);
  });
});
