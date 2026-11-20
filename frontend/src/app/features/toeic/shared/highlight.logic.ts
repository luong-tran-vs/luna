/**
 * The learner's yellow highlights of an attempt (FR-009, research R11): words of a block, kept on
 * this device only (localStorage "toeic-highlights:<attemptId>").
 */
export interface Highlight {
  /** The block: "<groupKey>/<doc>/<block>". */
  block: string;
  text: string;
}

export const HIGHLIGHT_PREFIX = 'toeic-highlights:';

/** The id of a block of a group's text. */
export function blockId(groupKey: string, doc: number, block: number): string {
  return `${groupKey}/${doc}/${block}`;
}

/** Adds a highlight; the same words of a block are kept once, blank ones not at all. */
export function addHighlight(list: readonly Highlight[], h: Highlight): Highlight[] {
  const text = h.text.trim();
  if (!text || list.some((x) => x.block === h.block && x.text === text)) {
    return [...list];
  }
  return [...list, { block: h.block, text }];
}

/** Removes the highlights of a block that overlap the words (contain them or are inside them). */
export function removeHighlight(list: readonly Highlight[], h: Highlight): Highlight[] {
  const text = h.text.trim();
  return list.filter(
    (x) => x.block !== h.block || !(x.text.includes(text) || text.includes(x.text)),
  );
}

/** The highlighted words of a block. */
export function highlightsOf(list: readonly Highlight[], block: string): string[] {
  return list.filter((x) => x.block === block).map((x) => x.text);
}

/** Reads the list kept for an attempt; anything unreadable is an empty list. */
export function parseHighlights(raw: string | null): Highlight[] {
  if (!raw) {
    return [];
  }
  try {
    const v: unknown = JSON.parse(raw);
    return Array.isArray(v)
      ? v.filter(
          (x): x is Highlight =>
            typeof x === 'object' &&
            x !== null &&
            typeof (x as Highlight).block === 'string' &&
            typeof (x as Highlight).text === 'string',
        )
      : [];
  } catch {
    return [];
  }
}
