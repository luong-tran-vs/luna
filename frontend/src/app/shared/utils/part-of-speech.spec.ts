import { posLabel } from './part-of-speech';

describe('posLabel', () => {
  it('names the parts of speech in Vietnamese', () => {
    expect(posLabel('noun')).toBe('danh từ');
    expect(posLabel('phrasal verb')).toBe('cụm động từ');
    expect(posLabel('phrase')).toBe('cụm từ');
  });

  it('is empty when unknown', () => {
    expect(posLabel('')).toBe('');
    expect(posLabel(undefined)).toBe('');
    expect(posLabel('gerund')).toBe('');
  });
});
