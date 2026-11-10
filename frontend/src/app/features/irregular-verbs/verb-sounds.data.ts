/**
 * "Đọc khác viết" (design/de-xuat-dong-tu.png part 8, added 2026-11-10): forms said otherwise than
 * learners expect from how they are written, with what to listen for. The IPA comes from VERB_IPA
 * (American English). Which form: 0 V1, 1 V2, 2 V3.
 */
export interface VerbSound {
  base: string;
  /** The forms to hear, by index (0: V1, 1: V2, 2: V3). */
  forms: readonly (0 | 1 | 2)[];
  tipVi: string;
}

export const VERB_SOUNDS: readonly VerbSound[] = [
  { base: 'read', forms: [0, 1], tipVi: 'V2, V3 viết như V1 nhưng đọc /red/ (như "red").' },
  { base: 'say', forms: [0, 1], tipVi: 'said đọc /sed/, không phải /seɪd/.' },
  { base: 'mean', forms: [0, 1], tipVi: 'meant đọc /ment/: âm "ea" thành "e" ngắn.' },
  { base: 'hear', forms: [0, 1], tipVi: 'heard đọc /hɜːrd/, khác hẳn hear /hɪr/.' },
  { base: 'deal', forms: [0, 1], tipVi: 'dealt đọc /delt/, chữ "a" không đọc.' },
  { base: 'go', forms: [1, 2], tipVi: 'gone đọc /ɡɔːn/, không phải /ɡoʊn/.' },
  { base: 'do', forms: [1, 2], tipVi: 'done đọc /dʌn/, giống "dun".' },
  { base: 'be', forms: [2], tipVi: 'been đọc /bɪn/ (giọng Mỹ), ngắn như "bin".' },
  { base: 'buy', forms: [1], tipVi: 'bought đọc /bɔːt/: "gh" không đọc.' },
  { base: 'catch', forms: [1], tipVi: 'caught đọc /kɔːt/, cùng vần với bought, taught.' },
  { base: 'lose', forms: [0, 1], tipVi: 'lose đọc /luːz/ (âm z), lost đọc /lɔːst/.' },
  { base: 'choose', forms: [1, 2], tipVi: 'chose /tʃoʊz/ có một "o"; choose /tʃuːz/ có hai.' },
  { base: 'build', forms: [0, 1], tipVi: 'build, built: chữ "u" không đọc: /bɪld/, /bɪlt/.' },
  { base: 'eat', forms: [1], tipVi: 'ate đọc /eɪt/, giống "eight".' },
];
