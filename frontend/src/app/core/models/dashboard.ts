import { Level } from './lesson';
import { GoalView, Step, StepState, TodayKind } from './study';

/** Types for the dashboard API (specs/009-home-dashboard/contracts/dashboard-api.md). */

export interface SkillCounts {
  read: number;
  listen: number;
  write: number;
  /** Lessons in the current roadmap. */
  total: number;
}

export interface DashboardLesson {
  id: string;
  /** Empty when the lesson was deleted. */
  title: string;
  topicName: string;
  level: Level;
}

export interface DashboardAction {
  kind: 'start' | 'continue';
  step: Step;
}

export interface Dashboard {
  kind: TodayKind;
  goal: GoalView | null;
  goalCompleted: boolean;
  skills: SkillCounts | null;
  lesson: DashboardLesson | null;
  steps: Record<Step, StepState>;
  currentStep: Step | 'done' | '';
  action: DashboardAction | null;
  streak: number;
  /** Cards due before the end of tomorrow, overdue cards included. */
  tomorrowCards: number;
}

export interface Stats {
  cards: number;
  dictation: {
    sentences: number;
    correctWords: number;
    totalWords: number;
    /** correctWords / totalWords (0–1); null when nothing was checked. */
    rate: number | null;
  };
  lessons: {
    read: number;
    listen: number;
    write: number;
    completed: number;
  };
  /** Comprehension answers (F15). */
  reading: {
    answered: number;
    correct: number;
    /** correct / answered (0–1); null when nothing was answered. */
    rate: number | null;
  };
  /** Writings (F8): averageScore is null until one is graded. */
  writing: {
    submitted: number;
    averageScore: number | null;
  };
}

/** Mean of the comprehension and dictation rates that exist (0–1); null when neither has data. */
export function accuracyOf(stats: Stats): number | null {
  const rates = [stats.reading.rate, stats.dictation.rate].filter((r): r is number => r !== null);
  return rates.length ? rates.reduce((a, b) => a + b, 0) / rates.length : null;
}
