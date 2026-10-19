/**
 * The offer to review the lesson before, made when a new lesson starts: once the review round is
 * done it is not made again for that lesson on this device. "Bỏ qua" or leaving the review
 * unfinished does not settle it; a step of the new lesson done ends it anyway.
 */
const OFFER_KEY = (lessonId: string) => `luna.review-settled.${lessonId}`;

export function reviewOfferSettled(lessonId: string): boolean {
  try {
    return localStorage.getItem(OFFER_KEY(lessonId)) !== null;
  } catch {
    return true; // without storage it would come back on every visit: better not to offer it
  }
}

export function settleReviewOffer(lessonId: string): void {
  try {
    localStorage.setItem(OFFER_KEY(lessonId), '1');
  } catch {
    // Storage can be blocked (private window): nothing to keep.
  }
}
