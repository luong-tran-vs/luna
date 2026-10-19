import { PracticeFill, PracticeTranslation } from '../../../core/models/practice';
import { CompletedLesson } from '../../../core/models/study';
import { DueCard, LessonResultInput, LessonStats, VocabItem } from '../../../core/models/vocab';
import { Choices, makeChoices, seedOf, shuffled } from '../../../shared/utils/choices';

/**
 * Pure helpers of the lesson review (design/tat-ca-de-xuat.png (mục on-tap-theo-bai)): the questions of one round
 * (words, fill-in, translation) built from the lesson, and the list of lessons of the review tab.
 */

export type Part = 'vocab' | 'fill' | 'translate';

export const PARTS: readonly Part[] = ['vocab', 'fill', 'translate'];

/** At most this many questions of each part in one round. */
export const PART_MAX: Record<Part, number> = { vocab: 8, fill: 4, translate: 3 };

export const PART_NAMES: Record<Part, string> = {
  vocab: 'Từ vựng',
  fill: 'Điền từ',
  translate: 'Dịch câu',
};

/** About how long one question takes, for the estimate of the round. */
const SECONDS_PER_QUESTION = 25;

/** "Từ này nghĩa là gì?": the word, its meaning among others. */
export interface VocabQuestion {
  kind: 'vocab';
  id: string;
  word: string;
  lemma: string;
  ipa: string;
  meaningVi: string;
  sentence: string;
  sentenceVi: string;
  choices: Choices;
  /** The saved card when it is due: the answer rates it. Null for other words. */
  card: DueCard | null;
}

/** A line of the lesson's dialogue with one word missing, the word among others. */
export interface FillQuestion {
  kind: 'fill';
  id: string;
  before: string;
  after: string;
  answer: string;
  vi: string;
  choices: Choices;
}

/** A Vietnamese sentence to build in English from tiles. */
export interface TranslateQuestion {
  kind: 'translate';
  id: string;
  vi: string;
  answer: string[];
  /** Answer tiles and distractors, shuffled. */
  tiles: string[];
}

export type Question = VocabQuestion | FillQuestion | TranslateQuestion;

/** What the learner did with a question: right or not, and what they gave (null: skipped/forgot). */
export interface Answer {
  question: Question;
  correct: boolean;
  given: string | null;
}

/** Every question the lesson can give in each part, before trimming to PART_MAX. */
export type Bank = Record<Part, Question[]>;

/**
 * The questions of one round. Words: the lesson's due cards first, then its other words; the wrong
 * meanings come from the lesson's other words, the same part of speech first. Fill-in: one question
 * per blank of the practice dialogue, the other blanks filled in. Translation: the practice's
 * sentences. Each part shuffled with `seed`, so another round asks other questions first.
 */
export function buildBank(
  items: readonly VocabItem[],
  due: readonly DueCard[],
  fill: PracticeFill | null,
  translations: readonly PracticeTranslation[],
  seed: number,
): Bank {
  return {
    vocab: vocabQuestions(items, due, seed),
    fill: fillQuestions(fill, items, seed),
    translate: translateQuestions(translations, seed),
  };
}

function vocabQuestions(
  items: readonly VocabItem[],
  due: readonly DueCard[],
  seed: number,
): VocabQuestion[] {
  const meanings = [...items.map((i) => i.meaningVi), ...due.map((c) => c.meaningVi)].filter((m) =>
    m.trim(),
  );
  const posOf = new Map(items.map((i) => [i.lemma.toLowerCase(), i.pos ?? '']));
  const samePos = (lemma: string) => {
    const pos = posOf.get(lemma.toLowerCase());
    return pos ? items.filter((i) => i.pos === pos).map((i) => i.meaningVi) : [];
  };
  const seen = new Set<string>();
  const out: VocabQuestion[] = [];
  for (const c of shuffled(due, seed)) {
    const lemma = c.lemma.toLowerCase();
    if (seen.has(lemma) || !c.meaningVi.trim()) {
      continue;
    }
    seen.add(lemma);
    out.push({
      kind: 'vocab',
      id: `v:${lemma}`,
      word: c.text,
      lemma,
      ipa: c.ipa,
      meaningVi: c.meaningVi,
      sentence: c.contextSentence,
      sentenceVi: c.contextSentenceVi ?? '',
      choices: makeChoices(c.meaningVi, meanings, seed ^ seedOf(lemma), samePos(lemma)),
      card: c,
    });
  }
  for (const i of shuffled(items, seed)) {
    const lemma = i.lemma.toLowerCase();
    if (seen.has(lemma) || !i.meaningVi.trim()) {
      continue;
    }
    seen.add(lemma);
    out.push({
      kind: 'vocab',
      id: `v:${lemma}`,
      word: i.text,
      lemma,
      ipa: i.ipa,
      meaningVi: i.meaningVi,
      sentence: i.sentence,
      sentenceVi: '',
      choices: makeChoices(i.meaningVi, meanings, seed ^ seedOf(lemma), samePos(lemma)),
      card: null,
    });
  }
  return out;
}

function fillQuestions(
  fill: PracticeFill | null,
  items: readonly VocabItem[],
  seed: number,
): FillQuestion[] {
  if (!fill) {
    return [];
  }
  const answers = fill.blanks.map((b) => b.answer);
  const pool = [...fill.wordBank, ...answers, ...items.map((i) => i.text)];
  const out: FillQuestion[] = [];
  for (const turn of fill.turns) {
    turn.parts.forEach((part, at) => {
      if (!('blank' in part)) {
        return;
      }
      const answer = answers[part.blank] ?? '';
      if (!answer.trim()) {
        return;
      }
      const text = (from: number, to: number) =>
        turn.parts
          .slice(from, to)
          .map((p) => ('text' in p ? p.text : (answers[p.blank] ?? '')))
          .join('');
      out.push({
        kind: 'fill',
        id: `f:${part.blank}`,
        before: text(0, at),
        after: text(at + 1, turn.parts.length),
        answer,
        vi: turn.meaningVi,
        choices: makeChoices(answer, pool, seed ^ seedOf(answer, part.blank), fill.wordBank),
      });
    });
  }
  return shuffled(out, seed);
}

function translateQuestions(
  translations: readonly PracticeTranslation[],
  seed: number,
): TranslateQuestion[] {
  return shuffled(
    translations
      .filter((t) => t.answer.length > 0)
      .map((t, i) => ({
        kind: 'translate' as const,
        id: `t:${i}`,
        vi: t.vi,
        answer: t.answer,
        tiles: shuffled(t.tiles, seed + i),
      })),
    seed,
  );
}

/**
 * The questions of a round: up to PART_MAX of each part kept, in the order of PARTS. The words due
 * (a card of the lesson) are always asked, even past PART_MAX or with Từ vựng left out: otherwise
 * the lesson would still say "Cần ôn" after a round got all right.
 */
export function roundOf(bank: Bank, parts: readonly Part[]): Question[] {
  return PARTS.flatMap((p) => {
    if (p === 'vocab') {
      const due = dueWords(bank);
      return parts.includes(p)
        ? bank.vocab.slice(0, Math.max(PART_MAX.vocab, due))
        : bank.vocab.slice(0, due);
    }
    return parts.includes(p) ? bank[p].slice(0, PART_MAX[p]) : [];
  });
}

/** How many words of the bank are due (they come first in its Từ vựng). */
export function dueWords(bank: Bank): number {
  return bank.vocab.filter((q) => q.kind === 'vocab' && q.card).length;
}

/** How many questions a part has in a round. */
export function partCount(bank: Bank, part: Part): number {
  const max = part === 'vocab' ? Math.max(PART_MAX.vocab, dueWords(bank)) : PART_MAX[part];
  return Math.min(bank[part].length, max);
}

export function minutesFor(questions: number): number {
  return Math.max(1, Math.round((questions * SECONDS_PER_QUESTION) / 60));
}

export function partOf(q: Question): Part {
  return q.kind;
}

/** A tile answer is right when the tiles read the answer, word for word. */
export function checkTiles(chosen: readonly string[], answer: readonly string[]): boolean {
  return chosen.length === answer.length && chosen.every((t, i) => t === answer[i]);
}

export interface PartScore {
  part: Part;
  correct: number;
  total: number;
}

/** Right answers per part, for the parts the round had. */
export function scores(answers: readonly Answer[]): PartScore[] {
  return PARTS.map((part) => {
    const of = answers.filter((a) => partOf(a.question) === part);
    return { part, correct: of.filter((a) => a.correct).length, total: of.length };
  }).filter((s) => s.total > 0);
}

/**
 * Words to bring back for review: the words got wrong (not the due cards, which are rated) and the
 * words of the blanks got wrong. The server keeps only the lesson's words.
 */
export function missedWords(answers: readonly Answer[]): string[] {
  const words = new Set<string>();
  for (const a of answers) {
    if (a.correct) {
      continue;
    }
    if (a.question.kind === 'vocab' && !a.question.card) {
      words.add(a.question.lemma);
    } else if (a.question.kind === 'fill') {
      words.add(a.question.answer);
    }
  }
  return [...words].slice(0, 30);
}

/** One row of the review tab's list of lessons. */
export interface LessonRow {
  id: string;
  title: string;
  topicName: string;
  /** Empty for the lesson being studied. */
  completedAt: string;
  stats: LessonStats | null;
}

export type BadgeTone = 'due' | 'done' | 'later' | 'new';

export interface Badge {
  tone: BadgeTone;
  text: string;
}

const DAY = 24 * 60 * 60 * 1000;

/** The lessons learned, with the review state of their saved words. */
export function lessonRows(
  completed: readonly CompletedLesson[],
  stats: readonly LessonStats[],
): LessonRow[] {
  const byId = new Map(stats.map((s) => [s.lessonId, s]));
  return completed.map((l) => ({
    id: l.id,
    title: l.title,
    topicName: l.topicName,
    completedAt: l.completedAt,
    stats: byId.get(l.id) ?? null,
  }));
}

/** When the lesson was last looked at: its last review, else when it was finished. */
function lastSeen(row: LessonRow): number {
  return Date.parse(lastReviewOf(row) ?? row.completedAt) || 0;
}

/** The latest review of the lesson: of one of its cards or a lesson review round; null for none. */
export function lastReviewOf(row: LessonRow): string | null {
  const times = [row.stats?.lastReview, row.stats?.result?.reviewedAt].filter(
    (t): t is string => !!t,
  );
  return times.sort((a, b) => Date.parse(b) - Date.parse(a))[0] ?? null;
}

/** The lessons by how much they need a review: the most words due, then the longest unseen. */
export function byNeed<T extends LessonRow>(rows: readonly T[]): T[] {
  return [...rows].sort(
    (a, b) => (b.stats?.due ?? 0) - (a.stats?.due ?? 0) || lastSeen(a) - lastSeen(b),
  );
}

/**
 * The lesson to review today: the first by need. A lesson already reviewed (a lesson review
 * round) today is left out; null once every lesson is.
 */
export function suggested<T extends LessonRow>(rows: readonly T[], now = Date.now()): T | null {
  const today = new Date(now).toDateString();
  const open = rows.filter((r) => {
    const at = r.stats?.result?.reviewedAt;
    return !at || new Date(at).toDateString() !== today;
  });
  return byNeed(open)[0] ?? null;
}

/** The label of a row: words due, reviewed lately, when next, or never reviewed. */
export function badgeOf(row: LessonRow, now: number): Badge {
  const s = row.stats;
  if (s && s.due > 0) {
    return { tone: 'due', text: `Cần ôn · ${s.due}` };
  }
  const last = lastReviewOf(row);
  if (last && now - Date.parse(last) < DAY) {
    return { tone: 'done', text: 'Đã ôn' };
  }
  if (s?.nextDue) {
    return {
      tone: 'later',
      text: `Còn ${Math.max(1, Math.ceil((Date.parse(s.nextDue) - now) / DAY))} ngày`,
    };
  }
  return { tone: 'new', text: 'Chưa ôn' };
}

/** Out of 5: the score of the latest lesson review round; null when it was never reviewed so. */
export function mastery(row: LessonRow): number | null {
  const r = row.stats?.result;
  return r && r.total > 0 ? Math.round((5 * r.correct) / r.total) : null;
}

/** "12/15 câu đúng": the score of the latest lesson review round; empty when none. */
export function scoreText(row: LessonRow): string {
  const r = row.stats?.result;
  return r ? `${r.correct}/${r.total} câu đúng` : '';
}

/** "hôm nay", "hôm qua", "3 ngày trước", by local calendar day. */
export function daysAgo(iso: string, now: number): string {
  const day = (t: number) => {
    const d = new Date(t);
    return Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()) / DAY;
  };
  const n = day(now) - day(Date.parse(iso));
  return n <= 0 ? 'hôm nay' : n === 1 ? 'hôm qua' : `${n} ngày trước`;
}

/** The small line of a row: when its words were last reviewed, else when it was finished. */
export function seenLine(row: LessonRow, now: number): string {
  const last = lastReviewOf(row);
  if (last) {
    return `Ôn lần cuối ${daysAgo(last, now)}`;
  }
  return row.completedAt ? `Học xong ${daysAgo(row.completedAt, now)}` : 'Đang học';
}

/** A round of every part chosen, or one of the questions got wrong last time. */
export type RoundKind = 'full' | 'wrong';

/**
 * The lesson's score after a round. A full round is its own score. A round of the questions got
 * wrong only mends the previous score: a question answered right now leaves its wrong list.
 */
export function resultAfter(
  kind: RoundKind,
  answers: readonly Answer[],
  previous: LessonResultInput | null,
): LessonResultInput {
  if (kind === 'wrong' && previous) {
    const mended = new Set(answers.filter((a) => a.correct).map((a) => a.question.id));
    const wrong = previous.wrong.filter((id) => !mended.has(id));
    return { total: previous.total, correct: previous.total - wrong.length, wrong };
  }
  return {
    total: answers.length,
    correct: answers.filter((a) => a.correct).length,
    wrong: answers.filter((a) => !a.correct).map((a) => a.question.id),
  };
}

/** The questions of the bank with these ids, in the order of the bank's parts. */
export function questionsWithIds(bank: Bank, ids: readonly string[]): Question[] {
  const wanted = new Set(ids);
  return PARTS.flatMap((p) => bank[p].filter((q) => wanted.has(q.id)));
}
