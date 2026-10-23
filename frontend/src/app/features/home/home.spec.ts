import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Dashboard, Stats } from '../../core/models/dashboard';
import { GoalView, PublicTopic } from '../../core/models/study';
import { User } from '../../core/models/user';
import { AuthService } from '../../core/services/auth.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { Home, nextTopic, weekSummary } from './home';

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
  steps: { vocab: 'done', listen: 'current', read: 'locked', grammar: 'locked', speak: 'locked', write: 'locked' },
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
  steps: { vocab: 'locked', listen: 'locked', read: 'locked', grammar: 'locked', speak: 'locked', write: 'locked' },
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

const weekFigures = stats({
  period: 'week',
  cards: 10,
  lessons: { read: 1, listen: 1, write: 1, completed: 1 },
  writing: { submitted: 1, averageScore: 3.5 },
});

describe('Home', () => {
  let fixture: ComponentFixture<Home>;
  let controller: HttpTestingController;
  let el: HTMLElement;
  const unseen = signal(0);

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const link = (label: string) => Array.from(el.querySelectorAll('a')).find((a) => text(a) === label) ?? null;

  /** This week's figures (the "Tuần này" card); null makes the request fail. */
  const flushWeek = (figures: Stats | null = weekFigures) => {
    const req = controller.expectOne((r) => r.url === '/stats' && r.params.get('period') === 'week');
    if (figures) {
      req.flush(figures);
    } else {
      req.flush('down', { status: 500, statusText: 'Error' });
    }
  };

  /** The cards due now (the page asks for one card and reads the total). */
  const flushDue = (total = 0) =>
    controller
      .expectOne((r) => r.url === '/vocab/review/due')
      .flush({ cards: [], total, nextDue: null });

  const open = async (data: Dashboard, figures: Stats = stats(), due = 0) => {
    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/dashboard').flush(data);
    controller.expectOne('/stats').flush(figures);
    flushWeek();
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
    expect(text(el.querySelector('.subtitle'))).toBe(
      'Bài này còn 5 bước nữa thôi (Nghe, Đọc, Ngữ pháp, Nói, Viết)!',
    );
    // One card for the lesson being studied, with its progress (no second card repeating it).
    expect(el.querySelector('.today-goal')).toBeNull();
    // The six steps of the lesson's page (Từ vựng, Nghe, Đọc, Ngữ pháp, Nói, Viết).
    expect(text(el.querySelector('.lesson .steps-bar .label'))).toBe('Bước 2/6: Nghe');
    expect(text(el.querySelector('.lesson .steps-bar .count'))).toBe('1/6 bước');
    expect(text(el.querySelector('.lesson-title'))).toBe('At the café');
    expect(text(el.querySelector('.lesson'))).toContain('Chủ đề: A1 · Gia đình · Bài 3/12');
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
    controller.expectOne('/dashboard').flush(studying());
    controller.expectOne('/stats').flush('down', { status: 500, statusText: 'Error' });
    flushWeek();
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
    await open(studying({
        action: { kind: 'start', step: 'read' },
        steps: { vocab: 'current', listen: 'locked', read: 'locked', grammar: 'locked', speak: 'locked', write: 'locked' },
        tomorrowCards: 0,
      }));
    expect(link('Bắt đầu học')?.getAttribute('href')).toBe('/lessons/l1');
    expect(text(el.querySelector('.tomorrow'))).toBe('Ngày mai: không có thẻ cần ôn');
  });

  it('goes on from the step saved on this device, as the lesson page does', async () => {
    // Đọc on screen on this device: the server knows no step done yet (Từ vựng, Nghe left for later).
    localStorage.setItem('luna.lesson-step.l1', 'read');
    try {
      await open(studying({ action: { kind: 'start', step: 'vocab' }, steps: { vocab: 'current', listen: 'locked', read: 'locked', grammar: 'locked', speak: 'locked', write: 'locked' } }));
      expect(link('Tiếp tục: Đọc')?.getAttribute('href')).toBe('/lessons/l1');
      expect(text(el.querySelector('.lesson .steps-bar .label'))).toBe('Bước 3/6: Đọc');
      expect(text(el.querySelector('.subtitle'))).toBe(
        'Bài này còn 6 bước nữa thôi (Từ vựng, Nghe, Đọc, Ngữ pháp, Nói, Viết)!',
      );
    } finally {
      localStorage.removeItem('luna.lesson-step.l1');
    }
  });

  it('never goes back behind a step the server knows is done', async () => {
    localStorage.setItem('luna.lesson-step.l1', 'vocab');
    try {
      await open(studying());
      expect(link('Tiếp tục: Nghe')).not.toBeNull();
    } finally {
      localStorage.removeItem('luna.lesson-step.l1');
    }
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

    controller.expectOne('/dashboard').flush('down', { status: 500, statusText: 'Error' });
    controller.expectOne('/stats').flush(stats());
    flushWeek();
    flushDue();
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"] p'))).toBe('Không tải được màn hình chính.');

    (el.querySelector('[role="alert"] button') as HTMLButtonElement).click();
    controller.expectOne('/dashboard').flush(studying());
    controller.expectOne('/stats').flush(stats());
    flushWeek();
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

  it('names the finished topic and offers the next one, which it makes the goal', async () => {
    const done = studying({ kind: 'noNewLesson', lesson: null, action: null, goalCompleted: true });
    await open(done);
    controller.expectOne((r) => r.url === '/topics' && r.params.get('level') === 'A1').flush({ topics: [topic('t1'), topic('t2', 'Âm nhạc')] });
    controller.expectOne('/goals').flush({ active: done.goal, others: [] });
    await fixture.whenStable();
    const card = el.querySelector('[role="status"]');
    expect(text(card)).toContain('Bạn đã học xong chủ đề A1 · Gia đình');
    expect(text(card)).not.toContain('lộ trình');
    expect(text(card?.querySelector('.next-name'))).toBe('Âm nhạc');
    expect(card?.querySelector('a.summary-link')?.getAttribute('href')).toBe('/topics/t1/summary?level=A1');

    (Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Học chủ đề này') as HTMLButtonElement).click();
    const post = controller.expectOne('/goals');
    expect(post.request.method).toBe('POST');
    expect(post.request.body).toEqual({ topicId: 't2', level: 'A1' });
    post.flush({ active: done.goal });
    await fixture.whenStable();
    controller.expectOne('/dashboard').flush(studying());
    controller.expectOne('/stats').flush(stats());
    flushWeek();
    flushDue();
    await fixture.whenStable();
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  it('offers the topic list once the finished topic has no next one', async () => {
    const done = studying({ kind: 'noNewLesson', lesson: null, action: null, goalCompleted: true });
    await open(done);
    controller.expectOne((r) => r.url === '/topics').flush({ topics: [topic('t1')] });
    controller.expectOne('/goals').flush({ active: done.goal, others: [] });
    await fixture.whenStable();
    expect(el.querySelector('.next-topic')).toBeNull();
    expect(link('Chọn chủ đề khác')?.getAttribute('href')).toBe('/goal');
  });

  it('invites to start a lesson that has no part done yet', async () => {
    await open(
      studying({
        steps: { vocab: 'current', listen: 'locked', read: 'locked', grammar: 'locked', speak: 'locked', write: 'locked' },
        currentStep: 'read',
        action: { kind: 'start', step: 'read' },
      }),
    );
    expect(text(el.querySelector('.subtitle'))).toBe('Sẵn sàng cho bài mới chưa? Bắt đầu thôi!');
  });

  it('loads fresh numbers every time the page opens', async () => {
    await open(studying());
    fixture.destroy();
    await open(studying({ action: { kind: 'continue', step: 'listen' } }), stats({ reading: { answered: 10, correct: 5, rate: 0.5 } }));
    expect(text(el.querySelector('.accuracy .tile-value'))).toBe('65%');
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  it('loads again when the app comes back to the screen, dropping a lesson deleted meanwhile', async () => {
    await open(studying());
    const later = { id: 'l2', title: 'At the park', topicName: 'Gia đình', level: 'A1' as const };
    document.dispatchEvent(new Event('visibilitychange'));
    controller.expectOne('/dashboard').flush(studying({ lesson: later }));
    controller.expectOne('/stats').flush(stats());
    flushWeek();
    flushDue();
    await fixture.whenStable();
    expect(text(el.querySelector('.lesson-title'))).toBe('At the park');
    expect(link('Tiếp tục: Nghe')?.getAttribute('href')).toBe('/lessons/l2');
    fixture.destroy();
    // A page that is gone asks for nothing.
    document.dispatchEvent(new Event('visibilitychange'));
  });

  it('puts the cards due today first, with a way to review them', async () => {
    await open(studying(), stats(), 12);
    // One card for reviewing, with tomorrow's cards on its second line.
    expect(el.querySelectorAll('.review')).toHaveLength(1);
    const card = el.querySelector('.review')!;
    expect(text(card.querySelector('.review-title'))).toBe('Ôn 12 thẻ · khoảng 4 phút');
    expect(text(card.querySelector('.tomorrow'))).toBe('Ngày mai: 17 thẻ cần ôn');
    expect(card.getAttribute('href')).toBe('/vocabulary/review');
  });

  it('sums up the week and links to the stats', async () => {
    await open(studying());
    const card = el.querySelector('.week')!;
    expect(text(card.querySelector('.review-title'))).toBe('Tuần này');
    expect(text(card.querySelector('.muted'))).toBe('1 bài · 10 từ · 1 bài viết (3,5/5)');
    expect(card.getAttribute('href')).toBe('/stats');
  });

  it('hides the week card when its figures fail', async () => {
    fixture = TestBed.createComponent(Home);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/dashboard').flush(studying());
    controller.expectOne('/stats').flush(stats());
    flushWeek(null);
    flushDue();
    await fixture.whenStable();
    expect(el.querySelector('.week')).toBeNull();
    expect(link('Tiếp tục: Nghe')).not.toBeNull();
  });

  it('says nothing is left for today when nothing is due', async () => {
    await open(studying());
    expect(text(el.querySelector('.review .review-title'))).toBe('Hôm nay không còn thẻ cần ôn');
  });
});

const topic = (id: string, name = 'Gia đình', lessonCount = 3): PublicTopic => ({
  id,
  name,
  level: 'A1',
  description: '',
  lessonCount,
  position: 0,
});

describe('weekSummary', () => {
  const none = { read: 0, listen: 0, write: 0, completed: 0 };

  it('leaves out what is zero and says when the week is empty', () => {
    expect(weekSummary(weekFigures)).toBe('1 bài · 10 từ · 1 bài viết (3,5/5)');
    expect(weekSummary(stats({ cards: 4, lessons: none, writing: { submitted: 0, averageScore: null } }))).toBe('4 từ');
    expect(weekSummary(stats({ cards: 0, lessons: none, writing: { submitted: 0, averageScore: null } }))).toBe(
      'Chưa học gì tuần này',
    );
  });
});

describe('nextTopic', () => {
  const goal = (topicId: string, completedLessons = 1, totalLessons = 1): GoalView => ({
    topicId,
    topicName: '',
    level: 'A1',
    completedLessons,
    totalLessons,
    status: 'paused',
    effectiveFrom: '2026-10-01',
  });
  const topics = [topic('a'), topic('b'), topic('c', 'C', 0), topic('d')];

  it('takes the next topic in the suggested order that has lessons and is not finished', () => {
    expect(nextTopic(goal('a'), topics, { active: goal('a'), others: [goal('b')] })?.id).toBe('d');
  });

  it('goes back to an earlier topic when none is left after the goal', () => {
    expect(nextTopic(goal('d'), topics, { active: goal('d'), others: [goal('b', 0, 2)] })?.id).toBe('a');
  });

  it('gives null when every topic is finished', () => {
    expect(nextTopic(goal('a'), topics, { active: goal('a'), others: [goal('b'), goal('d')] })).toBeNull();
  });
});
