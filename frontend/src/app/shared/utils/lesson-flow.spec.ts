import { LESSON_FLOW, resumeIndex } from './lesson-flow';

describe('resumeIndex', () => {
  it('is -1 before anything is done or saved', () => {
    expect(
      resumeIndex(LESSON_FLOW, null, { read: 'current', listen: 'locked', write: 'locked' }),
    ).toBe(-1);
  });

  it('goes on from the step saved on this device', () => {
    expect(
      resumeIndex(LESSON_FLOW, 'speak', { read: 'done', listen: 'done', write: 'current' }),
    ).toBe(3);
  });

  it('never goes back behind the step after the last one the server knows is done', () => {
    expect(
      resumeIndex(LESSON_FLOW, 'words', { read: 'done', listen: 'current', write: 'locked' }),
    ).toBe(2);
    expect(
      resumeIndex(LESSON_FLOW, null, { read: 'locked', listen: 'done', write: 'locked' }),
    ).toBe(3);
  });

  it('stays on the last step once every step is done', () => {
    expect(resumeIndex(LESSON_FLOW, null, { read: 'done', listen: 'done', write: 'done' })).toBe(4);
  });

  it('ignores a saved step the lesson does not have', () => {
    expect(resumeIndex(['read', 'listen', 'write'], 'speak', { read: 'done' })).toBe(1);
  });
});
