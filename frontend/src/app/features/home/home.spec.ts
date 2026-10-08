import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Dashboard, Stats } from '../../core/models/dashboard';
import { User } from '../../core/models/user';
import { AuthService } from '../../core/services/auth.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
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
  steps: { read: 'done', listen: 'current', write: 'locked' },
  currentStep: 'listen',
  action: { kind: 'continue', step: 'listen' },
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
  steps: { read: 'locked', listen: 'locked', write: 'locked' },
  currentStep: '',
  action: null,
  streak: 1,
  tomorrowCards: 0,
};

const stats = (over: Partial<Stats> = {}): Stats => ({
  cards: 25,
  dictation: { sentences: 40, correctWords: 82, totalWords: 100, rate: 0.8 },
  lessons: { read: 6, listen: 5, write: 4, completed: 4 },
  reading: { answered: 10, correct: 9, rate: 0.9 },
  writing: { submitted: 4, averageScore: 3.8 },
  ...over,
});

describe('Home', () => {
  let fixture: ComponentFixture<Home>;
  let controller: HttpTestingController;
  let el: HTMLElement;
  const unseen = signal(0);

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const link = (label: string) => Array.from(el.querySelectorAll('a')).find((a) => text(a) === label) ?? null;

  /** The cards due now (the page asks for one card and reads the total). */
  const flushDue = (total = 0) =>
    controller
      .expectOne((r) => r.url === '/api/vocab/review/due')
      .flush({ cards: [], total, nextDue: null });

  const open = async (data: Dashboard, figures: Stats = stats(), due = 0) => {
    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/api/dashboard').flush(data);
    controller.expectOne('/api/stats').flush(figures);
    flushDue(due);
    await fixture.whenStable();
  };

  beforeEach(async () => {
    unseen.set(0);
    const user: User = { id: '1', email: 'minh@example.com', role: 'member', timezone: 'Asia/Ho_Chi_Minh' };
    await TestBed.configureTestingModule({
      imports: [Home],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: { currentUser: signal(user) } },
        { provide: WritingNotifier, useValue: { unseen } },
      ],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  // --- US1 ---

  it('greets the learner and shows the progress of the lesson being studied and the button into it', async () => {
    await open(studying());
    expect(text(el.querySelector('h1'))).toBe('Xin chào, minh 👋');
    expect(text(el.querySelector('.subtitle'))).toBe('Bài này còn 2 phần nữa thôi (Đọc, Nghe, Viết)!');
    // One card for the lesson being studied, with its progress (no second card repeating it).
    expect(el.querySelector('.today-goal')).toBeNull();
    expect(text(el.querySelector('.lesson .steps-bar .label'))).toBe('Đọc · Nghe · Viết');
    expect(text(el.querySelector('.lesson .steps-bar .count'))).toBe('1/3 phần');
    expect(text(el.querySelector('.lesson-title'))).toBe('At the café');
    expect(text(el.querySelector('.lesson'))).toContain('Chủ đề: A1 · Gia đình');
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: 17 thẻ cần ôn');
    // No "today" page: the button opens the lesson itself.
    expect(link('Tiếp tục: Nghe')?.getAttribute('href')).toBe('/lessons/l1');
  });

  it('shows the streak, the words learned and the accuracy', async () => {
    await open(studying());
    expect(text(el.querySelector('.streak .tile-value'))).toBe('4');
    expect(text(el.querySelector('.words .tile-value'))).toBe('25');
    // Mean of comprehension (90%) and dictation (80%).
    expect(text(el.querySelector('.accuracy .tile-value'))).toBe('85%');
  });

  it('shows "—" for accuracy before anything was measured, and when the figures fail', async () => {
    await open(
      studying(),
      stats({
        dictation: { sentences: 0, correctWords: 0, totalWords: 0, rate: null },
        reading: { answered: 0, correct: 0, rate: null },
      }),
    );
    expect(text(el.querySelector('.accuracy .tile-value'))).toBe('—');
    fixture.destroy();

    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/api/dashboard').flush(studying());
    controller.expectOne('/api/stats').flush('down', { status: 500, statusText: 'Error' });
    flushDue();
    await fixture.whenStable();
    expect(text(el.querySelector('.words .tile-value'))).toBe('—');
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  it('links the bell to the writings, with the number of new results', async () => {
    unseen.set(2);
    await open(studying());
    const bell = el.querySelector('.bell')!;
    expect(bell.getAttribute('href')).toBe('/writings');
    expect(bell.getAttribute('aria-label')).toBe('Bài viết, 2 kết quả mới');
    expect(text(bell)).toBe('2');
  });

  it('says "Bắt đầu học" before any step is done', async () => {
    await open(studying({ action: { kind: 'start', step: 'read' }, tomorrowCards: 0 }));
    expect(link('Bắt đầu học')?.getAttribute('href')).toBe('/lessons/l1');
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
    controller.expectOne('/api/stats').flush(stats());
    flushDue();
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"] p'))).toBe('Không tải được màn hình chính.');

    (el.querySelector('[role="alert"] button') as HTMLButtonElement).click();
    controller.expectOne('/api/dashboard').flush(studying());
    controller.expectOne('/api/stats').flush(stats());
    flushDue();
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')).toBeNull();
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  // --- US2 ---

  it('invites a learner without a goal to choose a topic', async () => {
    await open(noGoal);
    expect(text(el.querySelector('[role="status"] .card-title'))).toBe('Bạn chưa chọn mục tiêu');
    expect(link('Chọn chủ đề')?.getAttribute('href')).toBe('/goal');
    expect(el.querySelector('.today-goal')).toBeNull();
    expect(el.querySelector('.lesson')).toBeNull();
    expect(el.querySelector('.course')).toBeNull();
    expect(text(el.querySelector('.streak .tile-value'))).toBe('1');
  });

  it('says there is no new lesson and offers free review or another topic', async () => {
    await open(studying({ kind: 'noNewLesson', lesson: null, action: null, currentStep: '' }));
    expect(el.querySelector('.today-goal')).toBeNull();
    expect(text(el.querySelector('[role="status"]'))).toContain('Chưa có bài mới');
    expect(text(el.querySelector('[role="status"]'))).not.toContain('Chúc mừng');
    expect(link('Ôn tự do')?.getAttribute('href')).toBe('/vocabulary/review');
    expect(link('Chọn chủ đề khác')?.getAttribute('href')).toBe('/goal');
  });

  it('tells a guest the next lessons are for members', async () => {
    await open(studying({ kind: 'membersOnly', lesson: null, action: null, currentStep: '' }));
    const card = el.querySelector('.members-only')!;
    expect(text(card)).toContain('Bài tiếp theo chỉ dành cho thành viên');
    expect(text(card)).not.toContain('Chưa có bài mới');
    expect(link('Học thử chủ đề khác')?.getAttribute('href')).toBe('/goal');
  });

  it('congratulates when the roadmap is finished', async () => {
    await open(studying({ kind: 'noNewLesson', lesson: null, action: null, goalCompleted: true }));
    expect(text(el.querySelector('[role="status"]'))).toContain('Chúc mừng, bạn đã học hết lộ trình!');
  });

  // --- US3 ---

  it('shows the topic being studied with the read, listen and write bars', async () => {
    await open(studying());
    const course = el.querySelector('.course-link')!;
    expect(course.getAttribute('href')).toBe('/lessons');
    expect(text(course.querySelector('.course-name'))).toBe('A1 · Gia đình');
    expect(text(course)).toContain('Đã học 2/12 bài');
    // Accuracy, not lessons done: reading 90%, dictation 80%, writing 3.8 out of 5.
    const bars = Array.from(el.querySelectorAll('.skills lu-progress-bar'));
    expect(bars.map((b) => text(b.querySelector('.label')))).toEqual(['Đọc', 'Nghe', 'Viết']);
    expect(bars.map((b) => text(b.querySelector('.count')))).toEqual(['90%', '80%', '76%']);
    expect(bars.map((b) => b.getAttribute('data-tone'))).toEqual(['read', 'listen', 'write']);
    expect(text(el)).not.toContain('Nói');
  });

  it('says "Chưa có" for a skill that was not measured yet', async () => {
    await open(
      studying(),
      stats({
        reading: { answered: 0, correct: 0, rate: null },
        writing: { submitted: 0, averageScore: null },
      }),
    );
    const empty = Array.from(el.querySelectorAll('.skills .skill-empty'));
    expect(empty.map((e) => text(e.children[0]))).toEqual(['Đọc', 'Viết']);
    expect(empty.map((e) => text(e.children[1]))).toEqual(['Chưa có', 'Chưa có']);
    expect(el.querySelectorAll('.skills lu-progress-bar').length).toBe(1);
  });

  it('loads fresh numbers every time the page opens', async () => {
    await open(studying());
    fixture.destroy();
    await open(studying({ action: { kind: 'continue', step: 'listen' } }), stats({ reading: { answered: 10, correct: 5, rate: 0.5 } }));
    expect(text(el.querySelector('.skills lu-progress-bar .count'))).toBe('50%');
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  it('puts the cards due today first, with a way to review them', async () => {
    await open(studying(), stats(), 12);
    const card = el.querySelector('.due-today')!;
    expect(text(card)).toContain('Hôm nay: 12 thẻ cần ôn');
    expect(text(card)).toContain('khoảng 4 phút');
    expect(card.getAttribute('href')).toBe('/vocabulary/review');
    // Before the lesson, and the tomorrow line stays.
    expect(card.compareDocumentPosition(el.querySelector('.continue')!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: 17 thẻ cần ôn');
  });

  it('shows no card for today when nothing is due', async () => {
    await open(studying());
    expect(el.querySelector('.due-today')).toBeNull();
  });
});
