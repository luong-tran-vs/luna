import { AttemptBody, AttemptKind, GrammarExercise, MASTERY_PERCENT } from '../../core/models/grammar-study';
import { normalizeAnswer } from '../../shared/utils/answer-match';
import { shuffle } from '../lesson/lesson-detail/practice-logic';

/** Pure helpers of the grammar exercises (F20): grading in the browser, shuffling and scoring. */

/** A multiple-choice answer is right when the chosen option is the marked one. */
export function gradeChoice(ex: GrammarExercise, chosen: number): boolean {
  return ex.answerIndex !== undefined && chosen === ex.answerIndex;
}

/**
 * A typed answer is right when it equals one accepted answer. Both sides are normalised: curly
 * apostrophes become straight, runs of spaces become one, outer spaces go, case is ignored.
 */
export function gradeFill(ex: GrammarExercise, typed: string): boolean {
  const t = normalizeAnswer(typed);
  return t !== '' && (ex.answers ?? []).some((a) => normalizeAnswer(a) === t);
}

/** A sentence for comparing: normalised like a typed answer, without the closing full stop, "?" or "!". */
function foldSentence(text: string): string {
  return normalizeAnswer(text).replace(/\s*[.?!]+$/, '');
}

/**
 * A corrected sentence (added 2026-10-20) is right when it is the stored correction or another
 * accepted one; case, spacing, curly apostrophes and the closing punctuation are ignored.
 */
export function gradeFix(ex: GrammarExercise, typed: string): boolean {
  const t = foldSentence(typed);
  return t !== '' && [ex.sentence ?? '', ...(ex.answers ?? [])].some((s) => s !== '' && foldSentence(s) === t);
}

/** The right answer of an exercise as text (the first accepted answer of a fill). */
export function correctAnswerText(ex: GrammarExercise): string {
  switch (ex.kind) {
    case 'choice':
      return ex.options?.[ex.answerIndex ?? -1] ?? '';
    case 'fill':
      return ex.answers?.[0] ?? '';
    default:
      return ex.sentence ?? '';
  }
}

/** A word compared without case and the punctuation around it ("brothers." is the tile "brothers"). */
function wordKey(word: string): string {
  return normalizeAnswer(word).replace(/^[.,!?;:"“”]+|[.,!?;:"“”]+$/g, '');
}

/**
 * A built sentence is right when its words, in order, are the words of the sentence; punctuation
 * around a word is ignored (fixed 2026-10-20: the last tile often has no full stop).
 */
export function gradeReorder(ex: GrammarExercise, built: readonly string[]): boolean {
  const target = normalizeAnswer(ex.sentence ?? '').split(' ').map(wordKey);
  return built.length === target.length && built.every((w, i) => wordKey(w) === target[i]);
}

/** The word tiles of a reorder exercise in a random order; the same seed gives the same order. */
export function shuffleWords(words: readonly string[], seed: number): string[] {
  return shuffle(words, seed);
}

/** The text of a fill exercise around its blank: [before, after]; no blank gives [text, '']. */
export function splitBlank(text: string): [string, string] {
  const i = text.indexOf('___');
  return i < 0 ? [text, ''] : [text.slice(0, i), text.slice(i + 3)];
}

export interface Outcome {
  id: string;
  correct: boolean;
  /** What the learner answered, as text (for the review of a test). */
  given?: string;
}

/** The body of POST /api/grammar/{id}/attempts for a finished round. */
export function toAttempt(kind: AttemptKind, outcomes: readonly Outcome[]): AttemptBody {
  const wrong = outcomes.filter((o) => !o.correct).map((o) => o.id);
  return { kind, correct: outcomes.length - wrong.length, total: outcomes.length, wrong };
}

export function percent(correct: number, total: number): number {
  return total > 0 ? Math.round((correct * 100) / total) : 0;
}

/** How many more right answers a mastery test needed to reach the pass mark (0 when it passed). */
export function missingToPass(correct: number, total: number): number {
  const needed = Math.ceil((total * MASTERY_PERCENT) / 100);
  return Math.max(0, needed - correct);
}

/** One piece of a structure's pattern; `tone` (0–3) is the same for the same piece in every structure. */
export interface PatternPart {
  text: string;
  tone: number;
}

/** How many colours the pieces of the patterns cycle through. */
const PATTERN_TONES = 4;

/** One line of a pattern: pieces to colour, or (after "->") a sample answer shown as plain text. */
export interface PatternRow {
  parts: PatternPart[];
  answer: boolean;
}

/**
 * The arrow before the answers of a question pattern: "Am + I …? -> Yes, I am." After anything but a
 * question it is part of the pattern: "y → ies", "go → went".
 */
const PATTERN_ARROW = /\s*(?:->|→|=>)\s*/;
/** "/" or "hoặc" after a whole sentence starts another line: "Am + I …? / Is + he …?" */
const PATTERN_LINE = /([.?!])\s+(?:\/|hoặc)\s+/g;

/** How deep in brackets the end of `text` is. */
function bracketDepth(text: string): number {
  let depth = 0;
  for (const ch of text) {
    depth += '(['.includes(ch) ? 1 : ')]'.includes(ch) ? -1 : 0;
  }
  return depth;
}

/** `text` cut at `sep` (global) outside brackets: "động từ (+ tân ngữ) + trạng từ" cuts once at "+". */
function splitOutside(text: string, sep: RegExp): string[] {
  const out: string[] = [];
  let from = 0;
  for (const m of text.matchAll(sep)) {
    if (bracketDepth(text.slice(0, m.index)) === 0) {
      out.push(text.slice(from, m.index));
      from = m.index + m[0].length;
    }
  }
  out.push(text.slice(from));
  return out;
}

/** The pieces of one line of a pattern: cut at "+" outside brackets. */
function linePieces(line: string): string[] {
  return splitOutside(line, /\+/g)
    .map((s) => s.trim())
    .filter((s) => s !== '');
}

/**
 * The lines of a pattern. Besides a "/" after a whole sentence, a ";" between forms ("I + was; you +
 * were") starts a line, and so does a "/" between forms that each have their "+" and begin alike, with
 * a subject or with a helping verb ("I + am / he, she, it + is"; "Were + S + to … / Were it not for …").
 * Choices inside a piece stay: "S + am / is / are", "động từ + into / out of + nơi chốn",
 * "What + do / does + S".
 */
function patternLines(text: string): string[] {
  return text
    .replace(PATTERN_LINE, '$1\n')
    .split('\n')
    .flatMap((line) => {
      const parts = splitOutside(line, /\s*;\s+/g);
      return parts.length > 1 && parts.every((p) => /\p{L}/u.test(p)) ? parts : [line];
    })
    .flatMap((line) => {
      const forms = splitOutside(line, /\s+\/\s+/g).map(linePieces);
      if (forms.length < 2 || forms.some((f) => f.length < 2)) {
        return [line];
      }
      const role = pieceRole(forms[0][0]);
      const word = firstWord(forms[0][0]);
      const alike =
        (role !== '' && forms.every((f) => pieceRole(f[0]) === role)) ||
        forms.every((f) => firstWord(f[0]) === word);
      return alike ? forms.map((f) => f.join(' + ')) : [line];
    })
    .map((s) => s.trim())
    .filter((s) => s !== '');
}

/** Words that make a piece the subject: "S", "he, she, it", "chủ ngữ" (the server's subjectWords). */
const SUBJECT_WORDS = new Set('i you he she it we they subject sb this these those'.split(' '));
/** Words that make a piece the verb "be" or a helping verb: "am / is / are", "Does", "will not". */
const HELPER_WORDS = new Set(
  "am is are was were be been being do does did have has had will would can could shall should may might must isn't aren't wasn't weren't don't doesn't didn't won't can't couldn't wouldn't shouldn't mustn't needn't haven't hasn't hadn't".split(
    ' ',
  ),
);

/**
 * The words that tell what a piece is: without a note in brackets ("is not (isn't)" gives is, not),
 * a leading "Yes," or "No," and punctuation; "S" and "chủ ngữ" give subject.
 */
function pieceWords(text: string): string[] {
  const outside = text.replace(/\([^()]*\)/g, ' ');
  const words = (/\p{L}/u.test(outside) ? outside : text)
    .replace(/chủ ngữ/gi, 'subject')
    .replace(/’/g, "'")
    .split(/[\s,/()[\]?.!…]+/)
    .filter((w) => w !== '')
    .map((w) => (w === 'S' ? 'subject' : w.toLowerCase()));
  return words.length > 1 && (words[0] === 'yes' || words[0] === 'no') ? words.slice(1) : words;
}

/** The first word of a piece in lower case: "There is not" gives there. */
function firstWord(text: string): string {
  return text.trim().split(/\s+/)[0].toLowerCase();
}

/** What a piece is: the subject, "be" or a helping verb, or '' for anything else. */
function pieceRole(text: string): 'subject' | 'helper' | '' {
  const words = pieceWords(text);
  if (words.length > 0 && words.every((w) => SUBJECT_WORDS.has(w))) {
    return 'subject';
  }
  return words.length > 0 && HELPER_WORDS.has(words[0]) ? 'helper' : '';
}

/**
 * The tone of a piece by what it is, so the same role has the same colour in every line and structure:
 * the subject 0, "be" or a helping verb 1, anything else 2 or 3 (the same text keeps its tone).
 */
function pieceTone(text: string, others: Map<string, number>): number {
  const role = pieceRole(text);
  if (role !== '') {
    return role === 'subject' ? 0 : 1;
  }
  const key = pieceWords(text).join(' ');
  if (!others.has(key)) {
    others.set(key, 2 + (others.size % (PATTERN_TONES - 2)));
  }
  return others.get(key)!;
}

/**
 * The patterns of a point's structures in lines, each cut at "+": "S + am/is/are + V-ing" is one
 * line of three pieces; "Am + I …? / Is + he, she, it …? -> Yes, I am. / No, I'm not." is two lines
 * of pieces then two answer lines. Tones follow the role of a piece (see pieceTone), so "I" and
 * "he, she, it" are coloured alike, and so are "am/is/are" and "Am/Is/Are?".
 */
export function patternRows(patterns: readonly string[]): PatternRow[][] {
  const others = new Map<string, number>();
  const tone = (text: string) => pieceTone(text, others);
  return patterns.map((pattern) => {
    const [ask, answers] = splitAnswers(pattern);
    const rows: PatternRow[] = patternLines(ask)
      .map((line) => ({
        parts: linePieces(line).map((text) => ({ text, tone: tone(text) })),
        answer: false,
      }))
      .filter((r) => r.parts.length > 0);
    return withAnswers(rows, answers.flatMap(patternLines));
  });
}

/** A pattern and its sample answers: cut at the first arrow only when a question comes before it. */
function splitAnswers(pattern: string): [string, string[]] {
  const [ask, ...answers] = pattern.split(PATTERN_ARROW);
  return answers.length > 0 && /\?\s*$/.test(ask) ? [ask, answers] : [pattern, []];
}

/** The words of an English line, contractions written out: "I'm not" gives i, am, not. */
function englishWords(text: string): string[] {
  return text
    .toLowerCase()
    .replace(/'m\b/g, ' am')
    .replace(/'re\b/g, ' are')
    .replace(/'s\b/g, ' is')
    .replace(/n't\b/g, ' not')
    .split(/[^a-z]+/)
    .filter((w) => w !== '' && w !== 'yes' && w !== 'no' && w !== 'not');
}

/**
 * Puts each sample answer right after the question it answers, the one sharing most words with it
 * ("No, I'm not." after "Am + I …?"); an answer that matches none goes last.
 */
function withAnswers(questions: PatternRow[], answers: string[]): PatternRow[] {
  const after = questions.map(() => [] as PatternRow[]);
  const rest: PatternRow[] = [];
  const asked = questions.map((q) => new Set(englishWords(q.parts.map((p) => p.text).join(' '))));
  for (const line of answers) {
    const row: PatternRow = { parts: [{ text: line, tone: 0 }], answer: true };
    const words = englishWords(line);
    const scores = asked.map((set) => words.filter((w) => set.has(w)).length);
    const best = Math.max(0, ...scores);
    (best > 0 ? after[scores.indexOf(best)] : rest).push(row);
  }
  return [...questions.flatMap((q, i) => [q, ...after[i]]), ...rest];
}

/**
 * The pieces of each pattern, line after line and without the answers: the "{n:words}" marks of an
 * example count them from 1 (the server's grammar.PatternPieces cuts them alike).
 */
export function patternParts(patterns: readonly string[]): PatternPart[][] {
  return patternRows(patterns).map((rows) => rows.filter((r) => !r.answer).flatMap((r) => r.parts));
}

export type StructureTone = 'positive' | 'negative' | 'question' | 'other';

/** The colour of a structure's label: affirmative, negative, question, or anything else. */
export function structureTone(label: string): StructureTone {
  const l = label.toLowerCase();
  if (l.includes('phủ định')) {
    return 'negative';
  }
  if (l.includes('nghi vấn') || l.includes('câu hỏi')) {
    return 'question';
  }
  return l.includes('khẳng định') ? 'positive' : 'other';
}

/**
 * A structure's label short enough for a tab: "Câu hỏi Yes/No và câu trả lời ngắn" gives "Yes/No",
 * "Câu hỏi có từ hỏi" "Từ để hỏi", "Thể phủ định" "Phủ định", "Dạng viết tắt" "Viết tắt".
 */
function shortLabel(label: string): string {
  if (/\byes\s*\/\s*no\b/i.test(label)) {
    return 'Yes/No';
  }
  if (/từ (để )?hỏi|\bwh/i.test(label)) {
    return 'Từ để hỏi';
  }
  const head = label.split(/\s+và\s+|\s*[(:–—]\s*|\s+-\s+/)[0].trim();
  const short = head.replace(/^(?:thể|dạng|câu(?= (?:khẳng|phủ) định))\s+/i, '').trim() || head;
  return short.charAt(0).toUpperCase() + short.slice(1);
}

/** The tabs' labels, each shortened (see shortLabel) unless that makes two of them alike. */
export function tabLabels(labels: readonly string[]): string[] {
  const short = labels.map(shortLabel);
  return short.map((s, i) => (short.indexOf(s) === short.lastIndexOf(s) ? s : labels[i]));
}

/** A run of an example sentence; `tone` is the colour of its pattern piece, null for plain text. */
export interface SentencePart {
  text: string;
  tone: number | null;
}

/**
 * An example cut into runs from its marks: "{1:She} {2:is} reading" gives She (the tone of the first
 * piece of `parts`), " ", is (the second), " reading". Without marks, or with marks the sentence does
 * not match, the whole sentence is one plain run.
 */
export function sentenceParts(en: string, marked: string | undefined, parts: readonly PatternPart[]): SentencePart[] {
  if (!marked) {
    return [{ text: en, tone: null }];
  }
  const out: SentencePart[] = [];
  const mark = /\{(\d+):([^{}]+)\}/g;
  let at = 0;
  for (let m = mark.exec(marked); m; m = mark.exec(marked)) {
    if (m.index > at) {
      out.push({ text: marked.slice(at, m.index), tone: null });
    }
    out.push({ text: m[2], tone: parts[Number(m[1]) - 1]?.tone ?? null });
    at = m.index + m[0].length;
  }
  if (at < marked.length) {
    out.push({ text: marked.slice(at), tone: null });
  }
  return out.map((p) => p.text).join('') === en ? out : [{ text: en, tone: null }];
}

/** The dash between a question and its answer: "Was Tom there? – No, he wasn't." */
const ANSWER_DASH = /\?(\s+[–—-]\s+)/g;

/**
 * Puts the answer of a question on its own line: the dash after "?" becomes a line break, wherever
 * the runs of a coloured sentence cut it. Shown with `white-space: pre-line`.
 */
export function breakAnswer(parts: readonly SentencePart[]): SentencePart[] {
  const full = parts.map((p) => p.text).join('');
  const cuts: [number, number][] = [];
  for (const m of full.matchAll(ANSWER_DASH)) {
    cuts.push([m.index + 1, m.index + m[0].length]);
  }
  if (cuts.length === 0) {
    return [...parts];
  }
  let at = 0;
  return parts
    .map((p) => {
      let text = '';
      for (const ch of p.text) {
        const cut = cuts.find(([from, to]) => at >= from && at < to);
        if (!cut) {
          text += ch;
        } else if (at === cut[0]) {
          text += '\n';
        }
        at += ch.length;
      }
      return { ...p, text };
    })
    .filter((p) => p.text !== '');
}

/** breakAnswer for plain text, such as the Vietnamese meaning. */
export function breakAnswerText(text: string): string {
  return breakAnswer([{ text, tone: null }])
    .map((p) => p.text)
    .join('');
}

const WH_WORDS = new Set(['what', 'where', 'when', 'who', 'whom', 'whose', 'why', 'how', 'which']);

/**
 * For a lesson whose examples do not name their structure (written before 2026-10-30): the
 * 1-based structure an example most likely shows from the labels (khẳng định, phủ định, câu hỏi
 * Yes/No, câu hỏi có từ để hỏi), 0 when no label fits.
 */
export function guessStructure(en: string, labels: readonly string[]): number {
  const kinds = labels.map((label) => {
    const tone = structureTone(label);
    if (/từ (để )?hỏi|\bwh/i.test(label)) {
      return 'wh';
    }
    return tone === 'question' || /yes\s*\/\s*no/i.test(label) ? 'question' : tone;
  });
  const first = (kind: string) => kinds.findIndex((k) => k === kind) + 1;
  const question = en.split(/[–—]/)[0];
  if (question.includes('?')) {
    const wh = WH_WORDS.has(question.trim().split(/\s+/)[0].toLowerCase());
    return wh ? first('wh') || first('question') : first('question');
  }
  return /\b(not|never)\b|n't\b/i.test(en) ? first('negative') : first('positive');
}

/** Stand-ins for any word in a pattern, never coloured as they are: S, V-ing, Wh-word, sb, sth. */
const PLACEHOLDER = /^(?:[A-HJ-Z]|[A-Za-z]+-[a-z]+|sb|sth)$/;

/** The English words a piece of a pattern names, its choices apart: "he, she, it …?" gives he, she, it. */
function literalWords(piece: string): string[] {
  return piece
    .split(/[/,]/)
    .map((w) => w.replace(/[.?!…]+/g, '').trim())
    .filter((w) => /^[a-z']+(?: [a-z']+)*$/i.test(w) && !PLACEHOLDER.test(w));
}

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

/**
 * Colours an unmarked example from its structure's pattern: the words a piece names (am, is, he…)
 * take its tone; "Wh-word" takes the question words. Stand-ins such as S or V-ing are not guessed.
 */
export function autoMark(en: string, parts: readonly PatternPart[]): SentencePart[] {
  const toneOf = new Map<string, number>();
  for (const p of parts) {
    for (const w of literalWords(p.text)) {
      if (!toneOf.has(w.toLowerCase())) {
        toneOf.set(w.toLowerCase(), p.tone);
      }
    }
    if (/^wh/i.test(p.text)) {
      for (const w of WH_WORDS) {
        toneOf.set(w, p.tone);
      }
    }
  }
  if (toneOf.size === 0) {
    return [{ text: en, tone: null }];
  }
  const words = [...toneOf.keys()].sort((a, b) => b.length - a.length).map(escapeRegExp);
  const find = new RegExp(`\\b(?:${words.join('|')})\\b`, 'gi');
  const out: SentencePart[] = [];
  let at = 0;
  for (const m of en.matchAll(find)) {
    if (m.index > at) {
      out.push({ text: en.slice(at, m.index), tone: null });
    }
    out.push({ text: m[0], tone: toneOf.get(m[0].toLowerCase()) ?? null });
    at = m.index + m[0].length;
  }
  if (at < en.length) {
    out.push({ text: en.slice(at), tone: null });
  }
  return out;
}

/** Runs cut at their line breaks: a question and its answer as two lines. */
export function splitLines(parts: readonly SentencePart[]): SentencePart[][] {
  const lines: SentencePart[][] = [[]];
  for (const p of parts) {
    p.text.split('\n').forEach((text, i) => {
      if (i > 0) {
        lines.push([]);
      }
      if (text !== '') {
        lines[lines.length - 1].push({ ...p, text });
      }
    });
  }
  return lines.filter((l) => l.length > 0);
}
