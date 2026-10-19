import { forkJoin, map, Observable } from 'rxjs';

import { StudyApiService } from '../../../core/services/study-api.service';
import { VocabApiService } from '../../../core/services/vocab-api.service';
import {
  Badge,
  badgeOf,
  byNeed,
  LessonRow,
  lessonRows,
  mastery,
  scoreText,
  seenLine,
} from '../lesson-review/lesson-review-logic';

/** A lesson of the list, ready to show. */
export interface LessonItem extends LessonRow {
  badge: Badge;
  /** Out of 5; null when the lesson was never reviewed in a round. */
  mastery: number | null;
  score: string;
  /** "Ôn lần cuối 3 ngày trước"; empty when the badge already says "Đã ôn". */
  seen: string;
}

/** Which lessons the full list shows. */
export type LessonFilter = 'all' | 'due' | 'new';

export function matches(item: LessonItem, filter: LessonFilter): boolean {
  switch (filter) {
    case 'due':
      return item.badge.tone === 'due';
    case 'new':
      return item.badge.tone === 'new';
    default:
      return true;
  }
}

/**
 * Every lesson learned, and the one being studied once words were saved from it, by how much it
 * needs a review (design/tat-ca-de-xuat.png (mục on-tap-bo-cuc)).
 */
export function loadLessonItems(
  study: StudyApiService,
  api: VocabApiService,
): Observable<LessonItem[]> {
  return forkJoin({ mine: study.myLessons(true), stats: api.lessonStats() }).pipe(
    map(({ mine, stats }) => {
      const now = Date.now();
      const current =
        mine.current && stats.some((s) => s.lessonId === mine.current?.id) ? [mine.current] : [];
      const rows = lessonRows(
        [...current.map((l) => ({ ...l, topicName: '', completedAt: '' })), ...mine.completed],
        stats,
      );
      return byNeed(rows).map((r) => {
        const badge = badgeOf(r, now);
        return {
          ...r,
          badge,
          mastery: mastery(r),
          score: scoreText(r),
          seen: badge.tone === 'done' ? '' : seenLine(r, now),
        };
      });
    }),
  );
}
