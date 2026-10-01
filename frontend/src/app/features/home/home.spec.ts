import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Dashboard } from '../../core/models/dashboard';
import { Home } from './home';

const studying = (over: Partial<Dashboard> = {}): Dashboard => ({
  kind: 'studying',
  goal: {
    topicId: 't1',
    topicName: 'Gia đình',
    level: 'A1',
    completedLessons: 2,
    totalLessons: 12,
    status: 'active',
    effectiveFrom: '2026-09-30',
  },
  goalCompleted: false,
  skills: { read: 3, listen: 2, write: 1, total: 12 },
  lesson: { id: 'l1', title: 'At the café', topicName: 'Gia đình', level: 'A1' },
  steps: { review: 'done', read: 'current', listen: 'locked', write: 'locked' },
  currentStep: 'read',
  action: { kind: 'continue', step: 'read' },
  streak: 4,
  tomorrowCards: 17,
  ...over,
});

const noGoal: Dashboard = {
  ...studying(),
  kind: 'noGoal',
  goal: null,
  skills: null,
  lesson: null,
  steps: { review: 'locked', read: 'locked', listen: 'locked', write: 'locked' },
  currentStep: '',
  action: null,
  streak: 1,
  tomorrowCards: 0,
};

describe('Home', () => {
  let fixture: ComponentFixture<Home>;
  let controller: HttpTestingController;
  let el: HTMLElement;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const link = (label: string) =>
    Array.from(el.querySelectorAll('a')).find((a) => text(a).replace(' ▶', '') === label) ?? null;

  const open = async (data: Dashboard) => {
    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/api/dashboard').flush(data);
    await fixture.whenStable();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Home],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  // --- US1 ---

  it('shows the goal, today’s lesson, the steps and the button to the current step', async () => {
    await open(studying());
    const goal = el.querySelector('.goal-bar')!;
    expect(text(goal.querySelector('.label'))).toBe('A1 · Gia đình');
    expect(text(goal.querySelector('.count'))).toBe('2/12 bài');
    expect(link('Đổi chủ đề')?.getAttribute('href')).toBe('/goal');
    expect(text(el.querySelector('.lesson-title'))).toBe('At the café');
    expect(text(el.querySelector('.lesson'))).toContain('A1 · Gia đình');
    expect(el.querySelector('.lesson lu-step-indicator')).not.toBeNull();
    expect(text(el.querySelector('.streak'))).toBe('🔥 4 ngày');
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: 17 thẻ cần ôn');
    expect(link('Tiếp tục: Đọc')?.getAttribute('href')).toBe('/today');
    expect(link('Xem thống kê')?.getAttribute('href')).toBe('/stats');
  });

  it('says "Bắt đầu" before any step is done', async () => {
    await open(studying({ action: { kind: 'start', step: 'review' } }));
    expect(link('Bắt đầu: Ôn')?.getAttribute('href')).toBe('/today');
  });

  it('names the read step when there is no card to review', async () => {
    await open(studying({ action: { kind: 'start', step: 'read' }, tomorrowCards: 0 }));
    expect(link('Bắt đầu: Đọc')).not.toBeNull();
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: không có thẻ cần ôn');
  });

  it('names a deleted lesson "Bài học"', async () => {
    await open(studying({ lesson: { id: 'l1', title: '', topicName: 'Gia đình', level: 'A1' } }));
    expect(text(el.querySelector('.lesson-title'))).toBe('Bài học');
  });

  it('keeps a placeholder while loading and offers a retry on error', async () => {
    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    expect(el.querySelector('.placeholder[aria-busy="true"]')).not.toBeNull();

    controller.expectOne('/api/dashboard').flush('down', { status: 500, statusText: 'Error' });
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"] p'))).toBe('Không tải được màn hình chính.');

    (el.querySelector('[role="alert"] button') as HTMLButtonElement).click();
    controller.expectOne('/api/dashboard').flush(studying());
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')).toBeNull();
    expect(link('Tiếp tục: Đọc')).not.toBeNull();
  });

  // --- US2 ---

  it('invites a learner without a goal to choose a topic', async () => {
    await open(noGoal);
    expect(text(el.querySelector('[role="status"] .card-title'))).toBe('Bạn chưa chọn mục tiêu');
    expect(link('Chọn chủ đề')?.getAttribute('href')).toBe('/goal');
    expect(el.querySelector('.goal')).toBeNull();
    expect(el.querySelector('.lesson')).toBeNull();
    expect(text(el.querySelector('.streak'))).toBe('🔥 1 ngày');
  });

  it('says see you tomorrow once today’s lesson is done', async () => {
    await open(
      studying({
        kind: 'doneToday',
        steps: { review: 'done', read: 'done', listen: 'done', write: 'done' },
        currentStep: 'done',
        action: null,
        tomorrowCards: 9,
      }),
    );
    expect(text(el.querySelector('[role="status"] .card-title'))).toBe('Đã xong bài hôm nay, hẹn bạn ngày mai');
    expect(link('Ôn tự do')?.getAttribute('href')).toBe('/vocabulary/review');
    expect(el.querySelectorAll('.lesson li[data-state="done"]').length).toBe(4);
    expect(link('Tiếp tục: Đọc')).toBeNull();
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: 9 thẻ cần ôn');
  });

  it('says there is no new lesson and offers free review or another topic', async () => {
    await open(studying({ kind: 'noNewLesson', lesson: null, action: null, currentStep: '' }));
    expect(text(el.querySelector('[role="status"]'))).toContain('Chưa có bài mới');
    expect(text(el.querySelector('[role="status"]'))).not.toContain('Chúc mừng');
    expect(link('Ôn tự do')?.getAttribute('href')).toBe('/vocabulary/review');
    expect(link('Chọn chủ đề khác')?.getAttribute('href')).toBe('/goal');
  });

  it('congratulates when the roadmap is finished', async () => {
    await open(studying({ kind: 'noNewLesson', lesson: null, action: null, goalCompleted: true }));
    expect(text(el.querySelector('[role="status"]'))).toContain('Chúc mừng, bạn đã học hết lộ trình!');
  });

  // --- US3 ---

  it('shows the read, listen and write bars', async () => {
    await open(studying());
    const bars = Array.from(el.querySelectorAll('.skills lu-progress-bar'));
    expect(bars.map((b) => text(b.querySelector('.label')))).toEqual(['Đọc', 'Nghe', 'Viết']);
    expect(bars.map((b) => text(b.querySelector('.count')))).toEqual(['3/12', '2/12', '1/12']);
    expect(bars.map((b) => b.getAttribute('data-tone'))).toEqual(['read', 'listen', 'write']);
    expect(text(el)).not.toContain('Nói');
  });

  it('loads fresh numbers every time the page opens', async () => {
    await open(studying());
    fixture.destroy();
    await open(studying({ skills: { read: 4, listen: 2, write: 1, total: 12 }, action: { kind: 'continue', step: 'listen' } }));
    expect(text(el.querySelector('.skills lu-progress-bar .count'))).toBe('4/12');
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });
});
