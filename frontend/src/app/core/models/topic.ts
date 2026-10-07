import { Level, LessonSummary } from './lesson';

/**
 * Types for the topic API (specs/007-topic-roadmaps/contracts/topics-api.md). A topic is shared by
 * every level; each (topic, level) pair has its own roadmap.
 */

/** One level of a topic: its lessons and roadmap. Only levels with lessons or a roadmap are listed. */
export interface TopicLevel {
  level: Level;
  lessonCount: number;
  roadmapCount: number;
  /** Roadmap lessons not yet studied (until feature L: every roadmap lesson). */
  remaining: number;
  warning: boolean;
}

export interface Topic {
  id: string;
  name: string;
  description: string;
  /** Lessons of every level. */
  lessonCount: number;
  /** A1 → C2. */
  levels: TopicLevel[];
  createdAt: string;
  /** F18: words in the topic's vocabulary list, and how many of them its lessons use. */
  wordCount: number;
  usedWordCount: number;
}

/** One word of a topic's vocabulary list with its coverage (specs/017-topic-vocabulary). */
export interface TopicWord {
  text: string;
  /** Lowest level the word is meant for; '' for every level. */
  level: Level | '';
  used: boolean;
  /** Lessons of the topic that use the word. */
  lessonCount: number;
}

/** A word as PUT .../words takes it. */
export interface TopicWordInput {
  text: string;
  level: Level | '';
}

/** Target words for one generation (F18): one group per lesson, and how many more unused words the topic needs. */
export interface WordPlan {
  groups: string[][];
  shortage: number;
}

export interface TopicInput {
  name: string;
  description: string;
}

export interface TopicRoadmap {
  topic: Topic;
  level: Level;
  lessons: LessonSummary[];
  remaining: number;
  warning: boolean;
}

/** The summary of one level of a topic (empty counts when it has nothing there). */
export function topicLevel(t: Pick<Topic, 'levels'>, level: Level): TopicLevel {
  return t.levels.find((l) => l.level === level) ?? { level, lessonCount: 0, roadmapCount: 0, remaining: 0, warning: true };
}

/** Roadmaps that have lessons but run low (fewer than 3 not studied). */
export function lowRoadmaps(topics: readonly Topic[]): { topic: Topic; level: TopicLevel }[] {
  return topics.flatMap((topic) =>
    topic.levels.filter((l) => l.lessonCount > 0 && l.warning).map((level) => ({ topic, level })),
  );
}

/** "A1 · Gia đình": the name of a roadmap. */
export function roadmapLabel(level: Level, t: Pick<Topic, 'name'>): string {
  return `${level} · ${t.name}`;
}

/** Topics sorted by name (Vietnamese order). */
export function sortTopics<T extends Pick<Topic, 'name'>>(topics: readonly T[]): T[] {
  return [...topics].sort((a, b) => a.name.localeCompare(b.name, 'vi'));
}
