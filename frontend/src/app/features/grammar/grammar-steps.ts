/** The four steps of a grammar point (F20) and the step where the learner left each point. */

export type StepKey = 'theory' | 'structure' | 'practice' | 'test';

export const STEPS: readonly { key: StepKey; label: string }[] = [
  { key: 'theory', label: 'Lý thuyết' },
  { key: 'structure', label: 'Cấu trúc & ví dụ' },
  { key: 'practice', label: 'Luyện tập' },
  { key: 'test', label: 'Kiểm tra' },
];

/** Index in STEPS of a step named in a link (?step=practice); null for none or an unknown one. */
export function stepIndex(key: string | null): number | null {
  const i = STEPS.findIndex((s) => s.key === key);
  return i < 0 ? null : i;
}

const storageKey = (pointId: string) => `luna.grammar.step.${pointId}`;

/**
 * The step last opened on a point in this browser, 0 when none. Only a convenience for "Học tiếp
 * từ chỗ dở": the server keeps the scores, not where the learner stopped.
 */
export function savedStep(pointId: string): number {
  try {
    const n = Number(localStorage.getItem(storageKey(pointId)));
    return Number.isInteger(n) && n > 0 && n < STEPS.length ? n : 0;
  } catch {
    return 0;
  }
}

export function saveStep(pointId: string, index: number): void {
  try {
    localStorage.setItem(storageKey(pointId), String(index));
  } catch {
    // Storage blocked: the point simply opens on its first step.
  }
}
