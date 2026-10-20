import { Step, StepState, STEPS } from '../../core/models/study';

/**
 * The main steps of a lesson being studied, in order (updated 2026-10-14). The server tracks Đọc,
 * Nghe and Viết; Từ vựng and Luyện nói are only known on this device. A lesson without words or
 * speaking sentences leaves those out on its page; the home page counts all five.
 */
export type FlowStep = 'words' | Step | 'speak';
export const LESSON_FLOW: readonly FlowStep[] = ['words', 'read', 'listen', 'speak', 'write'];

export const FLOW_STEP_NAMES: Record<FlowStep, string> = {
  words: 'Từ vựng',
  read: 'Đọc',
  listen: 'Nghe',
  speak: 'Luyện nói',
  write: 'Viết',
};

/** Where the learner stopped is kept on this device, since the server does not track every step. */
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
