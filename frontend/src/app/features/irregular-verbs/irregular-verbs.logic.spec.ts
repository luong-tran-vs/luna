import { IRREGULAR_VERBS, IrregularVerb } from './irregular-verbs.data';
import { byLetter, filterVerbs, fold, patternOf, spellings, spokenForms } from './irregular-verbs.logic';

const verb = (base: string, past: string, participle: string, meaningVi = '', common = false): IrregularVerb => ({
  base,
  past,
  participle,
  meaningVi,
  common,
});

describe('irregular verbs data', () => {
  it('is sorted, without duplicates, with every field filled', () => {
    const bases = IRREGULAR_VERBS.map((v) => v.base);
    expect(bases).toEqual([...bases].sort());
    expect(new Set(bases).size).toBe(bases.length);
    for (const v of IRREGULAR_VERBS) {
      for (const form of [v.base, ...spellings(v.past), ...spellings(v.participle)]) {
        expect(form, v.base).toMatch(/^[a-z]+$/);
      }
      expect(v.meaningVi.trim(), v.base).not.toBe('');
    }
  });

  it('has the verbs learners need most, with the reference typos fixed', () => {
    const get = (base: string) => IRREGULAR_VERBS.find((v) => v.base === base);
    for (const base of ['be', 'do', 'have', 'go', 'get', 'make', 'take', 'hold', 'set', 'shut', 'bite']) {
      expect(get(base)?.common, base).toBe(true);
    }
    expect(get('foresee')?.participle).toBe('foreseen');
    expect(get('work')).toBeUndefined();
    expect(IRREGULAR_VERBS.length).toBeGreaterThan(200);
    expect(IRREGULAR_VERBS.filter((v) => v.common).length).toBeGreaterThan(90);
  });
});

describe('irregular verbs logic', () => {
  it('finds the pattern from the first spelling of each form', () => {
    expect(patternOf(verb('cut', 'cut', 'cut'))).toBe('AAA');
    expect(patternOf(verb('buy', 'bought', 'bought'))).toBe('ABB');
    expect(patternOf(verb('come', 'came', 'come'))).toBe('ABA');
    expect(patternOf(verb('go', 'went', 'gone'))).toBe('ABC');
    expect(patternOf(verb('burn', 'burnt/burned', 'burnt/burned'))).toBe('ABB');
    expect(patternOf(verb('get', 'got', 'got/gotten'))).toBe('ABB');
  });

  it('splits spellings and says the three forms', () => {
    expect(spellings('chid/ chided ')).toEqual(['chid', 'chided']);
    expect(spokenForms(verb('be', 'was/were', 'been'))).toBe('be, was, been');
  });

  it('searches the forms by their start and the meaning without Vietnamese marks', () => {
    const verbs = [verb('go', 'went', 'gone', 'đi'), verb('see', 'saw', 'seen', 'nhìn thấy', true), verb('saw', 'sawed', 'sawn/sawed', 'cưa')];
    const bases = (q: string, commonOnly = false) => filterVerbs(verbs, { query: q, commonOnly, pattern: null }).map((v) => v.base);
    expect(bases('WENT')).toEqual(['go']);
    expect(bases('di')).toEqual(['go']);
    expect(bases('nhin thay')).toEqual(['see']);
    // An exact form comes first: "saw" is the past of see and the base of saw.
    expect(bases('saw')).toEqual(['saw', 'see']);
    expect(bases('sawn')).toEqual(['saw']);
    expect(bases('', true)).toEqual(['see']);
    expect(filterVerbs(verbs, { query: '', commonOnly: false, pattern: 'ABC' }).map((v) => v.base)).toEqual(['go', 'see', 'saw']);
    expect(fold(' Đường ')).toBe('duong');
  });

  it('groups by first letter', () => {
    const groups = byLetter([verb('bear', 'bore', 'borne'), verb('beat', 'beat', 'beaten'), verb('cut', 'cut', 'cut')]);
    expect(groups.map((g) => [g.letter, g.verbs.length])).toEqual([
      ['B', 2],
      ['C', 1],
    ]);
  });
});
