import { Step, STEP_LABELS, StepState, STEPS } from '../../core/models/study';

/**
 * The steps of a lesson being studied, in order (updated 2026-10-22): Từ vựng → Nghe → Đọc → Ngữ
 * pháp → Nói → Viết, all recorded on the server.
 */
export type FlowStep = Step;
export const LESSON_FLOW: readonly FlowStep[] = STEPS;

export const FLOW_STEP_NAMES: Record<FlowStep, string> = STEP_LABELS;

/**
 * The step on screen is also kept on this device, so coming back lands on the step left (it may be
 * one the server has not seen done, e.g. a step skipped with "Để sau").
 */
const STEP_KEY = (lessonId: string) => `luna.lesson-step.${lessonId}`;

export function readSavedStep(lessonId: string): string | null {
  try {
    return localStorage.getItem(STEP_KEY(lessonId));
  } catch {
    return null;
  }
}

export function writeSavedStep(lessonId: string, step: string | null): void {
  try {
    if (step === null) {
      localStorage.removeItem(STEP_KEY(lessonId));
    } else {
      localStorage.setItem(STEP_KEY(lessonId), step);
    }
  } catch {
    // Storage can be blocked (private window): the lesson then resumes from the server state alone.
  }
}

/**
 * The index in `steps` of the step to go on with: the one saved on this device, but never behind
 * the step after the last one the server knows is done. -1 when nothing is done or saved.
 */
export function resumeIndex(
  steps: readonly string[],
  saved: string | null,
  server: Partial<Record<Step, StepState>> | undefined,
): number {
  const doneAt = STEPS.filter((s) => server?.[s] === 'done').map((s) => steps.indexOf(s));
  const floor = doneAt.length ? Math.max(...doneAt) + 1 : -1;
  const index = saved ? steps.indexOf(saved) : -1;
  return Math.min(Math.max(index, floor), steps.length - 1);
}
