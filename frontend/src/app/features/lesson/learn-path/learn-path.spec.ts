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

  it('links a finished topic to its summary', async () => {
    const done = structuredClone(path);
    done.milestones[0].roadmaps[1] = {
      ...done.milestones[0].roadmaps[1],
      completed: 2,
      lessons: [
        { id: 's1', title: 'At the market', state: 'done' },
        { id: 's2', title: 'Prices', state: 'done' },
      ],
    };
    await setup(done);
    const links = Array.from(el.querySelectorAll<HTMLAnchorElement>('.finish a'));
    expect(links.map((a) => a.getAttribute('href'))).toEqual(['/topics/t2/summary?level=A1']);
    expect(links[0].getAttribute('aria-label')).toBe('Xem tổng kết chủ đề 2');
  });

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
  describe('with a course', () => {
    const coursePath = (
      familyLessons: LearningPath['milestones'][0]['roadmaps'][0]['lessons'],
      withNext = true,
    ): LearningPath => ({
      ...path,
      milestones: [
        {
          ...path.milestones[0],
          roadmaps: [
            {
              topicId: 't1',
              topicName: 'Gia đình',
              topicDescription: 'Nói về các thành viên trong gia đình.',
              course: {
                number: 2,
                canDoVi: 'giới thiệu gia đình.',
                grammarVi: ['have / has', "sở hữu 's"],
              },
              active: true,
              completed: familyLessons.filter((l) => l.state === 'done').length,
              lessons: familyLessons,
            },
            ...(withNext
              ? [
                  {
                    topicId: 't3',
                    topicName: 'Thời gian và ngày tháng',
                    course: { number: 3, canDoVi: 'nói giờ và ngày.', grammarVi: ['at / on / in'] },
                    active: false,
                    completed: 0,
                    lessons: [
                      { id: 'd1', title: 'What Time Is It?', state: 'locked' as const },
                      { id: 'd2', title: 'My Birthday', state: 'locked' as const },
                    ],
                  },
                ]
              : []),
          ],
        },
        ...path.milestones.slice(1),
      ],
    });
    const mainsDone = [
      { id: 'l1', title: 'My Family', state: 'done' as const },
      { id: 'l2', title: "My Brother's Room", state: 'done' as const },
      { id: 'l3', title: 'A Family Photo', state: 'done' as const },
      { id: 'x1', title: "Grandma's Birthday", state: 'current' as const, extra: true },
      { id: 'x2', title: 'A Day at My Uncle’s Farm', state: 'locked' as const, extra: true },
    ];

    it('names the topic by its course number, with its grammar and what it leads to', async () => {
      await setup(coursePath(mainsDone));

      expect(text(el.querySelector('.banner-title'))).toBe('Chủ đề 2 · Gia đình');
      expect(text(el.querySelector('.banner-text'))).toBe(
        'Sau chủ đề này bạn có thể giới thiệu gia đình.',
      );
      const [family, time] = Array.from(el.querySelectorAll('.roadmap'));
      expect(text(family.querySelector('.topic-title'))).toBe('Chủ đề 2 • Gia đình');
      expect(Array.from(family.querySelectorAll('.grammar-chip')).map((c) => text(c))).toEqual([
        'have / has',
        "sở hữu 's",
      ]);
      expect(text(family.querySelector('.grammar-label'))).toBe('Ngữ pháp:');
      // The banner already says what the featured topic leads to.
      expect(family.querySelector('.can-do')).toBeNull();
      expect(text(family.querySelector('.finish .node-label'))).toBe('👑 Hoàn thành chủ đề 2');

      expect(Array.from(el.querySelectorAll('.divider')).map((d) => text(d))).toEqual([
        'Chủ đề tiếp theo: Chủ đề 3 • Thời gian và ngày tháng',
      ]);
      expect(time.querySelector('.topic-title')).toBeNull();
      expect(text(time.querySelector('.can-do'))).toBe(
        'Sau chủ đề này bạn có thể nói giờ và ngày.',
      );
      expect(text(time.querySelector('.is-jump .cta'))).toBe('Bắt đầu chủ đề 3');
      // Progress counts the extra lessons too.
      expect(text(el.querySelector('.summary-line'))).toBe('Tiến độ chủ đề Gia đình: 3 / 5 bài');
    });

    it('tags the extra lessons, after a label', async () => {
      await setup(coursePath(mainsDone));

      const family = el.querySelector('.roadmap')!;
      const steps = Array.from(family.querySelectorAll('.node:not(.finish)'));
      expect(steps.map((s) => s.classList.contains('is-extra'))).toEqual([
        false,
        false,
        false,
        true,
        true,
      ]);
      expect(steps.map((s) => text(s.querySelector('.extra-tag')))).toEqual([
        '',
        '',
        '',
        '· Bài thêm',
        '· Bài thêm',
      ]);
      expect(text(steps[3].querySelector('.node-label'))).toBe('Bài 4 · Bài thêm');
      expect(steps[3].querySelector('a')?.getAttribute('aria-label')).toBe(
        "Bài 4 (bài thêm): Grandma's Birthday, đang học",
      );
      expect(steps[4].querySelector('[role="img"]')?.getAttribute('title')).toBe(
        'Bài thêm: A Day at My Uncle’s Farm',
      );
      const labels = family.querySelectorAll('.extra-label');
      expect(labels).toHaveLength(1);
      expect(text(labels[0])).toBe('Bài thêm · gặp lại từ và ngữ pháp');
      // The label sits between the last main lesson and the first extra one.
      const top = (n: Element) => parseFloat((n as HTMLElement).style.top);
      expect(top(labels[0])).toBeGreaterThan(top(steps[2]));
      expect(top(labels[0])).toBeLessThan(top(steps[3]));
    });

    it('offers the next topic once the main lessons are done, and starts it', async () => {
      await setup(coursePath(mainsDone));

      const card = el.querySelector('lu-move-on')!;
      expect(card.closest('.roadmap')?.classList).toContain('is-active');
      expect(card.querySelector('a')?.getAttribute('href')).toBe('/lessons/x1');
      const button = card.querySelector('button')!;
      expect(text(button)).toBe('Học xong 3 bài chính — sang Chủ đề 3: Thời gian và ngày tháng');

      button.click();
      const req = http.expectOne('/goals');
      expect(req.request.body).toEqual({ topicId: 't3', level: 'A1' });
      req.flush({ active: {} });
      await settle();
      expect(TestBed.inject(Router).url).toBe('/lessons/d1');
    });

    it('does not offer the next topic while a main lesson is left', async () => {
      await setup(
        coursePath([
          { id: 'l1', title: 'My Family', state: 'done' },
          { id: 'l2', title: "My Brother's Room", state: 'current' },
          { id: 'l3', title: 'A Family Photo', state: 'locked' },
          { id: 'x1', title: "Grandma's Birthday", state: 'locked', extra: true },
        ]),
      );
      expect(el.querySelector('lu-move-on')).toBeNull();
      expect(el.querySelectorAll('.extra-label')).toHaveLength(1);
    });

    it('does not offer a next topic when there is none', async () => {
      await setup(coursePath(mainsDone, false));
      expect(el.querySelector('lu-move-on')).toBeNull();
    });
  });

  it('keeps the description and plain lessons without a course', async () => {
    await setup(path);
    expect(el.querySelector('lu-topic-info')).toBeNull();
    expect(el.querySelector('.extra-label')).toBeNull();
    expect(el.querySelector('.is-extra')).toBeNull();
    expect(el.querySelector('lu-move-on')).toBeNull();
  });
});
