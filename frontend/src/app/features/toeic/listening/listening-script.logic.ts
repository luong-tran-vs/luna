import { Accent, PaperGroup, ToeicPart } from '../../../core/models/toeic';

/** The language tag of a browser voice for each accent of the content. */
export type VoiceLang = 'en-US' | 'en-GB' | 'en-AU' | 'en-CA';

export function accentLang(a: Accent | string | undefined): VoiceLang {
  switch (a) {
    case 'gb':
      return 'en-GB';
    case 'au':
      return 'en-AU';
    case 'ca':
      return 'en-CA';
    default:
      return 'en-US';
  }
}

export type SegmentKind = 'directions' | 'lead' | 'turn' | 'option' | 'question';

/** One thing read aloud, then a pause, in the order of the test. */
export interface Segment {
  kind: SegmentKind;
  part: ToeicPart;
  /** The group it belongs to; null for the directions of a part. */
  group: string | null;
  text: string;
  lang: VoiceLang;
  /** Which of the voices of that language: the people of a conversation sound different. */
  voice: number;
  /** Silence after it, in milliseconds (not changed by the speed). */
  pauseAfterMs: number;
  /** The speaker of a turn ("m", "w", "m2"...). */
  speaker?: string;
}

/** Pauses of the test (research R3): the answer time after Part 1/2 items and Part 3/4 questions. */
export const PAUSE = {
  afterDirections: 1500,
  afterLead: 700,
  betweenOptions: 600,
  afterItem12: 5000,
  betweenTurns: 350,
  afterTalk: 1200,
  afterQuestion34: 8000,
};

/**
 * The directions of each listening part, written for Luna (not the official wording), with a
 * Vietnamese summary shown under them.
 */
export const DIRECTIONS: Record<1 | 2 | 3 | 4, { en: string; vi: string }> = {
  1: {
    en:
      'Part 1. Directions: For each question in this part, you will hear four statements about a picture in your test book. ' +
      'When you hear the statements, select the one statement that best describes what you see in the picture. ' +
      'The statements will not be printed in your test book and will be spoken only one time.',
    vi: 'Nghe 4 câu mô tả bức ảnh, chọn câu mô tả đúng nhất. Các câu không in ra và chỉ đọc một lần.',
  },
  2: {
    en:
      'Part 2. Directions: You will hear a question or statement and three responses spoken in English. ' +
      'They will not be printed in your test book and will be spoken only one time. ' +
      'Select the best response to the question or statement.',
    vi: 'Nghe một câu hỏi hoặc câu nói và 3 câu đáp, chọn câu đáp phù hợp nhất. Chỉ đọc một lần.',
  },
  3: {
    en:
      'Part 3. Directions: You will hear some conversations between two or more people. ' +
      'You will be asked to answer three questions about what the speakers say in each conversation. ' +
      'Select the best response to each question. The conversations will not be printed and will be spoken only one time.',
    vi: 'Nghe các đoạn hội thoại, mỗi đoạn trả lời 3 câu hỏi in trong đề. Hội thoại chỉ đọc một lần.',
  },
  4: {
    en:
      'Part 4. Directions: You will hear some talks given by a single speaker. ' +
      'You will be asked to answer three questions about what the speaker says in each talk. ' +
      'Select the best response to each question. The talks will not be printed and will be spoken only one time.',
    vi: 'Nghe các bài nói của một người, mỗi bài trả lời 3 câu hỏi in trong đề. Bài nói chỉ đọc một lần.',
  },
};

/** The narrator's voice: the one reading numbers, questions and the directions. */
const NARRATOR: { lang: VoiceLang; voice: number } = { lang: 'en-US', voice: 0 };

const LETTERS = ['A', 'B', 'C', 'D'];

/**
 * The script of the listening parts of a paper, in order: the directions of each part (unless
 * skipped), then each group (Part 1: number, the four statements; Part 2: number, the question, the
 * three responses; Part 3/4: the lead, the turns, each question), with the pauses of the test.
 */
export function buildScript(paper: readonly PaperGroup[], opts: { directions: boolean } = { directions: true }): Segment[] {
  const out: Segment[] = [];
  let lastPart: ToeicPart | null = null;
  for (const g of paper) {
    if (g.part > 4) {
      continue;
    }
    if (g.part !== lastPart) {
      lastPart = g.part;
      if (opts.directions) {
        const d = DIRECTIONS[g.part as 1 | 2 | 3 | 4];
        out.push({ kind: 'directions', part: g.part, group: null, text: d.en, ...NARRATOR, pauseAfterMs: PAUSE.afterDirections });
      }
    }
    out.push(...groupScript(g));
  }
  return out;
}

function groupScript(g: PaperGroup): Segment[] {
  const out: Segment[] = [];
  const base = { part: g.part, group: g.key };
  const first = g.questions[0];
  const last = g.questions[g.questions.length - 1];
  if (!first) {
    return out;
  }
  const options = (q: (typeof g.questions)[number], lang: VoiceLang, voice: number, after: number): void => {
    q.options.forEach((o, i) => {
      out.push({
        ...base,
        kind: 'option',
        text: `(${LETTERS[i]}) ${o}`,
        lang,
        voice,
        pauseAfterMs: i === q.options.length - 1 ? after : PAUSE.betweenOptions,
      });
    });
  };
  switch (g.part) {
    case 1:
      out.push({
        ...base,
        kind: 'lead',
        text: `Number ${first.number}. Look at the picture marked number ${first.number} in your test book.`,
        ...NARRATOR,
        pauseAfterMs: PAUSE.afterLead,
      });
      options(first, 'en-US', 1, PAUSE.afterItem12);
      break;
    case 2: {
      const turn = g.script?.[0];
      const askLang = accentLang(turn?.accent);
      out.push({ ...base, kind: 'lead', text: `Number ${first.number}.`, ...NARRATOR, pauseAfterMs: PAUSE.afterLead });
      out.push({
        ...base,
        kind: 'turn',
        text: turn?.text ?? first.prompt ?? '',
        lang: askLang,
        voice: 0,
        speaker: turn?.speaker,
        pauseAfterMs: PAUSE.afterLead,
      });
      // The responses come from someone else.
      const replyLang: VoiceLang = askLang === 'en-US' ? 'en-GB' : 'en-US';
      options(first, replyLang, 1, PAUSE.afterItem12);
      break;
    }
    default: {
      const speakers: string[] = [];
      for (const t of g.script ?? []) {
        if (!speakers.includes(t.speaker)) {
          speakers.push(t.speaker);
        }
      }
      const kind = g.part === 3 ? 'conversation' : 'talk';
      const three = g.part === 3 && speakers.length > 2 ? ' with three speakers' : '';
      const graphic = g.graphic ? ' and graphic' : '';
      out.push({
        ...base,
        kind: 'lead',
        text: `Questions ${first.number} through ${last.number} refer to the following ${kind}${three}${graphic}.`,
        ...NARRATOR,
        pauseAfterMs: PAUSE.afterLead,
      });
      const turns = g.script ?? [];
      turns.forEach((t, i) => {
        out.push({
          ...base,
          kind: 'turn',
          text: t.text,
          lang: accentLang(t.accent),
          voice: speakers.indexOf(t.speaker),
          speaker: t.speaker,
          pauseAfterMs: i === turns.length - 1 ? PAUSE.afterTalk : PAUSE.betweenTurns,
        });
      });
      for (const q of g.questions) {
        out.push({
          ...base,
          kind: 'question',
          text: `Number ${q.number}. ${q.prompt ?? ''}`.trim(),
          ...NARRATOR,
          pauseAfterMs: PAUSE.afterQuestion34,
        });
      }
    }
  }
  return out;
}

/** The first segment of a group (its lead); -1 when the script does not have it. */
export function groupStart(script: readonly Segment[], key: string): number {
  return script.findIndex((s) => s.group === key);
}

/** Where to go back to for a segment: the start of its group, or the segment itself for directions. */
export function segmentStart(script: readonly Segment[], index: number): number {
  const s = script[index];
  if (!s || s.group === null) {
    return Math.max(0, index);
  }
  return groupStart(script, s.group);
}

/** Words a second at speed 1 (research R3: about 150 a minute). */
const WORDS_PER_SECOND = 2.5;

/** The estimated length of a script in seconds at a speed. */
export function scriptSeconds(script: readonly Segment[], rate = 1): number {
  let secs = 0;
  for (const s of script) {
    const words = s.text.split(/\s+/).filter(Boolean).length;
    secs += words / WORDS_PER_SECOND / rate + s.pauseAfterMs / 1000;
  }
  return secs;
}

/**
 * What the answer detail plays to hear a group again: its turns (and, in Part 1/2, the statements or
 * responses), without the directions, the lead or the long answer pauses.
 */
export function replayScript(group: PaperGroup): Segment[] {
  return buildScript([group], { directions: false })
    .filter((s) => s.kind === 'turn' || s.kind === 'option')
    .map((s) => ({ ...s, pauseAfterMs: Math.min(s.pauseAfterMs, PAUSE.betweenOptions) }));
}
