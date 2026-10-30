import { GrammarExercise } from '../../core/models/grammar-study';
import {
  correctAnswerText,
  gradeChoice,
  gradeFill,
  gradeFix,
  gradeReorder,
  missingToPass,
  patternParts,
  patternRows,
  autoMark,
  breakAnswer,
  breakAnswerText,
  guessStructure,
  percent,
  sentenceParts,
  shuffleWords,
  splitLines,
  splitBlank,
  structureTone,
  tabLabels,
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

  // Added 2026-10-20.
  it('grades a fix against the correction and the other accepted ones', () => {
    const e = ex({ kind: 'fix', text: 'She are a teacher.', sentence: 'She is a teacher.', answers: ["She's a teacher."] });
    expect(gradeFix(e, '  she is a  TEACHER ')).toBe(true);
    expect(gradeFix(e, 'She’s a teacher!')).toBe(true);
    expect(gradeFix(e, 'She are a teacher.')).toBe(false);
    expect(gradeFix(e, '')).toBe(false);
    expect(correctAnswerText(e)).toBe('She is a teacher.');
  });

  it('grades a reorder by the words in order', () => {
    const e = ex({ kind: 'reorder', sentence: 'She is a teacher.' });
    expect(gradeReorder(e, ['She', 'is', 'a', 'teacher.'])).toBe(true);
    expect(gradeReorder(e, ['she', 'is', 'a', 'Teacher.'])).toBe(true);
    expect(gradeReorder(e, ['is', 'She', 'a', 'teacher.'])).toBe(false);
    expect(gradeReorder(e, ['She', 'is', 'a'])).toBe(false);
    // Fixed 2026-10-20: the tiles often have no full stop.
    expect(gradeReorder(e, ['She', 'is', 'a', 'teacher'])).toBe(true);
    expect(gradeReorder(ex({ kind: 'reorder', sentence: 'Is she a teacher?' }), ['is', 'she', 'a', 'teacher'])).toBe(true);
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

  it('cuts patterns at "+" and colours the same piece alike in every structure', () => {
    const [pos, neg, question] = patternParts([
      'S + am/is/are + V-ing',
      'S + am/is/are + not + V-ing',
      'Am/Is/Are + S + V-ing?',
    ]);
    expect(pos).toEqual([
      { text: 'S', tone: 0 },
      { text: 'am/is/are', tone: 1 },
      { text: 'V-ing', tone: 2 },
    ]);
    expect(neg.map((p) => p.tone)).toEqual([0, 1, 3, 2]);
    expect(question).toEqual([
      { text: 'Am/Is/Are', tone: 1 },
      { text: 'S', tone: 0 },
      { text: 'V-ing?', tone: 2 },
    ]);
    expect(patternParts(['There is a book'])).toEqual([[{ text: 'There is a book', tone: 2 }]]);
  });

  it('colours a piece by its role: subjects alike, "be" and helping verbs alike', () => {
    const [rows] = patternRows(['I + am / he, she, it + is / you, we, they + are + ...']);
    expect(rows.map((r) => r.parts.map((p) => p.tone))).toEqual([[0, 1], [0, 1], [0, 1, 2]]);
    const [question] = patternRows(["Isn't + he, she, it ...?"]);
    expect(question[0].parts.map((p) => p.tone)).toEqual([1, 0]);
  });

  it('shortens structure labels for the tabs, unless two would be alike', () => {
    expect(
      tabLabels([
        'Thể khẳng định',
        'Câu phủ định',
        'Câu hỏi Yes/No và câu trả lời ngắn',
        'Câu hỏi có từ hỏi',
        'Dạng viết tắt',
        'Câu hỏi',
      ]),
    ).toEqual(['Khẳng định', 'Phủ định', 'Yes/No', 'Từ để hỏi', 'Viết tắt', 'Câu hỏi']);
    expect(tabLabels(['Đủ (tính từ)', 'Đủ (danh từ)', 'Ở phía sau (behind)'])).toEqual([
      'Đủ (tính từ)',
      'Đủ (danh từ)',
      'Ở phía sau',
    ]);
  });

  it('tells an affirmative, negative or question structure from its label', () => {
    expect(structureTone('Khẳng định')).toBe('positive');
    expect(structureTone('Phủ định')).toBe('negative');
    expect(structureTone('Nghi vấn (Yes/No)')).toBe('question');
    expect(structureTone('Câu hỏi Wh-')).toBe('question');
    expect(structureTone('Rút gọn')).toBe('other');
  });

  describe('sentenceParts', () => {
    const parts = [
      { text: 'S', tone: 0 },
      { text: 'am / is / are', tone: 1 },
      { text: 'V-ing', tone: 2 },
    ];

    it('colours the marked words like the pieces of the pattern', () => {
      expect(sentenceParts('She is reading a book.', '{1:She} {2:is} {3:reading} a book.', parts)).toEqual([
        { text: 'She', tone: 0 },
        { text: ' ', tone: null },
        { text: 'is', tone: 1 },
        { text: ' ', tone: null },
        { text: 'reading', tone: 2 },
        { text: ' a book.', tone: null },
      ]);
    });

    it('keeps the sentence plain without marks, with marks of another sentence, or past the pieces', () => {
      const plain = [{ text: 'She is here.', tone: null }];
      expect(sentenceParts('She is here.', undefined, parts)).toEqual(plain);
      expect(sentenceParts('She is here.', '{1:He} {2:is} here.', parts)).toEqual(plain);
      expect(sentenceParts('She is here.', 'She {9:is} here.', parts)[1]).toEqual({ text: 'is', tone: null });
    });
  });

  describe('line breaks', () => {
    it('puts the answer of a question on its own line, across coloured runs', () => {
      expect(breakAnswerText("Was Tom at the party? – No, he wasn't.")).toBe("Was Tom at the party?\nNo, he wasn't.");
      expect(breakAnswerText('Họ có mua xe không? - Không.')).toBe('Họ có mua xe không?\nKhông.');
      expect(
        breakAnswer([
          { text: 'Is it ', tone: null },
          { text: 'yours?', tone: 1 },
          { text: ' – Yes, it is.', tone: null },
        ]),
      ).toEqual([
        { text: 'Is it ', tone: null },
        { text: 'yours?', tone: 1 },
        { text: '\nYes, it is.', tone: null },
      ]);
    });

    it('leaves other dashes and sentences alone', () => {
      expect(breakAnswerText('I like it – a lot.')).toBe('I like it – a lot.');
    });
  });

  describe('patternRows', () => {
    it('puts each question of a pattern on its line, followed by the answers to it', () => {
      const [rows] = patternRows([
        "Am + I ...? / Is + he, she, it ...? / Are + you, we, they ...? -> Yes, I am. / No, I'm not. hoặc Yes, he is. / No, he isn't.",
      ]);
      expect(rows.map((r) => (r.answer ? `→ ${r.parts[0].text}` : r.parts.map((p) => p.text).join(' | ')))).toEqual([
        'Am | I ...?',
        '→ Yes, I am.',
        "→ No, I'm not.",
        'Is | he, she, it ...?',
        '→ Yes, he is.',
        "→ No, he isn't.",
        'Are | you, we, they ...?',
      ]);
    });

    it('puts forms that each have their "+" on their own lines', () => {
      const [rows] = patternRows(['I + am / he, she, it + is / you, we, they + are + ...']);
      expect(rows.map((r) => r.parts.map((p) => p.text).join(' | '))).toEqual([
        'I | am',
        'he, she, it | is',
        'you, we, they | are | ...',
      ]);
    });

    it('keeps short choices and "hoặc" inside a piece on one line', () => {
      const [rows] = patternRows(['S + am / is / are + danh từ hoặc tính từ']);
      expect(rows).toHaveLength(1);
      expect(rows[0].parts.map((p) => p.text)).toEqual(['S', 'am / is / are', 'danh từ hoặc tính từ']);
    });

    it('numbers the pieces for the marks line after line, without the answers', () => {
      expect(patternParts(['Am + I …? / Is + he …? -> Yes, I am.'])[0].map((p) => p.text)).toEqual(['Am', 'I …?', 'Is', 'he …?']);
    });
  });

  describe('guessStructure', () => {
    const labels = ['Thể khẳng định', 'Thể phủ định', 'Câu hỏi Yes/No và câu trả lời ngắn', 'Câu hỏi có từ để hỏi'];

    it('places an example under the structure its form shows', () => {
      expect(guessStructure('I am a new student.', labels)).toBe(1);
      expect(guessStructure('We are not tired today.', labels)).toBe(2);
      expect(guessStructure("She isn't here.", labels)).toBe(2);
      expect(guessStructure("Are you twenty? – No, I'm not.", labels)).toBe(3);
      expect(guessStructure('Where is the doctor?', labels)).toBe(4);
      expect(guessStructure('Where is it?', ['Khẳng định', 'Câu hỏi Yes/No', 'Câu hỏi có từ hỏi'])).toBe(3);
    });

    it('gives 0 when no label fits', () => {
      expect(guessStructure('I am happy.', ['Mạo từ a', 'Mạo từ the'])).toBe(0);
      expect(guessStructure('Where is it?', ['Khẳng định'])).toBe(0);
    });
  });

  describe('autoMark', () => {
    const tones = (runs: { text: string; tone: number | null }[]) =>
      runs.filter((r) => r.tone !== null).map((r) => `${r.text}:${r.tone}`);

    it('colours the words a pattern names, with the tone of their piece', () => {
      const parts = [
        { text: 'he, she, it', tone: 0 },
        { text: 'is', tone: 1 },
        { text: 'not', tone: 2 },
      ];
      expect(tones(autoMark('She is not tired. This is it.', parts))).toEqual(['She:0', 'is:1', 'not:2', 'is:1', 'it:0']);
    });

    it('takes the question words for "Wh-word" and leaves stand-ins such as S or V-ing alone', () => {
      const parts = [
        { text: 'Wh-word', tone: 0 },
        { text: 'am / is / are', tone: 1 },
        { text: 'S', tone: 2 },
        { text: 'V-ing', tone: 3 },
      ];
      expect(tones(autoMark('Where are you going?', parts))).toEqual(['Where:0', 'are:1']);
      expect(autoMark('Hi there.', [{ text: 'S', tone: 0 }])).toEqual([{ text: 'Hi there.', tone: null }]);
    });
  });

  it('splits coloured runs at their line breaks', () => {
    expect(
      splitLines([
        { text: 'Are', tone: 1 },
        { text: ' you here?\nNo, I', tone: null },
        { text: "'m", tone: 1 },
      ]),
    ).toEqual([
      [
        { text: 'Are', tone: 1 },
        { text: ' you here?', tone: null },
      ],
      [
        { text: 'No, I', tone: null },
        { text: "'m", tone: 1 },
      ],
    ]);
  });
});
