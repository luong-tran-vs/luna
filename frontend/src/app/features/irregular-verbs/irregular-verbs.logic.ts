import { IrregularVerb } from './irregular-verbs.data';

/**
 * How the three forms change: AAA all the same (cut – cut – cut), ABB past and participle the same
 * (buy – bought – bought), ABA participle like the base (come – came – come), ABC all different
 * (go – went – gone).
 */
export type VerbPattern = 'AAA' | 'ABB' | 'ABA' | 'ABC';

export const PATTERNS: readonly { key: VerbPattern; label: string; example: string; hintVi: string }[] = [
  { key: 'AAA', label: 'A – A – A', example: 'cut – cut – cut', hintVi: 'Ba dạng giống hệt nhau.' },
  { key: 'ABB', label: 'A – B – B', example: 'buy – bought – bought', hintVi: 'V2 và V3 giống nhau.' },
  { key: 'ABA', label: 'A – B – A', example: 'come – came – come', hintVi: 'V3 giống nguyên mẫu.' },
  { key: 'ABC', label: 'A – B – C', example: 'go – went – gone', hintVi: 'Ba dạng khác nhau, cần học thuộc.' },
];

/** The spellings of a form: "burnt/burned" gives ["burnt", "burned"]. */
export function spellings(form: string): string[] {
  return form
    .split('/')
    .map((s) => s.trim())
    .filter(Boolean);
}

/** The pattern of a verb, from the first spelling of each form. */
export function patternOf(v: Pick<IrregularVerb, 'base' | 'past' | 'participle'>): VerbPattern {
  const [a, b, c] = [v.base, spellings(v.past)[0], spellings(v.participle)[0]];
  if (a === b && b === c) {
    return 'AAA';
  }
  if (b === c) {
    return 'ABB';
  }
  return a === c ? 'ABA' : 'ABC';
}

/** What is read aloud for a verb: its three forms, first spellings, e.g. "go, went, gone". */
export function spokenForms(v: Pick<IrregularVerb, 'base' | 'past' | 'participle'>): string {
  return [v.base, spellings(v.past)[0], spellings(v.participle)[0]].join(', ');
}

/** Lowercase without Vietnamese marks, so "nhin thay" finds "nhìn thấy". */
export function fold(s: string): string {
  return s
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'd')
    .toLowerCase()
    .trim();
}

export interface VerbFilter {
  query: string;
  commonOnly: boolean;
  pattern: VerbPattern | null;
}

/**
 * The verbs that match the filter, alphabetical; a verb whose base, then another form, equals the
 * query comes first (typing "went" shows "go" on top, "saw" shows saw before see).
 */
export function filterVerbs(verbs: readonly IrregularVerb[], f: VerbFilter): IrregularVerb[] {
  const q = fold(f.query);
  const out = verbs.filter((v) => {
    if ((f.commonOnly && !v.common) || (f.pattern && patternOf(v) !== f.pattern)) {
      return false;
    }
    if (!q) {
      return true;
    }
    const forms = [v.base, ...spellings(v.past), ...spellings(v.participle)];
    return forms.some((form) => form.startsWith(q)) || fold(v.meaningVi).includes(q);
  });
  if (!q) {
    return out;
  }
  // 0: the base is the query, 1: another form is, 2: the rest; the sort keeps the order within each.
  const rank = (v: IrregularVerb) =>
    v.base === q ? 0 : [...spellings(v.past), ...spellings(v.participle)].includes(q) ? 1 : 2;
  return out.map((v) => ({ v, r: rank(v) })).sort((a, b) => a.r - b.r).map((x) => x.v);
}

/** Verbs grouped by first letter, in order. */
export function byLetter(verbs: readonly IrregularVerb[]): { letter: string; verbs: IrregularVerb[] }[] {
  const groups: { letter: string; verbs: IrregularVerb[] }[] = [];
  for (const v of verbs) {
    const letter = v.base[0].toUpperCase();
    const last = groups[groups.length - 1];
    if (last?.letter === letter) {
      last.verbs.push(v);
    } else {
      groups.push({ letter, verbs: [v] });
    }
  }
  return groups;
}
