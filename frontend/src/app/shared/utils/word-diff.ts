/** One run of the word diff: unchanged, added in the corrected text, or removed from the original. */
export interface DiffPart {
  kind: 'same' | 'add' | 'del';
  text: string;
}

/** Words with their trailing whitespace, so joining tokens gives the text back. */
function tokens(text: string): string[] {
  return text.match(/\S+\s*|\s+/g) ?? [];
}

/** Compares tokens ignoring the whitespace after them. */
const word = (t: string) => t.trimEnd();

/**
 * Word-level diff of original → corrected (longest common subsequence). Joining the "same" and
 * "del" parts gives the original; joining "same" and "add" gives the corrected text.
 */
export function wordDiff(original: string, corrected: string): DiffPart[] {
  const a = tokens(original);
  const b = tokens(corrected);
  // lcs[i][j] = length of the LCS of a[i:] and b[j:].
  const lcs: number[][] = Array.from({ length: a.length + 1 }, () => new Array<number>(b.length + 1).fill(0));
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = b.length - 1; j >= 0; j--) {
      lcs[i][j] = word(a[i]) === word(b[j]) ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1]);
    }
  }

  const parts: DiffPart[] = [];
  const push = (kind: DiffPart['kind'], text: string) => {
    const last = parts[parts.length - 1];
    if (last?.kind === kind) {
      last.text += text;
    } else {
      parts.push({ kind, text });
    }
  };
  let i = 0;
  let j = 0;
  while (i < a.length || j < b.length) {
    if (i < a.length && j < b.length && word(a[i]) === word(b[j])) {
      // Keep the corrected spacing so the "same" text reads like the corrected version.
      push('same', b[j]);
      i++;
      j++;
    } else if (j < b.length && (i === a.length || lcs[i][j + 1] >= lcs[i + 1][j])) {
      if (i < a.length && lcs[i][j + 1] === lcs[i + 1][j]) {
        // Equal choice: show the removal first, then the addition.
        push('del', a[i]);
        i++;
      } else {
        push('add', b[j]);
        j++;
      }
    } else {
      push('del', a[i]);
      i++;
    }
  }
  return parts;
}
