/** A piece of a text block as it is drawn. */
export interface Segment {
  kind: 'text' | 'bold' | 'blank';
  text: string;
  /** The blank's number in the group (Part 6 "[[n]]"). */
  no?: number;
  /** 'evidence' (the answer's words) or 'user' (highlighted by the learner). */
  mark?: 'evidence' | 'user';
}

/** Words to mark in a block. */
export interface Mark {
  text: string;
  kind: 'evidence' | 'user';
}

const TOKEN = /\*\*(.+?)\*\*|\[\[(\d+)\]\]/g;

/**
 * Splits a block's text into plain words, **bold** words and the blanks [[n]] of Part 6, then marks
 * the given words (every place they appear; evidence wins over a learner's highlight).
 */
export function splitText(text: string, marks: readonly Mark[] = []): Segment[] {
  const out: Segment[] = [];
  let last = 0;
  for (const m of text.matchAll(TOKEN)) {
    const at = m.index ?? 0;
    if (at > last) {
      out.push(...markPieces(text.slice(last, at), 'text', marks));
    }
    if (m[1] !== undefined) {
      out.push(...markPieces(m[1], 'bold', marks));
    } else {
      out.push({ kind: 'blank', text: '', no: Number(m[2]) });
    }
    last = at + m[0].length;
  }
  if (last < text.length) {
    out.push(...markPieces(text.slice(last), 'text', marks));
  }
  return out;
}

function markPieces(text: string, kind: 'text' | 'bold', marks: readonly Mark[]): Segment[] {
  const ordered = [...marks]
    .filter((m) => m.text.trim() !== '')
    .sort((a, b) => (a.kind === b.kind ? 0 : a.kind === 'evidence' ? -1 : 1));
  let pieces: Segment[] = [{ kind, text }];
  for (const mark of ordered) {
    const next: Segment[] = [];
    for (const p of pieces) {
      if (p.mark) {
        next.push(p);
        continue;
      }
      let rest = p.text;
      let i = rest.indexOf(mark.text);
      while (i >= 0) {
        if (i > 0) {
          next.push({ kind, text: rest.slice(0, i) });
        }
        next.push({ kind, text: mark.text, mark: mark.kind });
        rest = rest.slice(i + mark.text.length);
        i = rest.indexOf(mark.text);
      }
      if (rest) {
        next.push({ kind, text: rest });
      }
    }
    pieces = next;
  }
  return pieces;
}
