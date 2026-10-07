import { Stats } from '../../core/models/dashboard';
import { IconName } from '../../shared/components/icon/icon';

/** A milestone of the stats page, earned once its figure reaches the goal. */
export interface Badge {
  id: string;
  label: string;
  icon: IconName;
  /** What it takes, read out with the badge. */
  hint: string;
  /** Where the learner is (capped at goal) and what earns it. */
  value: number;
  goal: number;
  earned: boolean;
}

/** Milestones: figure, goal and wording. Worked out from every-day figures, nothing is stored. */
const RULES: readonly {
  id: string;
  label: string;
  icon: IconName;
  goal: number;
  hint: (goal: number) => string;
  value: (stats: Stats, streak: number) => number;
}[] = [
  { id: 'streak', label: 'Học đều', icon: 'flame', goal: 7, hint: (g) => `Học ${g} ngày liên tiếp`, value: (_, streak) => streak },
  { id: 'words', label: 'Từ vựng', icon: 'notebook', goal: 100, hint: (g) => `Lưu ${g} từ vào sổ`, value: (s) => s.cards },
  { id: 'listen', label: 'Luyện nghe', icon: 'headphones', goal: 10, hint: (g) => `Xong bước Nghe ${g} bài`, value: (s) => s.lessons.listen },
  { id: 'read', label: 'Luyện đọc', icon: 'book', goal: 10, hint: (g) => `Xong bước Đọc ${g} bài`, value: (s) => s.lessons.read },
  { id: 'write', label: 'Luyện viết', icon: 'pencil', goal: 5, hint: (g) => `Nộp ${g} bài viết`, value: (s) => s.writing.submitted },
  { id: 'lessons', label: 'Chăm chỉ', icon: 'trophy', goal: 20, hint: (g) => `Hoàn thành ${g} bài học`, value: (s) => s.lessons.completed },
];

/** The badges for every-day figures and the current streak. */
export function badges(stats: Stats, streak: number): Badge[] {
  return RULES.map((r) => {
    const value = Math.max(0, r.value(stats, streak));
    return {
      id: r.id,
      label: r.label,
      icon: r.icon,
      hint: r.hint(r.goal),
      value: Math.min(value, r.goal),
      goal: r.goal,
      earned: value >= r.goal,
    };
  });
}
