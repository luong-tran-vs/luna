import {
  GroupAdmin,
  TestAdmin,
  TestCheck,
  ToeicPart,
  ToeicStatus,
} from '../../../core/models/toeic';
import { kindLabel } from '../../toeic/shared/score.logic';

/** Pure helpers shared by the TOEIC admin pages (home, Kho cụm câu, a test). */

export const STATUS_LABELS: Record<ToeicStatus, string> = {
  draft: 'Nháp',
  published: 'Đã đăng',
  hidden: 'Ẩn',
};

/** The "Thuộc đề" filter value for the groups in no test ("" is every group, a key is one test). */
export const NO_TEST = 'none';

/** The groups that pass the "Thuộc đề" filter. */
export function byTest(groups: GroupAdmin[], test: string): GroupAdmin[] {
  if (!test) {
    return groups;
  }
  return groups.filter((g) => (test === NO_TEST ? g.tests.length === 0 : g.tests.includes(test)));
}

/** Why a test cannot be published yet, in Vietnamese. */
export function reasonsOf(c: TestCheck): string[] {
  const out: string[] = [];
  for (const p of c.parts) {
    if (p.have > 0 && p.have !== p.need) {
      out.push(`Part ${p.part}: có ${p.have}/${p.need} câu`);
    } else if (p.have === 0 && p.part >= 5) {
      out.push(`Part ${p.part}: chưa có câu`);
    }
  }
  if (c.missingGroups.length) {
    out.push(`Cụm không còn: ${c.missingGroups.join(', ')}`);
  }
  if (c.draftGroups.length) {
    out.push(`${c.draftGroups.length} cụm chưa đăng`);
  }
  if (c.missingPhotos.length) {
    out.push(`Thiếu ảnh Part 1 ở câu ${c.missingPhotos.map((m) => m.number).join(', ')}`);
  }
  return out;
}

/** How full one part of a test is: all its questions, some, or none. */
export type PartState = 'ok' | 'some' | 'none';

/** One test as a card on the TOEIC home page. */
export interface TestCard {
  key: string;
  title: string;
  kind: string;
  order: number;
  status: ToeicStatus;
  groups: number;
  have: number;
  need: number;
  percent: number;
  parts: { part: ToeicPart; state: PartState; label: string }[];
  /** Why it cannot be published yet; empty when it is published. */
  reasons: string[];
  ready: boolean;
}

/** The card of a test: the questions it has out of what its kind needs, and each part's state. */
export function testCard(t: TestAdmin): TestCard {
  const c = t.check;
  const have = c.parts.reduce((n, p) => n + p.have, 0);
  const need = c.parts.reduce((n, p) => n + p.need, 0);
  return {
    key: t.key,
    title: t.title,
    kind: kindLabel(t.kind),
    order: t.order,
    status: t.status,
    groups: t.groups.length,
    have,
    need,
    percent: need > 0 ? Math.min(100, Math.round((have * 100) / need)) : 0,
    parts: c.parts.map((p) => ({
      part: p.part,
      state: p.have === 0 ? 'none' : p.have === p.need ? 'ok' : 'some',
      label: `Part ${p.part}: ${p.have}/${p.need} câu`,
    })),
    reasons: t.status === 'published' ? [] : reasonsOf(c),
    ready: c.ready,
  };
}

/** The group bank in numbers, for the home page. */
export interface BankSummary {
  total: number;
  published: number;
  draft: number;
  hidden: number;
  /** Groups in no test yet. */
  free: number;
}

export function bankSummary(groups: GroupAdmin[]): BankSummary {
  return {
    total: groups.length,
    published: groups.filter((g) => g.status === 'published').length,
    draft: groups.filter((g) => g.status === 'draft').length,
    hidden: groups.filter((g) => g.status === 'hidden').length,
    free: groups.filter((g) => g.tests.length === 0).length,
  };
}

/** The "Nguồn" filter: every group, the groups the AI wrote, or the groups imported from a file. */
export type SourceFilter = '' | 'ai' | 'file';

/** The groups that pass the "Nguồn" filter. */
export function bySource(groups: GroupAdmin[], source: SourceFilter): GroupAdmin[] {
  if (!source) {
    return groups;
  }
  return groups.filter((g) => (source === 'ai' ? g.source === 'ai' : g.source !== 'ai'));
}

/** The target scores a group is written for. */
export const TOEIC_LEVELS = [450, 600, 750, 900] as const;

/** The groups of one level ("Mức" filter); null keeps them all. */
export function atLevel(groups: GroupAdmin[], level: number | null): GroupAdmin[] {
  return level === null ? groups : groups.filter((g) => g.level === level);
}

/** The order of the bank's list: as the server gives it (part, then key), or by level (450 → 900 or 900 → 450). */
export type LevelSort = '' | 'asc' | 'desc';

/** The groups in the "Sắp xếp" order; groups of the same level keep the server's order. */
export function byLevel(groups: GroupAdmin[], sort: LevelSort): GroupAdmin[] {
  if (!sort) {
    return groups;
  }
  const sign = sort === 'asc' ? 1 : -1;
  return [...groups].sort((a, b) => sign * (a.level - b.level));
}

/** How many groups one page of the bank's list can show ("Mỗi trang"); 30 unless the admin picks another. */
export const GROUPS_PAGE_SIZES = [10, 30, 50, 100] as const;
export const GROUPS_PAGE_SIZE = 30;

/** What one "Nhờ AI soạn" request writes in each part (the server's plan: about 10 questions). */
export const DRAFT_PLAN: Record<ToeicPart, string> = {
  1: '10 câu (10 ảnh)',
  2: '10 câu',
  3: '4 đoạn hội thoại · 12 câu',
  4: '4 bài nói · 12 câu',
  5: '10 câu',
  6: '3 văn bản · 12 câu',
  7: '10 câu (3–5 cụm, có cụm nhiều văn bản)',
};
