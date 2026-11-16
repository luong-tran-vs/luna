import { PictureQuestion } from '../../../core/models/games';
import { answerPicture, currentPicture, nextPicture, pictureFinished, startPicture } from './picture-quiz.logic';

const q = (lemma: string): PictureQuestion => ({
  lemma,
  imageUrl: `/api/words/${lemma}/image`,
  meaningVi: 'nghĩa ' + lemma,
  options: [lemma, 'x', 'y', 'z'],
  answer: 0,
});

describe('picture quiz', () => {
  it('raises the combo up to x5 and scores 100 × combo', () => {
    let s = startPicture([q('a'), q('b'), q('c'), q('d'), q('e'), q('f')]);
    const gains: number[] = [];
    for (let i = 0; i < 6; i++) {
      s = answerPicture(s, 0);
      gains.push(s.gained);
      s = nextPicture(s);
    }
    expect(gains).toEqual([200, 300, 400, 500, 500, 500]);
    expect(s.score).toBe(2400);
    expect(s.right).toBe(6);
    expect(pictureFinished(s)).toBe(true);
    expect(currentPicture(s)).toBeNull();
  });

  it('drops the combo on a mistake and keeps the word once', () => {
    let s = startPicture([q('bicycle'), q('car'), q('bicycle')]);
    s = nextPicture(answerPicture(s, 0)); // combo x2
    s = answerPicture(s, 2);
    expect(s.combo).toBe(1);
    expect(s.gained).toBe(0);
    // A second tap on the same question changes nothing.
    expect(answerPicture(s, 0)).toBe(s);
    s = nextPicture(s);
    s = answerPicture(s, 1);
    expect(s.missed.map((m) => m.lemma)).toEqual(['car', 'bicycle']);
    expect(s.wrong).toBe(2);
  });

  it('keeps a word missed twice once', () => {
    let s = startPicture([q('a'), q('a')]);
    s = nextPicture(answerPicture(s, 1));
    s = answerPicture(s, 1);
    expect(s.missed.length).toBe(1);
  });
});
