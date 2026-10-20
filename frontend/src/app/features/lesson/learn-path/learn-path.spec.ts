import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';

import { LearningPath, Milestone } from '../../../core/models/study';
import { LearnPath } from './learn-path';

const empty = (level: Milestone['level']): Milestone => ({
  level,
  active: false,
  completed: 0,
  total: 0,
  roadmaps: [],
});

const path: LearningPath = {
  topicId: 't1',
  topicName: 'Gia đình',
  streak: 7,
  milestones: [
    {
      level: 'A1',
      active: true,
      completed: 2,
      total: 6,
      roadmaps: [
        {
          topicId: 't1',
          topicName: 'Gia đình',
          topicDescription: 'Nói về các thành viên trong gia đình.',
          active: true,
          completed: 2,
          lessons: [
            { id: 'l1', title: 'My family', state: 'done' },
            { id: 'l2', title: 'My home', state: 'done' },
            { id: 'l3', title: 'At the café', state: 'current' },
            { id: 'l4', title: 'At the bank', state: 'locked' },
          ],
        },
        {
          topicId: 't2',
          topicName: 'Mua sắm',
          active: false,
          completed: 0,
          lessons: [
            { id: 's1', title: 'At the market', state: 'locked' },
            { id: 's2', title: 'Prices', state: 'locked' },
          ],
        },
      ],
    },
    empty('A2'),
    empty('B1'),
    empty('B2'),
    empty('C1'),
    empty('C2'),
  ],
};

describe('LearnPath', () => {
  let fixture: ComponentFixture<unknown>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  const setup = async (data: LearningPath, level = 'A1') => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([
          { path: 'lessons/levels/:level', component: LearnPath },
          { path: 'lessons/:id', children: [] },
        ]),
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl(`/lessons/levels/${level}`, LearnPath);
    fixture = harness.fixture;
    el = harness.routeNativeElement!;
    http.expectOne('/lessons/path').flush(data);
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the roadmaps of one part one after another, their lessons as steps', async () => {
    await setup(path);

    expect(text(el.querySelector('h1'))).toBe('Trình độ A1');
    expect(el.querySelector('a[aria-label="Quay lại trang Học"]')?.getAttribute('href')).toBe(
      '/lessons',
    );
    expect(text(el.querySelector('.summary-line'))).toBe('Tiến độ chủ đề Gia đình: 2 / 4 bài');
    expect(text(el.querySelector('.percent'))).toBe('50%');
    const bar = el.querySelector('[role="progressbar"]');
    expect(bar?.getAttribute('aria-label')).toBe('Tiến độ chủ đề Gia đình');
    expect(bar?.getAttribute('aria-valuenow')).toBe('2');

    const parts = Array.from(el.querySelectorAll('.milestone'));
    expect(parts).toHaveLength(1);
    // The part's card names its first roadmap; the next roadmap gets a title line.
    expect(text(parts[0].querySelector('.banner-label'))).toBe('Trình độ A1 • Mới bắt đầu');
    expect(text(parts[0].querySelector('.banner-title'))).toBe('Gia đình');
    expect(text(parts[0].querySelector('.banner-text'))).toBe(
      'Nói về các thành viên trong gia đình.',
    );
    expect(text(parts[0].querySelector('.banner-count'))).toBe('2 chủ đề');
    expect(Array.from(parts[0].querySelectorAll('.divider')).map((d) => text(d))).toEqual([
      'Chủ đề tiếp theo: Chủ đề 2 • Mua sắm',
    ]);

    const [family, shopping] = Array.from(parts[0].querySelectorAll('.roadmap'));
    expect(family.classList).toContain('is-active');
    const steps = Array.from(family.querySelectorAll('.node:not(.finish)'));
    expect(steps.map((s) => text(s.querySelector('.node-label')))).toEqual([
      'Bài 1',
      'Bài 2',
      'Bài 3',
      'Bài 4',
    ]);
    // Lessons done and the current one open their page; the locked one is not a link.
    expect(steps.map((s) => s.querySelector('a')?.getAttribute('href') ?? null)).toEqual([
      '/lessons/l1',
      '/lessons/l2',
      '/lessons/l3',
      null,
    ]);
    expect(steps[2].querySelector('a')?.getAttribute('aria-label')).toBe(
      'Bài 3: At the café, đang học',
    );
    expect(text(steps[2].querySelector('.cta'))).toBe('Tiếp tục');
    // Luna stands by the lesson being studied, and only there.
    expect(steps[2].classList).toContain('is-focus');
    expect(el.querySelectorAll('.luna')).toHaveLength(1);
    expect(steps[2].querySelector('.luna')).not.toBeNull();
    expect(steps[3].querySelector('[role="img"]')?.getAttribute('aria-label')).toBe(
      'Bài 4: At the bank, chưa mở',
    );
    expect(family.querySelector('.is-jump')).toBeNull();

    expect(text(family.querySelector('.finish .node-label'))).toBe('👑 Hoàn thành chủ đề 1');

    // Another roadmap: every lesson locked, a step to study it first, naming its first lesson.
    expect(shopping.querySelectorAll('.node.is-locked')).toHaveLength(2);
    const jump = shopping.querySelector('.is-jump')!;
    expect(jump.classList).not.toContain('is-focus');
    expect(text(jump.querySelector('.cta'))).toBe('Bắt đầu chủ đề 2');
    expect(text(jump.querySelector('.node-label'))).toBe('Bài 1: At the market');
  });

  it('shows a part with no lesson yet as locked', async () => {
    await setup(path, 'A2');

    expect(text(el.querySelector('h1'))).toBe('Trình độ A2');
    // The goal's roadmap is in another part: no progress here.
    expect(el.querySelector('.summary')).toBeNull();
    const part = el.querySelector('.milestone')!;
    expect(part.classList).toContain('is-empty');
    expect(text(part.querySelector('.banner-title'))).toBe('Trình độ A2');
    expect(part.querySelector('.banner-chip')?.getAttribute('aria-label')).toBe('Đang khoá');
    expect(text(part.querySelector('.no-lesson'))).toBe('Trình độ này sẽ mở khi có bài mới.');
  });

  it('makes another roadmap the goal and opens its next lesson', async () => {
    await setup(path);

    (el.querySelector('.is-jump button') as HTMLButtonElement).click();
    const req = http.expectOne('/goals');
    expect(req.request.body).toEqual({ topicId: 't2', level: 'A1' });
    req.flush({ active: {} });
    await settle();

    expect(TestBed.inject(Router).url).toBe('/lessons/s1');
  });

  it('only shows the new goal when its next lesson is for members', async () => {
    const [family, shopping] = path.milestones[0].roadmaps;
    const guestShopping = {
      ...shopping,
      completed: 1,
      lessons: [
        { ...shopping.lessons[0], state: 'done' as const },
        { ...shopping.lessons[1], state: 'membersOnly' as const },
      ],
    };
    await setup({
      ...path,
      milestones: [
        { ...path.milestones[0], roadmaps: [family, guestShopping] },
        ...path.milestones.slice(1),
      ],
    });

    (el.querySelector('.is-jump button') as HTMLButtonElement).click();
    http.expectOne('/goals').flush({ active: {} });
    await settle();
    http.expectOne('/lessons/path').flush({
      ...path,
      topicId: 't2',
      milestones: [
        {
          ...path.milestones[0],
          roadmaps: [
            { ...family, active: false },
            { ...guestShopping, active: true },
          ],
        },
        ...path.milestones.slice(1),
      ],
    });
    await settle();

    expect(TestBed.inject(Router).url).toBe('/lessons/levels/A1');
    expect(el.querySelectorAll('.roadmap')[1].classList).toContain('is-active');
    expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-label')).toBe(
      'Tiến độ chủ đề Mua sắm',
    );
  });

  it('invites to pick the first roadmap when there is no goal', async () => {
    await setup({
      ...path,
      topicId: '',
      topicName: '',
      milestones: [
        {
          ...path.milestones[0],
          active: false,
          // Without a goal no lesson is being studied.
          roadmaps: path.milestones[0].roadmaps.map((r) => ({
            ...r,
            active: false,
            lessons: r.lessons.map((l) => ({
              ...l,
              state: l.state === 'current' ? ('locked' as const) : l.state,
            })),
          })),
        },
      ],
    });

    const jumps = Array.from(el.querySelectorAll('.is-jump'));
    expect(jumps).toHaveLength(2);
    // The first one is the big green step with Luna; the next ones are small blue stars.
    expect(jumps[0].classList).toContain('is-focus');
    expect(text(jumps[0].querySelector('.cta'))).toBe('Học chủ đề này?');
    expect(text(jumps[0].querySelector('.node-label'))).toBe('Bắt đầu ngay');
    expect(jumps[0].querySelector('.luna')).not.toBeNull();
    expect(jumps[1].classList).not.toContain('is-focus');
    expect(text(el.querySelector('.summary-line'))).toBe('Tiến độ cả trình độ A1: 2 / 6 bài');
  });

  it('opens the guide under the banner', async () => {
    await setup(path);

    const button = el.querySelector('.guide-btn') as HTMLButtonElement;
    const guide = el.querySelector('#path-guide') as HTMLElement;
    expect(button.getAttribute('aria-expanded')).toBe('false');
    expect(guide.hidden).toBe(true);
    button.click();
    await settle();
    expect(button.getAttribute('aria-expanded')).toBe('true');
    expect(guide.hidden).toBe(false);
  });

  it('tells guests the next lessons are for members', async () => {
    const [family] = path.milestones[0].roadmaps;
    await setup({
      ...path,
      milestones: [
        {
          ...path.milestones[0],
          roadmaps: [
            {
              ...family,
              completed: 1,
              lessons: [
                { id: 'l1', title: 'My family', state: 'done' },
                { id: 'l2', title: 'My home', state: 'membersOnly' },
              ],
            },
          ],
        },
      ],
    });

    expect(text(el.querySelector('.members-banner'))).toContain('chỉ dành cho thành viên');
    expect(
      el
        .querySelectorAll('.node:not(.finish)')[1]
        .querySelector('[role="img"]')
        ?.getAttribute('aria-label'),
    ).toBe('Bài 2: My home, chỉ dành cho thành viên');
  });
});
