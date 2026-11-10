/**
 * Pairs of verbs learners mix up (design/de-xuat-dong-tu.png part 6, added 2026-11-10), shown beside
 * a verb of "Từ hay sai". The other verb of a pair may be regular ("raise"); its forms are written
 * here. Each tip says how to tell them apart.
 */
export interface VerbPair {
  /** The two verbs, each with its three forms and Vietnamese meaning. */
  verbs: readonly [PairVerb, PairVerb];
  tipVi: string;
}

export interface PairVerb {
  base: string;
  forms: string;
  meaningVi: string;
}

export const VERB_PAIRS: readonly VerbPair[] = [
  {
    verbs: [
      { base: 'lie', forms: 'lie · lay · lain', meaningVi: 'nằm' },
      { base: 'lay', forms: 'lay · laid · laid', meaningVi: 'đặt, để' },
    ],
    tipVi: 'lay luôn có vật đi sau (lay the book); lie thì không. V2 của lie lại chính là lay.',
  },
  {
    verbs: [
      { base: 'rise', forms: 'rise · rose · risen', meaningVi: 'mọc, tăng lên' },
      { base: 'raise', forms: 'raise · raised · raised', meaningVi: 'nâng lên' },
    ],
    tipVi: 'raise có vật đi sau (raise your hand) và là động từ có quy tắc; rise tự nó lên.',
  },
  {
    verbs: [
      { base: 'fall', forms: 'fall · fell · fallen', meaningVi: 'rơi, ngã' },
      { base: 'feel', forms: 'feel · felt · felt', meaningVi: 'cảm thấy' },
    ],
    tipVi: 'fell là V2 của fall (ngã), felt là V2 của feel (cảm thấy): đừng nhầm hai chữ e.',
  },
  {
    verbs: [
      { base: 'find', forms: 'find · found · found', meaningVi: 'tìm thấy' },
      { base: 'found', forms: 'found · founded · founded', meaningVi: 'thành lập' },
    ],
    tipVi: 'found vừa là V2/V3 của find, vừa là động từ "thành lập" có quy tắc (founded).',
  },
  {
    verbs: [
      { base: 'sit', forms: 'sit · sat · sat', meaningVi: 'ngồi' },
      { base: 'set', forms: 'set · set · set', meaningVi: 'đặt, sắp đặt' },
    ],
    tipVi: 'set có vật đi sau và không đổi ở cả ba dạng; sit đổi thành sat.',
  },
  {
    verbs: [
      { base: 'see', forms: 'see · saw · seen', meaningVi: 'nhìn thấy' },
      { base: 'saw', forms: 'saw · sawed · sawn', meaningVi: 'cưa' },
    ],
    tipVi: 'saw thường là V2 của see; động từ saw (cưa) có V2 sawed, V3 sawn.',
  },
  {
    verbs: [
      { base: 'wind', forms: 'wind · wound · wound', meaningVi: 'cuộn, lên dây' },
      { base: 'wound', forms: 'wound · wounded · wounded', meaningVi: 'làm bị thương' },
    ],
    tipVi: 'wound /waʊnd/ là V2 của wind; wound /wuːnd/ (làm bị thương) có quy tắc.',
  },
];

/** The pair a verb belongs to, if any. */
export function pairOf(base: string): VerbPair | null {
  return VERB_PAIRS.find((p) => p.verbs.some((v) => v.base === base)) ?? null;
}
