import { pickVoice } from './speech.service';

const v = (name: string, lang: string, localService = true) => ({ name, lang, localService });

describe('pickVoice', () => {
  const voices = [
    v('Microsoft David', 'en-US'),
    v('Microsoft Libby Online (Natural)', 'en-GB', false),
    v('Microsoft George', 'en-GB'),
    v('Google français', 'fr-FR'),
    v('Microsoft Zira', 'en-US'),
  ];

  it('picks a voice of the language, the best first', () => {
    expect(pickVoice(voices, 'en-GB', 0)?.name).toBe('Microsoft Libby Online (Natural)');
    expect(pickVoice(voices, 'en-GB', 1)?.name).toBe('Microsoft George');
  });

  it('gives different voices to different indexes when there are two or more', () => {
    expect(pickVoice(voices, 'en-US', 0)?.name).not.toBe(pickVoice(voices, 'en-US', 1)?.name);
  });

  it('wraps round and keeps to the device voices when asked', () => {
    expect(pickVoice(voices, 'en-GB', 2)?.name).toBe('Microsoft Libby Online (Natural)');
    expect(pickVoice(voices, 'en-GB', 0, true)?.name).toBe('Microsoft George');
  });

  it('falls back to the English voices when the language has none', () => {
    expect(pickVoice(voices, 'en-AU', 0)?.lang.startsWith('en')).toBe(true);
    expect(pickVoice([v('Google français', 'fr-FR')], 'en-AU', 0)).toBeNull();
  });
});
