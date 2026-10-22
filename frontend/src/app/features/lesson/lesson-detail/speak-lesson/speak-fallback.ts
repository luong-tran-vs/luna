import { PracticeTurn, ReadAloud, SpeakQuestion } from '../../../../core/models/practice';

/**
 * The Speaking step of a lesson whose practice was made before the six steps (2026-10-22) has no
 * sentences with tips nor questions from the AI. These build them from what the lesson has, so the
 * step still follows the design (section 5): 2 sentences with a pronunciation tip, then questions
 * from the dialogue with an answer frame.
 */

/** A word of the lesson with its pronunciation. */
export interface FallbackWord {
  text: string;
  lemma: string;
  ipa: string;
}

const READ_COUNT = 2;
const QUESTION_COUNT = 3;
const MAX_READ_WORDS = 12;
const MIN_READ_WORDS = 3;

const VOICELESS = ['p', 't', 'k', 'f', 'θ'];
const SIBILANTS = ['s', 'z', 'ʃ', 'ʒ', 'tʃ', 'dʒ'];
/** Final consonants Vietnamese learners often drop or soften. */
const FINAL_CONSONANTS = [
  'tʃ',
  'dʒ',
  'p',
  'b',
  't',
  'd',
  'k',
  'g',
  'f',
  'v',
  's',
  'z',
  'ʃ',
  'ʒ',
  'θ',
  'ð',
  'l',
];

/** The IPA without slashes, stress and length marks: "/ˈsɪŋ/" → "sɪŋ". */
function bareIpa(ipa: string): string {
  return ipa.replace(/[/[\]ˈˌː.]/g, '').trim();
}

function lastSound(ipa: string): string {
  const s = bareIpa(ipa);
  const two = s.slice(-2);
  return two === 'tʃ' || two === 'dʒ' ? two : s.slice(-1);
}

function wordsOf(sentence: string): string[] {
  return sentence.match(/[A-Za-z']+/g) ?? [];
}

/** A tip for one word of a sentence, '' when it has no sound worth a tip. */
export function tipFor(token: string, w: FallbackWord): string {
  const ipa = bareIpa(w.ipa);
  if (!ipa) {
    return '';
  }
  const t = token.toLowerCase();
  const base = (w.lemma || w.text).toLowerCase();
  const end = lastSound(ipa);
  if (t !== base && t.endsWith('s') && !base.endsWith('s')) {
    const sound = SIBILANTS.includes(end) ? 'ɪz' : VOICELESS.includes(end) ? 's' : 'z';
    return `${token} /${ipa}${sound}/: nhớ đọc âm cuối /${sound}/. Người Việt hay bỏ âm cuối -s.`;
  }
  if (t !== base && t.endsWith('ed')) {
    const sound =
      end === 't' || end === 'd'
        ? 'ɪd'
        : VOICELESS.includes(end) || SIBILANTS.includes(end)
          ? 't'
          : 'd';
    return `${token} /${ipa}${sound}/: đuôi -ed đọc là /${sound}/, nhớ đọc rõ âm cuối.`;
  }
  if (t !== base) {
    return '';
  }
  if (/[θð]/.test(ipa)) {
    const sound = ipa.includes('θ') ? 'θ' : 'ð';
    return `${token} /${ipa}/: âm /${sound}/ đặt đầu lưỡi giữa hai răng, không đọc thành /t/ hay /d/.`;
  }
  if (FINAL_CONSONANTS.includes(end)) {
    return `${token} /${ipa}/: nhớ đọc rõ âm cuối /${end}/. Người Việt hay nuốt âm cuối.`;
  }
  return '';
}

/** The word of the lesson a token is a form of, if any. */
function lessonWord(token: string, words: readonly FallbackWord[]): FallbackWord | null {
  const t = token.toLowerCase();
  for (const w of words) {
    const forms = [w.text, w.lemma].filter(Boolean).map((f) => f.toLowerCase());
    for (const f of forms) {
      if (t === f || t === f + 's' || t === f + 'es' || t === f + 'ed' || t === f + 'd') {
        return w;
      }
    }
  }
  return null;
}

/** The first tip of a sentence, '' when none of its words has one. */
function sentenceTip(sentence: string, words: readonly FallbackWord[]): string {
  for (const token of wordsOf(sentence)) {
    const w = lessonWord(token, words);
    const tip = w ? tipFor(token, w) : '';
    if (tip) {
      return tip;
    }
  }
  return '';
}

/**
 * Two short sentences to repeat, those with a word of the lesson worth a tip chosen first, each tip about a
 * different word. `sentences` are in order of preference (the dialogue's turns, then the text's).
 */
export function fallbackReadAloud(
  sentences: readonly string[],
  words: readonly FallbackWord[],
): ReadAloud[] {
  const fit = [...new Set(sentences.map((s) => s.trim()))].filter((s) => {
    const n = wordsOf(s).length;
    return n >= MIN_READ_WORDS && n <= MAX_READ_WORDS && !s.endsWith('?');
  });
  const out: ReadAloud[] = [];
  const tipped = new Set<string>();
  for (const text of fit) {
    const tipVi = sentenceTip(text, words);
    const key = tipVi.split(' ')[0];
    if (tipVi && !tipped.has(key)) {
      tipped.add(key);
      out.push({ text, tipVi });
    }
    if (out.length === READ_COUNT) {
      break;
    }
  }
  for (const text of fit) {
    if (out.length === READ_COUNT) {
      break;
    }
    if (!out.some((r) => r.text === text)) {
      out.push({ text, tipVi: '' });
    }
  }
  // In the order of the lesson.
  return out.sort((a, b) => fit.indexOf(a.text) - fit.indexOf(b.text));
}

/**
 * An answer frame from the reply to a question: its first sentence when it speaks of oneself
 * ("I …", "My …"), with the lesson's words (or else its last word) left blank.
 */
export function frameFrom(reply: string, words: readonly FallbackWord[]): string {
  let s = (reply.match(/^[^.!?]+[.!?]?/)?.[0] ?? '').trim();
  s = s.replace(/^(yes|no|well|oh|hmm|sure)\b[,!]?\s*/i, '');
  if (!/^(I|I'm|I've|My|We)\b/.test(s)) {
    return '';
  }
  let blanked = false;
  const out = s.replace(/[A-Za-z']+/g, (token, at: number) => {
    if (at > 0 && lessonWord(token, words)) {
      blanked = true;
      return '___';
    }
    return token;
  });
  if (blanked) {
    return out.replace(/(___\s+)+___/g, '___');
  }
  const m = s.match(/^(.*\s)([A-Za-z']+)([.!?]?)$/);
  return m ? `${m[1]}___${m[3] || '.'}` : '';
}

/** Up to three questions of the dialogue asked to "you", with the reply as the answer frame. */
export function fallbackQuestions(
  turns: readonly PracticeTurn[],
  words: readonly FallbackWord[],
): SpeakQuestion[] {
  const out: SpeakQuestion[] = [];
  const seen = new Set<string>();
  turns.forEach((t, i) => {
    const question = (t.text.match(/[^.!?]*\?/)?.[0] ?? '').trim();
    if (out.length === QUESTION_COUNT || !question || !/\byou(r)?\b/i.test(question)) {
      return;
    }
    const key = question.toLowerCase();
    if (seen.has(key)) {
      return;
    }
    seen.add(key);
    const next = turns[i + 1];
    const reply = next && next.speaker !== t.speaker ? next.text : '';
    out.push({
      question,
      questionVi: t.text.trim() === question ? t.meaningVi : '',
      frame: frameFrom(reply, words),
    });
  });
  return out;
}
