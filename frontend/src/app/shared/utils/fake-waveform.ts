/**
 * An illustrative waveform built from a sentence (F4 listening, F17 dialogue): syllables become
 * bumps, words a short gap, punctuation a pause, function words are quieter and the first syllable
 * of a long word louder. It is not decoded from any audio, so it only roughly follows the speech.
 */

/** Short words said without stress. */
const FUNCTION_WORDS = new Set([
  'a', 'an', 'the', 'to', 'of', 'in', 'on', 'at', 'for', 'and', 'or', 'but', 'is', 'are', 'was',
  'were', 'be', 'am', 'i', 'you', 'he', 'she', 'it', 'we', 'they', 'my', 'your', 'his', 'her',
  'our', 'their', 'me', 'him', 'us', 'them', 'do', 'does', 'did', 'have', 'has', 'can', 'will',
]);

/** Timings at normal speed, in milliseconds. */
const SYLLABLE_MS = 190;
const WORD_GAP_MS = 40;
const COMMA_MS = 250;
const STOP_MS = 450;
/** Envelope resolution. */
const STEP_MS = 10;

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

/** FNV-1a hash of the text, so a sentence always gets the same shape. */
function hash(text: string): number {
  let h = 2166136261;
  for (let i = 0; i < text.length; i++) {
    h = Math.imul(h ^ text.charCodeAt(i), 16777619);
  }
  return h >>> 0;
}

function syllables(word: string): number {
  return Math.max(1, word.toLowerCase().match(/[aeiouy]+/g)?.length ?? 1);
}

/** Loudness every STEP_MS for the whole sentence. */
function envelope(text: string, rand: () => number): number[] {
  const env: number[] = [];
  const push = (ms: number, f: (t: number) => number) => {
    const n = Math.max(1, Math.round(ms / STEP_MS));
    for (let i = 0; i < n; i++) {
      env.push(f(i / n));
    }
  };
  // Real silence is never flat zero.
  const silence = (ms: number) => push(ms, () => 0.04 + 0.03 * rand());

  for (const token of text.match(/[A-Za-z0-9’']+|[.,!?;:]/g) ?? []) {
    if (/[.!?]/.test(token)) {
      silence(STOP_MS);
      continue;
    }
    if (/[,;:]/.test(token)) {
      silence(COMMA_MS);
      continue;
    }
    const n = syllables(token);
    const base = FUNCTION_WORDS.has(token.toLowerCase()) ? 0.45 : 0.7;
    for (let s = 0; s < n; s++) {
      const peak = (s === 0 && n > 1 ? 0.95 : base) * (0.85 + 0.3 * rand());
      // Fast attack, slower release.
      push(SYLLABLE_MS - 30 + 60 * rand(), (t) => peak * Math.pow(Math.sin(Math.PI * t), 0.6));
    }
    silence(WORD_GAP_MS);
  }
  return env;
}

/** About how long the sentence takes to say at normal speed, in seconds. */
export function estimateSeconds(text: string): number {
  return Math.max(0.5, (envelope(text, () => 0.5).length * STEP_MS) / 1000);
}

/** `bars` heights in [0, 1] for `text`; the same text and count always give the same heights. */
export function fakeWaveform(text: string, bars: number): number[] {
  if (bars <= 0) {
    return [];
  }
  const rand = random(hash(text));
  const env = envelope(text, rand);
  if (env.length === 0) {
    return Array(bars).fill(0.05);
  }
  const raw = Array.from({ length: bars }, (_, b) => {
    const from = Math.floor((b * env.length) / bars);
    const to = Math.max(from + 1, Math.floor(((b + 1) * env.length) / bars));
    return Math.max(...env.slice(from, to)) * (0.75 + 0.25 * rand());
  });
  const smooth = raw.map((v, i) => ((raw[i - 1] ?? v) + v * 2 + (raw[i + 1] ?? v)) / 4);
  const top = Math.max(...smooth);
  // Normalized, then compressed so quiet syllables still show.
  return smooth.map((v) => Math.max(0.05, Math.pow(v / top, 0.7)));
}
