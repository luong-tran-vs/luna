import { Stats } from '../../core/models/dashboard';
import { badges } from './badges';

const stats = (over: Partial<Stats> = {}): Stats => ({
  cards: 0,
  dictation: { sentences: 0, correctWords: 0, totalWords: 0, rate: null },
  lessons: { read: 0, listen: 0, write: 0, completed: 0 },
  reading: { answered: 0, correct: 0, rate: null },
  writing: { submitted: 0, averageScore: null },
  ...over,
});

describe('badges', () => {
  it('earns nothing for a new learner, with how far each one is', () => {
    const list = badges(stats(), 0);
    expect(list.map((b) => b.id)).toEqual(['streak', 'words', 'listen', 'read', 'write', 'lessons']);
    expect(list.every((b) => !b.earned && b.value === 0)).toBe(true);
    expect(list[0].hint).toBe('Học 7 ngày liên tiếp');
  });

  it('earns a badge at its goal and caps the progress there', () => {
    const list = badges(stats({ cards: 250, lessons: { read: 9, listen: 10, write: 0, completed: 0 } }), 7);
    const byId = new Map(list.map((b) => [b.id, b]));
    expect(byId.get('streak')!.earned).toBe(true);
    expect(byId.get('words')).toMatchObject({ earned: true, value: 100, goal: 100 });
    expect(byId.get('listen')!.earned).toBe(true);
    expect(byId.get('read')).toMatchObject({ earned: false, value: 9 });
  });
});
