import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Dashboard } from '../../../core/models/dashboard';
import { GoalView, Goals, PublicTopic } from '../../../core/models/study';
import { Goal } from './goal';

const goal = (over: Partial<GoalView> = {}): GoalView => ({
  topicId: 't1', topicName: 'Gia đình', level: 'A1', completedLessons: 2, totalLessons: 12, status: 'active',
  effectiveFrom: '2026-09-30', ...over,
});

const today = (over: Partial<Dashboard> = {}): Dashboard => ({
  kind: 'studying', goal: goal(), goalCompleted: false, skills: null, lesson: null,
  steps: { review: 'done', read: 'current', listen: 'locked', write: 'locked' }, currentStep: 'read',
  action: { kind: 'continue', step: 'read' }, streak: 1, tomorrowCards: 0, ...over,
});

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

  const setup = async (goals: Goals, query: Record<string, string> = {}, dashboard: Dashboard = today()) => {
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
    http.expectOne('/api/dashboard').flush(dashboard);
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

  it('shows the course being studied with a way into today’s lesson', async () => {
    await setup({ active: goal(), others: [] });
    http.expectOne((r) => r.url === '/api/topics' && r.params.get('level') === 'A1').flush({ topics: a1Topics });
    await settle();
    const current = el.querySelector('.current')!;
    expect(text(current.querySelector('.current-name'))).toBe('A1 · Gia đình');
    expect(text(current)).toContain('Đã học 2/12 bài');
    expect(current.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('2');
    expect(current.querySelector('a[href="/today"]')?.textContent?.trim()).toBe('Tiếp tục học');
    expect(current.querySelector('a[href="/lessons"]')?.textContent?.trim()).toBe('Danh sách bài');
  });

  const openCurrent = async (dashboard: Dashboard) => {
    await setup({ active: goal(), others: [] }, {}, dashboard);
    http.expectOne((r) => r.url === '/api/topics').flush({ topics: a1Topics });
    await settle();
    return el.querySelector('.current')!;
  };

  it('says "Bắt đầu học" before any step of today’s lesson is done', async () => {
    const current = await openCurrent(today({ action: { kind: 'start', step: 'review' } }));
    expect(current.querySelector('a[href="/today"]')?.textContent?.trim()).toBe('Bắt đầu học');
  });

  it('offers no lesson to continue when the course has no new lesson', async () => {
    const current = await openCurrent(today({ kind: 'noNewLesson', action: null }));
    expect(current.querySelector('a[href="/today"]')).toBeNull();
    expect(text(current.querySelector('[role="status"]'))).toContain('Khóa học này chưa có bài mới');
    expect(current.querySelector('a[href="/lessons"]')).not.toBeNull();
  });

  it('says see you tomorrow when today’s lesson is done', async () => {
    const current = await openCurrent(today({ kind: 'doneToday', action: null }));
    expect(current.querySelector('a[href="/today"]')).toBeNull();
    expect(text(current.querySelector('[role="status"]'))).toBe('Đã xong bài hôm nay, hẹn bạn ngày mai.');
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

  it('sets the goal and goes to today', async () => {
    await setup({ active: null, others: [] });
    await chooseLevel('A1');
    topicRow('Mua sắm').querySelector<HTMLButtonElement>('button')!.click();
    await settle();
    const req = http.expectOne('/api/goals');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ topicId: 't2' });
    req.flush({ active: goal({ topicId: 't2' }), effectiveFrom: '2026-09-30', startsTomorrow: false });
    await settle();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/today');
  });

  it('says when the new topic starts tomorrow', async () => {
    await setup({ active: null, others: [] });
    await chooseLevel('A1');
    topicRow('Gia đình').querySelector<HTMLButtonElement>('button')!.click();
    await settle();
    http.expectOne('/api/goals').flush({ active: goal(), effectiveFrom: '2026-10-01', startsTomorrow: true });
    await settle();
    expect(text(el.querySelector('[role="status"]'))).toContain('Chủ đề mới bắt đầu từ ngày mai');
    expect(router.navigateByUrl).not.toHaveBeenCalled();
    expect(el.querySelector('a[href="/today"]')).toBeTruthy();
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
