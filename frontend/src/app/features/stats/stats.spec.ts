import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Dashboard, Stats as StatsData } from '../../core/models/dashboard';
import { Stats } from './stats';

const allTime: StatsData = {
  period: 'all',
  cards: 120,
  dictation: { sentences: 40, lessons: 5, correctWords: 328, totalWords: 400, rate: 0.82 },
  lessons: { read: 12, listen: 5, write: 3, completed: 4 },
  reading: { answered: 12, correct: 9, rate: 0.75 },
  writing: { submitted: 3, averageScore: 3.75 },
  grammar: { lessons: 8 },
};

const week: StatsData = {
  period: 'week',
  cards: 14,
  dictation: { sentences: 6, lessons: 2, correctWords: 30, totalWords: 40, rate: 0.75 },
  lessons: { read: 2, listen: 2, write: 1, completed: 1 },
  reading: { answered: 0, correct: 0, rate: null },
  writing: { submitted: 1, averageScore: null },
  grammar: { lessons: 1 },
};

const dashboard = (over: Partial<Dashboard> = {}): Dashboard => ({
  kind: 'studying',
  goal: {
    topicId: 't1',
    topicName: 'Gia đình',
    level: 'A1',
    completedLessons: 7,
    totalLessons: 20,
    status: 'active',
    effectiveFrom: '2026-09-01',
  },
  goalCompleted: false,
  skills: null,
  lesson: null,
  steps: { read: 'done', listen: 'current', write: 'locked' },
  currentStep: 'listen',
  action: null,
  streak: 9,
  tomorrowCards: 0,
  ...over,
});

describe('Stats', () => {
  let fixture: ComponentFixture<Stats>;
  let controller: HttpTestingController;
  let el: HTMLElement;

  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const tab = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('[role="tab"]')).find((b) => text(b) === label)!;
  const counts = () =>
    Array.from(el.querySelectorAll('.count')).map((c) =>
      Array.from(c.children).map((part) => text(part)).filter(Boolean).join(' '),
    );

  /** Opens the page: every-day figures and the dashboard, then this week's figures. */
  const open = async (weekly: StatsData = week, dash: Dashboard = dashboard()) => {
    fixture = TestBed.createComponent(Stats);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne((r) => r.url === '/stats' && !r.params.has('period')).flush(allTime);
    controller.expectOne('/dashboard').flush(dash);
    await fixture.whenStable();
    controller.expectOne((r) => r.url === '/stats' && r.params.get('period') === 'week').flush(weekly);
    await fixture.whenStable();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Stats],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  it("opens on this week's figures by skill, next to the roadmap ring", async () => {
    await open();
    expect(text(el.querySelector('h1'))).toBe('Lộ trình & Tiến độ');
    expect(tab('Tuần').getAttribute('aria-selected')).toBe('true');
    expect(counts()).toEqual([
      'Từ vựng 14 từ',
      'Ngữ pháp 1 bài',
      'Luyện nghe 2 bài',
      'Luyện nói —',
      'Luyện đọc 2 bài',
      'Luyện viết 1 bài',
    ]);
    expect(el.querySelector('lu-progress-ring')?.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('35');
    expect(text(el.querySelector('.goal'))).toContain('7/20 bài · A1 Gia đình');
    expect(text(el.querySelector('#skills-heading'))).toBe('Độ chính xác tuần này');
  });

  it('asks for the month, and shows every day without asking again', async () => {
    await open();
    tab('Tháng').click();
    await fixture.whenStable();
    controller
      .expectOne((r) => r.url === '/stats' && r.params.get('period') === 'month')
      .flush({ ...week, period: 'month', cards: 50 });
    await fixture.whenStable();
    expect(counts()[0]).toBe('Từ vựng 50 từ');

    tab('Tổng').click();
    await fixture.whenStable();
    controller.expectNone('/stats');
    expect(counts()[0]).toBe('Từ vựng 120 từ');
    expect(text(el.querySelector('#skills-heading'))).toBe('Độ chính xác từ trước tới nay');
  });

  it('moves between the periods with the arrow keys', async () => {
    await open();
    tab('Tuần').dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    await fixture.whenStable();
    expect(tab('Tổng').getAttribute('aria-selected')).toBe('true');
    expect(document.activeElement === tab('Tổng') || !el.isConnected).toBe(true);
  });

  it('shows the badges earned from every-day figures and the streak', async () => {
    await open();
    expect(text(el.querySelector('.badges-head .muted'))).toBe('Đã đạt 3/6');
    const earned = Array.from(el.querySelectorAll('.badge.earned .badge-label')).map((b) => text(b));
    expect(earned).toEqual(['Học đều', 'Từ vựng', 'Luyện đọc']);
    const listen = Array.from(el.querySelectorAll('.badge')).find((b) => text(b).startsWith('Luyện nghe'))!;
    expect(text(listen)).toContain('Chưa đạt: Xong bước Nghe 10 bài (5/10)');
  });

  it('says when no roadmap is chosen, and "—" for figures an older server leaves out', async () => {
    const noGrammar: StatsData = { ...week };
    delete noGrammar.grammar;
    await open({ ...noGrammar, dictation: { ...week.dictation, lessons: undefined } }, dashboard({ goal: null }));
    expect(text(el.querySelector('.goal'))).toContain('Chưa chọn lộ trình');
    expect(counts()[1]).toBe('Ngữ pháp —');
    expect(counts()[2]).toBe('Luyện nghe 2 bài');
  });

  it('offers a retry when loading fails', async () => {
    fixture = TestBed.createComponent(Stats);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    controller.expectOne('/stats').flush('down', { status: 500, statusText: 'Error' });
    controller.match('/dashboard');
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được thống kê.');
    el.querySelector<HTMLButtonElement>('[role="alert"] button')!.click();
    await fixture.whenStable();
    controller.expectOne('/stats').flush(allTime);
    controller.expectOne('/dashboard').flush(dashboard());
    await fixture.whenStable();
    controller.expectOne((r) => r.params.get('period') === 'week').flush(week);
    await fixture.whenStable();
    expect(counts()).toHaveLength(6);
  });
});
