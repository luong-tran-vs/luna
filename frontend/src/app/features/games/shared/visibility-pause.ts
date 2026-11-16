/**
 * Whether the page is hidden (another tab, the screen locked): the games then pause their clock
 * and show "Tạm dừng" (FR-007).
 */
export function pageHidden(doc: Document): boolean {
  return doc.visibilityState === 'hidden';
}

/** Whether the learner asked the system for less motion (FR-009). */
export function reducedMotion(win: Window | null | undefined): boolean {
  try {
    return !!win?.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
  } catch {
    return false;
  }
}
