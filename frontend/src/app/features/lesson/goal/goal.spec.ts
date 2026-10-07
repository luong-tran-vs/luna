import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Dashboard } from '../../../core/models/dashboard';
import { GoalView, Goals, MyLessons, PublicTopic } from '../../../core/models/study';
import { Goal } from './goal';

const goal = (over: Partial<GoalView> = {}): GoalView => ({
  topicId: 't1', topicName: 'Gia đình', level: 'A1', completedLessons: 2, totalLessons: 12, status: 'active',
  effectiveFrom: '2026-09-30', ...over,
});

const dashboard = (over: Partial<Dashboard> = {}): Dashboard => ({
  kind: 'studying', goal: goal(), goalCompleted: false, skills: null,
  lesson: { id: 'l2', title: 'My family', topicName: 'Gia đình', level: 'A1' },
  steps: { read: 'done', listen: 'current', write: 'locked' }, currentStep: 'listen',
  action: { kind: 'continue', step: 'listen' }, streak: 1, tomorrowCards: 0, ...over,
});

const studyingMine: MyLessons = { current: { id: 'l2', title: 'My family' }, completed: [], upcoming: [] };

const a1Topics: PublicTopic[] = [
  { id: 't1', name: 'Gia đình', level: 'A1', description: 'Người thân', lessonCount: 12 },
  { id: 't2', name: 'Mua sắm', level: 'A1', description: '', lessonCount: 5 },
  { id: 't3', name: 'Du lịch', level: 'A1', description: '', lessonCount: 0 },
];

describe('Goal', () => {
  let fixture: ComponentFixture<Goal>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) => Array.from(el.querySelectorAll('button')).find((b) => text(b) === label);
  const buttonStarting = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => text(b).startsWith(label));
  const topicRow = (name: string) =>
    Array.from(el.querySelectorAll('.topic')).find((t) => text(t.querySelector('.topic-name')) === name)!;

  const setup = async (
    goals: Goals,
    query: Record<string, string> = {},
    home: Dashboard = dashboard(),
    mine: MyLessons = studyingMine,
  ) => {
    TestBed.configureTestingModule({
      imports: [Goal],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { queryParamMap: convertToParamMap(query) } } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture = TestBed.createComponent(Goal);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/dashboard').flush(home);
    http.expectOne('/api/lessons/mine').flush(mine);
    http.expectOne('/api/goals').flush(goals);
    await settle();
  };
  const chooseLevel = async (level: string) => {
    buttonStarting(level)!.click();
    await settle();
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === level).flush({ topics: level === 'A1' ? a1Topics : [] });
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the course being studied with a way into the lesson being studied', async () => {
    await setup({ active: goal(), others: [] });
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === 'A1').flush({ topics: a1Topics });
    await settle();
    const current = el.querySelector('.current')!;
    expect(text(current.querySelector('.current-name'))).toBe('A1 · Gia đình');
    expect(text(current)).toContain('Đã học 2/12 bài');
    expect(current.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('2');
    expect(current.querySelector('a[href="/lessons/l2"]')?.textContent?.trim()).toBe('Tiếp tục học');
    expect(current.querySelector('a[href="/lessons"]')?.textContent?.trim()).toBe('Danh sách bài');
  });

  const openCurrent = async (home: Dashboard, mine: MyLessons = studyingMine) => {
    await setup({ active: goal(), others: [] }, {}, home, mine);
    http.expectOne((r) => r.url === '/api/topics').flush({ topics: a1Topics });
    await settle();
    return el.querySelector('.current')!;
  };

  it('says "Bắt đầu học" before any step of the lesson is done', async () => {
    const current = await openCurrent(dashboard({ action: { kind: 'start', step: 'read' } }));
    expect(current.querySelector('a[href="/lessons/l2"]')?.textContent?.trim()).toBe('Bắt đầu học');
  });

  it('offers no lesson to continue when the course has no new lesson', async () => {
    const current = await openCurrent(dashboard({ kind: 'noNewLesson', lesson: null, action: null }), {
      current: null,
      completed: [],
      upcoming: [],
    });
    expect(current.querySelector('.btn-primary')).toBeNull();
    expect(text(current.querySelector('[role="status"]'))).toContain('Khóa học này chưa có bài mới');
    expect(current.querySelector('a[href="/lessons"]')).not.toBeNull();
  });

  it('opens the lesson finished last when the course has no new lesson', async () => {
    const current = await openCurrent(dashboard({ kind: 'noNewLesson', lesson: null, action: null }), {
      current: null,
      completed: [
        { id: 'l7', title: 'Newest', topicName: 'Gia đình', completedAt: '2026-10-01T10:00:00Z' },
        { id: 'l6', title: 'Older', topicName: 'Gia đình', completedAt: '2026-09-30T10:00:00Z' },
      ],
      upcoming: [],
    });
    const link = current.querySelector('a.btn-primary')!;
    expect(link.getAttribute('href')).toBe('/lessons/l7');
    expect(link.textContent?.trim()).toBe('Xem lại bài gần nhất');
  });

  it('has no current course card before a goal is chosen', async () => {
    await setup({ active: null, others: [] });
    expect(el.querySelector('.current')).toBeNull();
  });

  it('asks for a level first', async () => {
    await setup({ active: null, others: [] });
    const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(radios.map((r) => text(r.querySelector('.level-code')))).toEqual(['A1', 'A2', 'B1', 'B2', 'C1', 'C2']);
    expect(text(radios[0].querySelector('.level-name'))).toBe('Mới bắt đầu');
    expect(el.querySelector('.topic')).toBeNull();
  });

  it('lists the topics of the level with lesson counts and progress', async () => {
    await setup({ active: goal(), others: [goal({ topicId: 't2', topicName: 'Mua sắm', completedLessons: 1, totalLessons: 5, status: 'paused' })] });
    // The current level is preselected.
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === 'A1').flush({ topics: a1Topics });
    await settle();
    expect(text(topicRow('Gia đình'))).toContain('12 bài');
    expect(text(topicRow('Gia đình'))).toContain('Đã học 2/12');
    expect(text(topicRow('Gia đình'))).toContain('Đang học');
    expect(text(topicRow('Mua sắm'))).toContain('Đã học 1/5');
    expect(text(topicRow('Du lịch'))).toContain('Chưa có bài');
  });

  it('does not let the learner pick a topic that has no lesson yet', async () => {
    await setup({ active: null, others: [] });
    await chooseLevel('A1');
    expect(topicRow('Du lịch').querySelector<HTMLButtonElement>('button')!.disabled).toBe(true);
    expect(topicRow('Mua sắm').querySelector<HTMLButtonElement>('button')!.disabled).toBe(false);
  });

  it('sets the goal and goes to its lessons at once', async () => {
    await setup({ active: null, others: [] });
    await chooseLevel('A1');
    topicRow('Mua sắm').querySelector<HTMLButtonElement>('button')!.click();
    await settle();
    const req = http.expectOne('/api/goals');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ topicId: 't2', level: 'A1' });
    req.flush({ active: goal({ topicId: 't2' }) });
    await settle();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/lessons');
  });

  it('congratulates at the end of a roadmap', async () => {
    await setup({ active: goal({ completedLessons: 12 }), others: [] }, { completed: '1' });
    expect(text(el.querySelector('.congrats'))).toContain('Chúc mừng');
    expect(text(el.querySelector('.congrats'))).toContain('A1 · Gia đình');
    button('Lên trình độ A2')!.click();
    await settle();
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === 'A2').flush({ topics: [] });
    await settle();
    expect(el.querySelector('.congrats')).toBeNull();
  });

  it('offers other topics of the same level after congratulations', async () => {
    await setup({ active: goal({ completedLessons: 12 }), others: [] }, { completed: '1' });
    button('Chọn chủ đề khác cùng trình độ')!.click();
    await settle();
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === 'A1').flush({ topics: a1Topics });
    await settle();
    expect(topicRow('Mua sắm')).toBeTruthy();
  });
});
