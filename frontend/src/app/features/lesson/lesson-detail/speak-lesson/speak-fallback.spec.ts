import { fallbackQuestions, fallbackReadAloud, frameFrom, tipFor } from './speak-fallback';

const words = [
  { text: 'sing', lemma: 'sing', ipa: '/sɪŋ/' },
  { text: 'track', lemma: 'track', ipa: '/træk/' },
  { text: 'music', lemma: 'music', ipa: '/ˈmjuːzɪk/' },
  { text: 'the', lemma: 'the', ipa: '' },
];

describe('speak fallback', () => {
  it('gives a tip on the final -s, -ed and consonants', () => {
    expect(tipFor('sings', words[0])).toBe(
      'sings /sɪŋz/: nhớ đọc âm cuối /z/. Người Việt hay bỏ âm cuối -s.',
    );
    expect(tipFor('tracks', words[1])).toContain('/s/');
    expect(tipFor('tracked', words[1])).toContain('/t/');
    expect(tipFor('track', words[1])).toBe(
      'track /træk/: nhớ đọc rõ âm cuối /k/. Người Việt hay nuốt âm cuối.',
    );
    expect(tipFor('sing', words[0])).toBe('');
    expect(tipFor('the', words[3])).toBe('');
  });

  it('picks two short statements, those with a tip first', () => {
    const got = fallbackReadAloud(
      [
        'Do you like music?',
        'I like it a lot.',
        'She sings every day.',
        'I play this track at night.',
        'Hi.',
      ],
      words,
    );
    expect(got.map((r) => r.text)).toEqual(['She sings every day.', 'I play this track at night.']);
    expect(got.every((r) => r.tipVi !== '')).toBe(true);
  });

  it('fills up with sentences without a tip', () => {
    const got = fallbackReadAloud(['I like it a lot.', 'We are at home now.'], words);
    expect(got).toEqual([
      { text: 'I like it a lot.', tipVi: '' },
      { text: 'We are at home now.', tipVi: '' },
    ]);
  });

  it('makes a frame from a reply about oneself', () => {
    expect(frameFrom('Yes, I listen to music every night.', words)).toBe(
      'I listen to ___ every night.',
    );
    expect(frameFrom('I go to school.', words)).toBe('I go to ___.');
    expect(frameFrom('That is nice.', words)).toBe('');
  });

  it('asks the dialogue questions put to "you"', () => {
    const turns = [
      { speaker: 0, text: 'What music do you like?', meaningVi: 'Bạn thích nhạc gì?' },
      { speaker: 1, text: 'I like pop music.', meaningVi: '' },
      { speaker: 1, text: 'Is it loud?', meaningVi: '' },
      { speaker: 0, text: 'Hi! Where do you play it?', meaningVi: 'Chào! Bạn chơi ở đâu?' },
      { speaker: 1, text: 'I play it at home.', meaningVi: '' },
    ];
    expect(fallbackQuestions(turns, words)).toEqual([
      {
        question: 'What music do you like?',
        questionVi: 'Bạn thích nhạc gì?',
        frame: 'I like pop ___.',
      },
      { question: 'Where do you play it?', questionVi: '', frame: 'I play it at ___.' },
    ]);
  });
});
