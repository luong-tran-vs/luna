import { differsByEnding, sameSkeleton } from './answer-match';
import { Comparison, ComparedWord } from './dictation-compare';

/** A kind of listening slip, from the words of the dictation results (added 2026-10-20). */
export type MishearingKind = 'weak' | 'ending' | 'linking' | 'vowel' | 'spelling';

export interface Mishearing {
  kind: MishearingKind;
  title: string;
  tip: string;
  count: number;
  /** A few of the learner's own slips, e.g. "to (bỏ sót)", "like → likes". */
  examples: string[];
}

/**
 * Small words said weakly inside a sentence, which learners often miss: articles, prepositions,
 * auxiliaries, pronouns and conjunctions.
 */
const WEAK_WORDS = new Set([
  'a', 'an', 'the', 'to', 'of', 'for', 'and', 'at', 'from', 'in', 'on', 'or', 'but', 'as',
  'is', 'are', 'was', 'were', 'am', 'be', 'been', 'do', 'does', 'can', 'could', 'have', 'has', 'had',
  'will', 'would', 'should', 'must', 'that', 'than', 'them', 'him', 'her', 'his', 'your', 'some',
]);

const INFO: Record<MishearingKind, { title: string; tip: string }> = {
  weak: {
    title: 'Từ nhỏ đọc lướt',
    tip: 'Trong câu, a, the, to, of, and… đọc rất nhẹ: to → /tə/, the → /ðə/, and → /ən/. Nghe kỹ khoảng giữa hai từ chính.',
  },
  ending: {
    title: 'Âm cuối bị nuốt (-s, -ed)',
    tip: 'Đuôi -s đọc /s/, /z/ hoặc /ɪz/; đuôi -ed đọc /t/, /d/ hoặc /ɪd/. Chúng nhẹ nhưng đổi nghĩa câu (số nhiều, quá khứ).',
  },
  linking: {
    title: 'Nối âm',
    tip: 'Phụ âm cuối nối sang nguyên âm đầu của từ sau, nên hai từ nghe như một: a lot → /əˈlɑːt/, turn on → /tɜːrˈnɑːn/.',
  },
  vowel: {
    title: 'Nguyên âm gần giống',
    tip: 'Hai từ chỉ khác nguyên âm (went/want, ship/sheep). Nghe lại chậm và để ý độ mở miệng, độ dài của nguyên âm.',
  },
  spelling: {
    title: 'Nghe đúng, viết chưa đúng',
    tip: 'Bạn nghe ra đúng từ, chỉ sai chính tả. Xem lại cách viết của các từ này.',
  },
};

function classify(w: ComparedWord): { kind: MishearingKind; example: string } | null {
  const typed = w.word ?? '';
  const expected = w.expected ?? '';
  if (w.status === 'close') {
    return (w.count ?? 1) > 1
      ? { kind: 'linking', example: `${typed} → ${expected}` }
      : { kind: 'spelling', example: `${typed} → ${expected}` };
  }
  if (w.status === 'missing') {
    return WEAK_WORDS.has(expected) ? { kind: 'weak', example: `${expected} (bỏ sót)` } : null;
  }
  if (w.status !== 'wrong' || !expected) {
    return null;
  }
  if (WEAK_WORDS.has(expected) && WEAK_WORDS.has(typed)) {
    return { kind: 'weak', example: `${typed} → ${expected}` };
  }
  if (differsByEnding(typed, expected)) {
    return { kind: 'ending', example: `${typed} → ${expected}` };
  }
  if (sameSkeleton(typed, expected)) {
    return { kind: 'vowel', example: `${typed} → ${expected}` };
  }
  return null;
}

/**
 * What the learner most often heard wrong in a set of dictation results, most frequent first, at
 * most `limit` kinds. Only slips that clearly fit a kind are counted; the others say nothing.
 */
export function mishearings(results: readonly Comparison[], limit = 3): Mishearing[] {
  const found = new Map<MishearingKind, Mishearing>();
  for (const r of results) {
    for (const w of r.words) {
      const c = classify(w);
      if (!c) {
        continue;
      }
      const m = found.get(c.kind) ?? { kind: c.kind, ...INFO[c.kind], count: 0, examples: [] };
      m.count++;
      if (m.examples.length < 3 && !m.examples.includes(c.example)) {
        m.examples.push(c.example);
      }
      found.set(c.kind, m);
    }
  }
  return [...found.values()].sort((a, b) => b.count - a.count).slice(0, limit);
}
