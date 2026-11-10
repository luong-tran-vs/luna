import { VerbProgress } from '../../core/models/verbs';
import { VERB_LESSONS, VerbLesson } from './verb-lessons.data';
import { isLearned, nextLesson, reviewSummary } from './verb-study.logic';
import { isStruggling } from './verb-mistakes.logic';

/**
 * The groups of the "Động từ" tab (design/de-xuat-dong-tu.png part 7, added 2026-11-10): the
 * lessons that change the same way, together (the common lessons and the "Nâng cao" ones of one
 * kind), so the learner sees which kinds they know. Taken from the lessons' tags.
 */

interface GroupDef {
  key: string;
  label: string;
  tags: readonly string[];
}

const GROUP_DEFS: readonly GroupDef[] = [
  { key: 'khong-doi', label: 'Không đổi', tags: ['A – A – A', 'Không đổi'] },
  { key: 'v3-nhu-v1', label: 'V3 như V1', tags: ['A – B – A'] },
  { key: 'd-thanh-t', label: '-d → -t', tags: ['-d → -t'] },
  { key: 't-hay-ed', label: '-t hay -ed', tags: ['-t / -ed'] },
  { key: 'am-ngan', label: 'ee → e', tags: ['ee → e'] },
  { key: 'aid-old', label: '-aid / -old', tags: ['-aid / -old'] },
  { key: 'ought', label: '-ought', tags: ['-ought', '-ought / -ound'] },
  { key: 'v2-bang-v3', label: 'V2 = V3', tags: ['V2 = V3', 'i → u'] },
  { key: 'hay-gap', label: 'Hay gặp', tags: ['Hay gặp'] },
  { key: 'i-a-u', label: 'i – a – u', tags: ['i – a – u'] },
  { key: 'ew-own', label: '-ew → -own', tags: ['-ew → -own'] },
  {
    key: 'them-en',
    label: 'V3 thêm -n / -en',
    tags: ['o → o-en', 'i → o → en', '-en', '-ed → -n', '-n / -en'],
  },
  { key: 'con-lai', label: 'Còn lại', tags: ['Còn lại'] },
  {
    key: 'tien-to',
    label: 'Có tiền tố',
    tags: ['over-', 'out-', 'under-', 're-', 'mis-', 'with- / un-', 'fore- / in-'],
  },
  { key: 'hiem', label: 'Hiếm gặp', tags: ['Hiếm'] },
];

export interface VerbGroup {
  key: string;
  label: string;
  /** The rule, said by the cat: the tip of its first lesson. */
  tipVi: string;
  lessons: VerbLesson[];
  bases: string[];
}

/** Every group, in the order of their first lesson; a lesson whose tag no group lists is its own. */
export function verbGroups(lessons: readonly VerbLesson[] = VERB_LESSONS): VerbGroup[] {
  const byKey = new Map<string, VerbGroup>();
  for (const l of lessons) {
    const def = GROUP_DEFS.find((d) => d.tags.includes(l.tag)) ?? {
      key: `bai-${l.number}`,
      label: l.tag,
      tags: [],
    };
    let g = byKey.get(def.key);
    if (!g) {
      g = { key: def.key, label: def.label, tipVi: l.tipVi, lessons: [], bases: [] };
      byKey.set(def.key, g);
    }
    g.lessons.push(l);
    g.bases.push(...l.bases);
  }
  return [...byKey.values()];
}

/** How many of a group's verbs are learned. */
export function groupLearned(g: VerbGroup, progress: ReadonlyMap<string, VerbProgress>): number {
  return g.bases.filter((b) => isLearned(progress.get(b))).length;
}

/** One thing to do today on the "Động từ" tab. */
export interface TodayTask {
  key: 'review' | 'lesson' | 'mistakes';
  label: string;
  done: boolean;
}

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

/**
 * Today's tasks, only those that apply: review the due verbs (once something is learned), learn the
 * next lesson (while one is left; done once a verb was learned today), practise the verbs often typed
 * wrong (while there are some). The same figures as the rest of the tab.
 */
export function todayTasks(progress: ReadonlyMap<string, VerbProgress>, now: Date): TodayTask[] {
  const out: TodayTask[] = [];
  const review = reviewSummary(progress, now);
  if (review.learned > 0) {
    out.push({
      key: 'review',
      label: review.dueToday > 0 ? `Ôn ${review.dueToday} từ đến hạn` : 'Ôn các từ đến hạn',
      done: review.dueToday === 0,
    });
  }
  const learnedToday = [...progress.values()].some(
    (p) => !!p.learnedAt && sameDay(new Date(p.learnedAt), now),
  );
  const next = nextLesson(VERB_LESSONS, progress);
  if (next || learnedToday) {
    out.push({
      key: 'lesson',
      // Once a lesson is learned today the task is done: naming the next one would read as done too.
      label: learnedToday
        ? 'Đã học một bài hôm nay'
        : next
          ? `Học bài ${next.number} · ${next.title}`
          : 'Học một bài mới',
      done: learnedToday,
    });
  }
  const struggling = [...progress.values()].filter((p) => isStruggling(p, now)).length;
  if (struggling > 0) {
    out.push({ key: 'mistakes', label: `Luyện ${struggling} từ hay sai`, done: false });
  }
  return out;
}
