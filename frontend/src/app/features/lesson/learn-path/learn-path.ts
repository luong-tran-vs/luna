import { NgOptimizedImage, NgTemplateOutlet } from '@angular/common';
import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { Level, LEVEL_NAMES, LEVELS } from '../../../core/models/lesson';
import { LearningPath, Milestone, PathLesson, PathRoadmap } from '../../../core/models/study';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { MoveOn, MoveOnCard } from './move-on/move-on';
import { PathBanner } from './path-banner/path-banner';
import { TopicInfo } from './topic-info/topic-info';

/**
 * Width of the path's drawing, in px. The steps are placed in % of it, so the path narrows with
 * the screen; on wide screens it keeps this width, like the design.
 */
const PATH_WIDTH = 480;
/** Vertical distance between two steps, in px. */
const ROW = 150;
/** Extra room before the step with Luna and the bubble over it. */
const FOCUS_ROOM = 40;
/** Extra room before the first extra lesson, for the "Bài thêm" label above it. */
const EXTRA_ROOM = 40;
/** Distance from the last main lesson down to the "Bài thêm" label. */
const EXTRA_LABEL = 72;
/** Room above the first step: more when a bubble sits over it. */
const TOP = 64;
const TOP_BUBBLE = 104;
/** Room under the finish for its label and the way on to the next roadmap. */
const BOTTOM = 84;
/** Offset of the steps from the middle, in px, repeated: the path winds left and right. */
const OFFSETS = [0, 90, -90, 70, -70];
const MIDDLE = PATH_WIDTH / 2;

interface Point {
  x: number;
  y: number;
}

interface PathNode extends Point {
  lesson: PathLesson;
  /** 1-based number of the lesson in the roadmap. */
  n: number;
  /** The lesson being studied: the big green step with Luna beside it. */
  focus: boolean;
  /** An extra lesson of a course topic: a lighter step tagged "Bài thêm". */
  extra: boolean;
}

interface StartStep extends Point {
  /** The first roadmap to pick when no lesson is being studied here: green, with Luna. */
  focus: boolean;
  /** Label under a blue start: the lesson it opens. */
  next: string;
}

interface RoadmapView {
  roadmap: PathRoadmap;
  /** Number of the topic: its place in the level's course, else in the part (1-based). */
  number: number;
  /** Where the "Bài thêm" label sits, before the first extra lesson; null without extras. */
  extraLabel: Point | null;
  moveOn: MoveOn | null;
  /** The "Học chủ đề này?" step before the lessons of a roadmap that is not the goal. */
  start: StartStep | null;
  nodes: PathNode[];
  /** The trophy after the last lesson; done once every lesson is. */
  finish: Point & { done: boolean };
  height: number;
  /** SVG path data: the whole road, the way already open (solid) and the locked part (dashed). */
  road: string;
  open: string;
  locked: string;
}

interface MilestoneView {
  milestone: Milestone;
  /** 1-based number of the part, in level order. */
  number: number;
  levelName: string;
  roadmaps: RoadmapView[];
}

/** A curve from one step to the next, in px. */
function segment(a: Point, b: Point): string {
  const mid = (a.y + b.y) / 2;
  return `M${a.x} ${a.y}C${a.x} ${mid} ${b.x} ${mid} ${b.x} ${b.y}`;
}

function toRoadmapView(
  r: PathRoadmap,
  index: number,
  count: number,
  focusStart: boolean,
): RoadmapView {
  const done = r.lessons.length > 0 && r.completed === r.lessons.length;
  const jump = !r.active && !done && r.lessons.length > 0;
  const first = jump ? 1 : 0;
  const top = jump || r.lessons[0]?.state === 'current' ? TOP_BUBBLE : TOP;
  let y = top - ROW;
  // The first step and the finish stay in the middle, where the way from the previous roadmap
  // comes in and the way to the next one goes on.
  const at = (i: number, room = 0) => {
    y += i === 0 ? ROW : ROW + room;
    return { x: MIDDLE + OFFSETS[i % OFFSETS.length], y };
  };
  // The lesson a start opens: the first not done.
  const k = Math.max(
    r.lessons.findIndex((l) => l.state !== 'done'),
    0,
  );
  const start = jump
    ? { ...at(0), focus: focusStart, next: `Bài ${k + 1}: ${r.lessons[k].title}` }
    : null;
  let extraLabel: Point | null = null;
  const nodes = r.lessons.map((lesson, i) => {
    const focus = lesson.state === 'current';
    const extra = !!lesson.extra;
    // The first extra lesson after the main ones comes a little lower, under its label.
    const firstExtra = extra && i > 0 && !r.lessons[i - 1].extra;
    if (firstExtra) {
      extraLabel = { x: MIDDLE, y: y + EXTRA_LABEL };
    }
    const room = (focus ? FOCUS_ROOM : 0) + (firstExtra ? EXTRA_ROOM : 0);
    return { lesson, n: i + 1, focus, extra, ...at(i + first, room) };
  });
  y += ROW;
  const finish = { x: MIDDLE, y, done };
  const points: Point[] = [...(start ? [start] : []), ...nodes, finish];
  const height = finish.y + BOTTOM;

  let open = '';
  let locked = '';
  for (let i = 1; i < points.length; i++) {
    const next = nodes[i - first]?.lesson.state;
    // The way to a lesson is open once the learner can study it; the way to the finish once all are done.
    const isOpen = next ? next === 'done' || next === 'current' : done;
    if (isOpen) {
      open += segment(points[i - 1], points[i]);
    } else {
      locked += segment(points[i - 1], points[i]);
    }
  }
  // The way goes on through the title of the next roadmap.
  if (index > 0) {
    locked = `M${MIDDLE} 0L${points[0].x} ${points[0].y}` + locked;
  }
  if (index < count - 1) {
    locked += `M${finish.x} ${finish.y}L${MIDDLE} ${height}`;
  }
  return {
    roadmap: r,
    number: r.course?.number || index + 1,
    extraLabel,
    moveOn: null,
    start,
    nodes,
    finish,
    height,
    road: open + locked,
    open,
    locked,
  };
}

function toView(m: Milestone): MilestoneView {
  // Luna stands by the lesson being studied; with none here, by the first roadmap to pick.
  const studying = m.roadmaps.some((r) => r.lessons.some((l) => l.state === 'current'));
  let focusLeft = !studying;
  const roadmaps = m.roadmaps.map((r, i) => {
    const v = toRoadmapView(r, i, m.roadmaps.length, focusLeft);
    if (v.start) {
      focusLeft = false;
    }
    return v;
  });
  // Main lessons done and an extra one being studied: the next topic may start now.
  roadmaps.forEach((v, i) => {
    const next = roadmaps[i + 1];
    const lessons = v.roadmap.lessons;
    const current = lessons.find((l) => l.state === 'current');
    const mains = lessons.filter((l) => !l.extra);
    if (
      next &&
      v.roadmap.active &&
      current?.extra &&
      mains.length > 0 &&
      mains.every((l) => l.state === 'done')
    ) {
      v.moveOn = { roadmap: next.roadmap, number: next.number, mains: mains.length, current };
    }
  });
  return {
    milestone: m,
    number: LEVELS.indexOf(m.level) + 1,
    levelName: LEVEL_NAMES[m.level],
    roadmaps,
  };
}

/**
 * One part of the "Học" tab (design/lo-trinh.png), opened from the list of parts: a banner with the
 * part's progress, then every topic's roadmap at that level on one winding road, with the next
 * roadmap's title between two of them. Lessons done reopen freely, the current one is the big green
 * step with Luna, the rest stay locked; another roadmap is studied by making it the goal
 * ("Học chủ đề này?").
 */
@Component({
  selector: 'lu-learn-path',
  imports: [
    Loading,
    Icon,
    RouterLink,
    NgOptimizedImage,
    NgTemplateOutlet,
    PathBanner,
    TopicInfo,
    MoveOnCard,
  ],
  templateUrl: './learn-path.html',
  styleUrl: './learn-path.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LearnPath {
  private readonly api = inject(StudyApiService);
  private readonly router = inject(Router);
  private readonly injector = inject(Injector);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  protected readonly pathWidth = PATH_WIDTH;
  protected readonly middle = MIDDLE;
  protected readonly data = signal<LearningPath | null>(null);
  protected readonly error = signal(false);
  protected readonly pending = signal(false);
  protected readonly switchError = signal<string | null>(null);

  /** The level of the part shown (route `levels/:level`). */
  protected readonly level = inject(ActivatedRoute).snapshot.paramMap.get('level') ?? '';
  protected readonly part = computed(() => {
    const m = this.data()?.milestones.find((x) => x.level === this.level);
    return m ? toView(m) : null;
  });
  protected readonly title = computed(() => {
    const p = this.part();
    return p ? `Trình độ ${p.milestone.level}` : 'Học';
  });
  /** The goal's roadmap when it belongs to this part, for the progress at the top. */
  protected readonly active = computed(
    () => this.part()?.milestone.roadmaps.find((r) => r.active) ?? null,
  );
  /** The roadmap named in the banner: the one being studied, else the part's first. */
  protected readonly featured = computed(
    () => this.active() ?? this.part()?.milestone.roadmaps[0] ?? null,
  );
  protected readonly bannerTitle = computed(() => {
    const f = this.featured();
    if (!f) {
      return `Trình độ ${this.part()?.milestone.level ?? ''}`;
    }
    return f.course ? `Chủ đề ${f.course.number} · ${f.topicName}` : f.topicName;
  });
  protected readonly bannerText = computed(() => {
    const f = this.featured();
    if (!f) {
      return 'Trình độ này sẽ mở khi có bài mới.';
    }
    const canDo = f.course?.canDoVi.trim();
    if (canDo) {
      return `Sau chủ đề này bạn có thể ${canDo}`;
    }
    return f.topicDescription?.trim() || `Học các bài của chủ đề ${f.topicName} từng bước một.`;
  });
  /** Progress in the banner: the goal's roadmap when it is in this part, else the whole part. */
  protected readonly progress = computed(() => {
    const a = this.active();
    const m = this.part()?.milestone;
    const p = a
      ? { label: `chủ đề ${a.topicName}`, done: a.completed, total: a.lessons.length }
      : m?.total
        ? { label: `cả trình độ ${m.level}`, done: m.completed, total: m.total }
        : null;
    return p ? { ...p, percent: p.total ? Math.round((p.done / p.total) * 100) : 0 } : null;
  });
  /** A guest: the lessons after the first of each roadmap are for members only. */
  protected readonly membersOnly = computed(() =>
    (this.part()?.milestone.roadmaps ?? []).some((r) =>
      r.lessons.some((l) => l.state === 'membersOnly'),
    ),
  );

  constructor() {
    this.load();
  }

  protected lessonLabel(node: PathNode): string {
    const state = {
      done: 'đã học',
      current: 'đang học',
      locked: 'chưa mở',
      membersOnly: 'chỉ dành cho thành viên',
    };
    const extra = node.extra ? ' (bài thêm)' : '';
    return `Bài ${node.n}${extra}: ${node.lesson.title}, ${state[node.lesson.state]}`;
  }

  /** Luna stands on the side of the step with more room. */
  protected lunaLeft(x: number): boolean {
    return x > MIDDLE;
  }

  /**
   * Makes another roadmap the goal and opens its next lesson (the first not done) at once. A guest
   * who already did the first lesson cannot study the next ones: the path just shows the new goal.
   */
  protected async study(r: PathRoadmap, level: Level): Promise<void> {
    this.pending.set(true);
    this.switchError.set(null);
    try {
      await firstValueFrom(this.api.setGoal(r.topicId, level));
      const next = r.lessons.find((l) => l.state !== 'done');
      if (next && next.state !== 'membersOnly') {
        await this.router.navigate(['/lessons', next.id]);
      } else {
        this.load();
      }
    } catch {
      this.switchError.set('Không chuyển được chủ đề, vui lòng thử lại.');
    } finally {
      this.pending.set(false);
    }
  }

  private load(): void {
    this.api.path().subscribe({
      next: (p) => {
        this.data.set(p);
        if (this.active()) {
          afterNextRender(
            () => {
              // Back where the learner stands: the lesson being studied, else the goal's roadmap.
              const host = this.host.nativeElement;
              const target =
                host.querySelector('.roadmap.is-active .is-current') ??
                host.querySelector('.roadmap.is-active');
              target?.scrollIntoView?.({ block: 'center' });
            },
            { injector: this.injector },
          );
        }
      },
      error: () => this.error.set(true),
    });
  }
}
