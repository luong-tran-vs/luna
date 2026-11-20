import { PaperGroup, SaveProgress, ToeicPart } from '../../../core/models/toeic';

/** A question of the exam with where it sits. */
export interface ExamQuestion {
  id: string;
  number: number;
  part: ToeicPart;
  /** The page it is shown on. */
  page: number;
}

/** What is shown at once: a group with its text (Part 6/7…), or a few Part 5 questions. */
export interface ExamPage {
  part: ToeicPart;
  groups: number[];
  questionIds: string[];
}

/** The answers, flags and position of an attempt in the exam room. */
export interface ExamState {
  answers: Readonly<Record<string, number>>;
  flags: readonly string[];
  position: string;
}

export type CellState = 'current' | 'flagged' | 'answered' | 'empty';

/** How many Part 5 questions one page holds. */
export const PART5_PER_PAGE = 5;

/** Cuts the paper into pages: Part 5 (and Part 2) a few questions a page, the others a group each. */
export function pagesOf(paper: readonly PaperGroup[]): ExamPage[] {
  const pages: ExamPage[] = [];
  paper.forEach((g, gi) => {
    const last = pages.at(-1);
    const stack = (g.part === 5 || g.part === 2) && last?.part === g.part && last.groups.length < PART5_PER_PAGE;
    if (stack && last) {
      last.groups.push(gi);
      last.questionIds.push(...g.questions.map((q) => q.id));
    } else {
      pages.push({ part: g.part, groups: [gi], questionIds: g.questions.map((q) => q.id) });
    }
  });
  return pages;
}

/** Every question in order, with its page. */
export function questionsOf(pages: readonly ExamPage[], paper: readonly PaperGroup[]): ExamQuestion[] {
  const out: ExamQuestion[] = [];
  pages.forEach((p, pi) => {
    for (const gi of p.groups) {
      for (const q of paper[gi].questions) {
        out.push({ id: q.id, number: q.number, part: paper[gi].part, page: pi });
      }
    }
  });
  return out;
}

/** Chooses an option (choosing it again keeps it). */
export function select(s: ExamState, id: string, choice: number): ExamState {
  return { ...s, answers: { ...s.answers, [id]: choice }, position: id };
}

/** Clears a choice. */
export function clear(s: ExamState, id: string): ExamState {
  const answers = { ...s.answers };
  delete answers[id];
  return { ...s, answers };
}

export function toggleFlag(s: ExamState, id: string): ExamState {
  const flags = s.flags.includes(id) ? s.flags.filter((f) => f !== id) : [...s.flags, id];
  return { ...s, flags };
}

export function goto(s: ExamState, id: string): ExamState {
  return { ...s, position: id };
}

/** The first question of the next (or previous) page; the same position at either end. */
export function step(s: ExamState, qs: readonly ExamQuestion[], pages: readonly ExamPage[], dir: 1 | -1): ExamState {
  const cur = qs.find((q) => q.id === s.position);
  const page = (cur?.page ?? 0) + dir;
  if (page < 0 || page >= pages.length) {
    return s;
  }
  return { ...s, position: pages[page].questionIds[0] };
}

/** The questions not answered yet. */
export function unanswered(s: ExamState, qs: readonly ExamQuestion[]): ExamQuestion[] {
  return qs.filter((q) => !(q.id in s.answers));
}

/** The state of a cell of the question grid; current wins, then flagged, then answered. */
export function cellState(s: ExamState, id: string): CellState {
  if (s.position === id) {
    return 'current';
  }
  if (s.flags.includes(id)) {
    return 'flagged';
  }
  return id in s.answers ? 'answered' : 'empty';
}

/** The questions grouped by part, in order. */
export function byPart(qs: readonly ExamQuestion[]): { part: ToeicPart; questions: ExamQuestion[] }[] {
  const out: { part: ToeicPart; questions: ExamQuestion[] }[] = [];
  for (const q of qs) {
    const last = out.at(-1);
    if (last?.part === q.part) {
      last.questions.push(q);
    } else {
      out.push({ part: q.part, questions: [q] });
    }
  }
  return out;
}

/** What changed since the state last sent: the answers (a cleared one as -1), the flags, the position. */
export function changes(sent: ExamState, now: ExamState): SaveProgress {
  const out: SaveProgress = {};
  const answers: Record<string, number> = {};
  for (const [id, v] of Object.entries(now.answers)) {
    if (sent.answers[id] !== v) {
      answers[id] = v;
    }
  }
  for (const id of Object.keys(sent.answers)) {
    if (!(id in now.answers)) {
      answers[id] = -1;
    }
  }
  if (Object.keys(answers).length > 0) {
    out.answers = answers;
  }
  if (sent.flags.length !== now.flags.length || sent.flags.some((f) => !now.flags.includes(f))) {
    out.flags = [...now.flags];
  }
  if (sent.position !== now.position) {
    out.position = now.position;
  }
  return out;
}
