/** Multiple choice (design/tat-ca-de-xuat.png (mục on-tap-theo-bai)): the right answer among a few others, shuffled. */

/** How many options a question shows, the right one included. */
export const CHOICE_COUNT = 4;

/** Case, spaces and a trailing full stop do not make two options different. */
export function choiceKey(text: string): string {
  return text.trim().replace(/\s+/g, ' ').replace(/\.$/, '').toLowerCase();
}

/** Small seeded PRNG (mulberry32): the same seed always gives the same numbers in [0, 1). */
function random(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** A number from a string, to shuffle each question the same way while it is on screen. */
export function seedOf(text: string, salt = 0): number {
  let h = 2166136261 ^ salt;
  for (let i = 0; i < text.length; i++) {
    h = Math.imul(h ^ text.charCodeAt(i), 16777619);
  }
  return h >>> 0;
}

/** Returns a shuffled copy of `items` (Fisher–Yates); the same seed gives the same order. */
export function shuffled<T>(items: readonly T[], seed: number): T[] {
  const next = random(seed);
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(next() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

export interface Choices {
  options: string[];
  /** Index of the right answer in `options`. */
  answer: number;
}

/**
 * The right answer and up to CHOICE_COUNT − 1 others, shuffled with `seed`. The others come from
 * `preferred` first (e.g. words of the same part of speech), then from `pool`; options that read the
 * same as the answer or as each other are skipped. Fewer options when the pools are small.
 */
export function makeChoices(
  answer: string,
  pool: readonly string[],
  seed: number,
  preferred: readonly string[] = [],
): Choices {
  const seen = new Set([choiceKey(answer)]);
  const others: string[] = [];
  for (const group of [preferred, pool]) {
    for (const text of shuffled(group, seed)) {
      const key = choiceKey(text);
      if (others.length < CHOICE_COUNT - 1 && key && !seen.has(key)) {
        seen.add(key);
        others.push(text.trim());
      }
    }
  }
  const options = shuffled([answer.trim(), ...others], seed + 1);
  return { options, answer: options.indexOf(answer.trim()) };
}
