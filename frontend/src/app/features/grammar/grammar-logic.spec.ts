import { GrammarExercise } from '../../core/models/grammar-study';
import {
  correctAnswerText,
  gradeChoice,
  gradeFill,
  gradeReorder,
  missingToPass,
  percent,
  shuffleWords,
  splitBlank,
  toAttempt,
} from './grammar-logic';

const ex = (over: Partial<GrammarExercise>): GrammarExercise => ({
  id: 'p1',
  kind: 'choice',
  explanationVi: 'vì sao',
  ...over,
});

describe('grammar logic', () => {
  it('grades a choice by its index', () => {
    const e = ex({ options: ['am', 'is', 'are', 'be'], answerIndex: 1 });
    expect(gradeChoice(e, 1)).toBe(true);
    expect(gradeChoice(e, 0)).toBe(false);
    expect(gradeChoice(ex({}), 0)).toBe(false);
  });

  it('grades a fill ignoring case and outer spaces, with several accepted answers', () => {
    const e = ex({ kind: 'fill', answers: ['is', "isn't"] });
    expect(gradeFill(e, '  IS ')).toBe(true);
    expect(gradeFill(e, 'isn’t')).toBe(true);
    expect(gradeFill(e, 'are')).toBe(false);
    expect(gradeFill(e, '   ')).toBe(false);
  });

  it('grades a fill with curly apostrophes, repeated spaces and case on both sides', () => {
    const e = ex({ kind: 'fill', answers: ['is not', "isn't", 'It’s'] });
    expect(gradeFill(e, 'isn’t')).toBe(true);
    expect(gradeFill(e, 'ISN‘T')).toBe(true);
    expect(gradeFill(e, 'is  not')).toBe(true);
    expect(gradeFill(e, '  Is \t NOT ')).toBe(true);
    expect(gradeFill(e, "it's")).toBe(true);
    expect(gradeFill(e, 'isnt')).toBe(false);
  });

  it('gives the right answer of each kind as text', () => {
    expect(correctAnswerText(ex({ options: ['a', 'b'], answerIndex: 1 }))).toBe('b');
    expect(correctAnswerText(ex({ options: ['a'] }))).toBe('');
    expect(correctAnswerText(ex({ kind: 'fill', answers: ['x', 'y'] }))).toBe('x');
    expect(correctAnswerText(ex({ kind: 'reorder', sentence: 'She is' }))).toBe('She is');
  });

  it('grades a reorder by the words in order', () => {
    const e = ex({ kind: 'reorder', sentence: 'She is a teacher.' });
    expect(gradeReorder(e, ['She', 'is', 'a', 'teacher.'])).toBe(true);
    expect(gradeReorder(e, ['she', 'is', 'a', 'Teacher.'])).toBe(true);
    expect(gradeReorder(e, ['is', 'She', 'a', 'teacher.'])).toBe(false);
    expect(gradeReorder(e, ['She', 'is', 'a'])).toBe(false);
  });

  it('shuffles words without losing any, the same way for the same seed', () => {
    const words = ['a', 'b', 'c', 'd', 'e', 'f'];
    const out = shuffleWords(words, 7);
    expect([...out].sort()).toEqual(words);
    expect(shuffleWords(words, 7)).toEqual(out);
    expect(words).toEqual(['a', 'b', 'c', 'd', 'e', 'f']);
  });

  it('splits the text around the blank', () => {
    expect(splitBlank('She ___ a teacher.')).toEqual(['She ', ' a teacher.']);
    expect(splitBlank('No blank')).toEqual(['No blank', '']);
  });

  it('builds the attempt body from the outcomes', () => {
    const body = toAttempt('practice', [
      { id: 'p1', correct: true },
      { id: 'p2', correct: false },
      { id: 'p3', correct: true },
    ]);
    expect(body).toEqual({ kind: 'practice', correct: 2, total: 3, wrong: ['p2'] });
  });

  it('computes the percent and what a mastery test still missed', () => {
    expect(percent(4, 5)).toBe(80);
    expect(percent(0, 0)).toBe(0);
    expect(missingToPass(3, 5)).toBe(1);
    expect(missingToPass(4, 5)).toBe(0);
    expect(missingToPass(5, 10)).toBe(3);
    expect(missingToPass(9, 10)).toBe(0);
  });
});
