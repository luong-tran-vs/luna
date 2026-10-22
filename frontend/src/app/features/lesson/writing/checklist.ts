/**
 * The "Cần có" checklist of the Writing step: which lesson words and grammar words a text uses,
 * matched as whole words, case-insensitive, with the simple inflections of a lesson word.
 */

/** Letters and digits: a match must not touch one, so "is" does not match inside "this". */
const L = '\\p{L}\\p{N}';

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

/** Curly apostrophes as typed on phones, made straight (same length, so offsets stay valid). */
const straight = (s: string) => s.replace(/[‘’]/g, "'");

/** A word and its simple inflections: -s, -es, -ed, -ing (dance → dancing, study → studies, stop → stopped). */
export function wordForms(word: string): string[] {
  const w = word.toLowerCase();
  const forms = [w, `${w}s`, `${w}es`, `${w}ed`, `${w}ing`];
  if (w.endsWith('e')) {
    forms.push(`${w}d`, `${w.slice(0, -1)}ing`);
  }
  if (/[^aeiou]y$/.test(w)) {
    forms.push(`${w.slice(0, -1)}ies`, `${w.slice(0, -1)}ied`);
  }
  if (/(^|[^aeiou])[aeiou][bdgmnprt]$/.test(w)) {
    const last = w.slice(-1);
    forms.push(`${w}${last}ed`, `${w}${last}ing`);
  }
  return forms;
}

/** The pattern of a word or phrase; only the first word of a phrase is inflected ("turned on"). */
function pattern(target: string, inflect: boolean): string {
  const [first, ...rest] = straight(target.trim()).split(/\s+/);
  const head = inflect ? wordForms(first).map(escape).join('|') : escape(first.toLowerCase());
  return [`(?:${head})`, ...rest.map(escape)].join('\\s+');
}

const bounded = (inner: string) => `(?<![${L}])(?:${inner})(?![${L}])`;

/** Whether text uses the lesson word or phrase (or an inflection of it). */
export function containsWord(text: string, target: string): boolean {
  return !!target.trim() && new RegExp(bounded(pattern(target, true)), 'iu').test(straight(text));
}

/** Whether text has one of the words exactly (whole word, case-insensitive). */
export function containsAny(text: string, words: string[]): boolean {
  const alts = words.filter((w) => w.trim()).map((w) => pattern(w, false));
  return alts.length > 0 && new RegExp(bounded(alts.join('|')), 'iu').test(straight(text));
}

/**
 * Up to n distinct lesson words to ask for: inflected forms give way to their base word
 * ("songs" to "song"), single words come before phrases.
 */
export function pickWords(targets: string[], n = 3): string[] {
  const seen = new Set<string>();
  const unique = targets
    .map((t) => t.trim())
    .filter((t) => t && !seen.has(t.toLowerCase()) && seen.add(t.toLowerCase()));
  const bases = unique.filter(
    (c) => !unique.some((b) => b.toLowerCase() !== c.toLowerCase() && containsWord(c, b) && c.length > b.length),
  );
  const single = (w: string) => !/\s/.test(w);
  return [...bases.filter(single), ...bases.filter((w) => !single(w))].slice(0, n);
}

export interface Segment {
  text: string;
  /** "word": a lesson word, "grammar": one of the grammar words, "plain": the rest. */
  kind: 'plain' | 'word' | 'grammar';
}

/** The text cut into runs, marking the lesson words (with inflections) and the grammar words. */
export function segments(text: string, words: string[], grammar: string[] = []): Segment[] {
  const byLength = (a: string, b: string) => b.length - a.length;
  const w = [...words].filter((x) => x.trim()).sort(byLength).map((x) => pattern(x, true));
  const g = [...grammar].filter((x) => x.trim()).sort(byLength).map((x) => pattern(x, false));
  if (!text || (!w.length && !g.length)) {
    return text ? [{ text, kind: 'plain' }] : [];
  }
  const groups = [w.length ? `(${w.join('|')})` : '(?!)', g.length ? `(${g.join('|')})` : '(?!)'];
  const re = new RegExp(`(?<![${L}])(?:${groups.join('|')})(?![${L}])`, 'giu');
  const out: Segment[] = [];
  const source = straight(text);
  let at = 0;
  for (const m of source.matchAll(re)) {
    const start = m.index ?? 0;
    if (start > at) {
      out.push({ text: text.slice(at, start), kind: 'plain' });
    }
    out.push({ text: text.slice(start, start + m[0].length), kind: m[1] !== undefined ? 'word' : 'grammar' });
    at = start + m[0].length;
  }
  if (at < text.length) {
    out.push({ text: text.slice(at), kind: 'plain' });
  }
  return out;
}
