import { Level, LEVELS, LessonSummary } from './lesson';

/** Types for the topic API (specs/007-topic-roadmaps/contracts/topics-api.md). */

export interface Topic {
  id: string;
  name: string;
  level: Level;
  description: string;
  lessonCount: number;
  roadmapCount: number;
  /** Roadmap lessons not yet studied (until feature L: every roadmap lesson). */
  remaining: number;
  warning: boolean;
  createdAt: string;
}

export interface TopicInput {
  name: string;
  level: Level | '';
  description: string;
}

export interface TopicRoadmap {
  topic: Topic;
  lessons: LessonSummary[];
  remaining: number;
  warning: boolean;
}

export interface LevelGroup {
  level: Level;
  topics: Topic[];
}

/** Topics grouped by level A1 → C2, keeping the given order inside each level; empty levels skipped. */
export function groupByLevel(topics: readonly Topic[]): LevelGroup[] {
  return LEVELS.map((level) => ({ level, topics: topics.filter((t) => t.level === level) })).filter(
    (g) => g.topics.length > 0,
  );
}

/** "A1 · Gia đình" */
export function topicLabel(t: Pick<Topic, 'level' | 'name'>): string {
  return `${t.level} · ${t.name}`;
}
