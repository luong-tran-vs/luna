import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GoalView, Goals, PublicTopic } from '../../../core/models/study';
import { Goal } from './goal';

const goal = (over: Partial<GoalView> = {}): GoalView => ({
  topicId: 't1', topicName: 'Gia đình', level: 'A1', completedLessons: 2, totalLessons: 12, status: 'active',
  effectiveFrom: '2026-09-30', ...over,
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

  const setup = async (goals: Goals, query: Record<string, string> = {}) => {
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

  it('asks for a level first', async () => {
    await setup({ active: null, others: [] });
    const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(radios.map((r) => text(r))).toEqual(['A1', 'A2', 'B1', 'B2', 'C1', 'C2']);
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
